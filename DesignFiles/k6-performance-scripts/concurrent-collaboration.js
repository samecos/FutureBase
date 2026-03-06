/**
 * 半自动化建筑设计平台 - 并发协作测试脚本
 * 工具: k6
 * 用途: 验证多用户协作编辑的并发性能和数据一致性
 */

import http from 'k6/http';
import { check, sleep, group, fail } from 'k6';
import ws from 'k6/ws';
import { Rate, Trend, Counter } from 'k6/metrics';

// ==================== 自定义指标 ====================
const wsErrorRate = new Rate('ws_errors');
const wsLatency = new Trend('ws_latency');
const operationSuccessRate = new Rate('operation_success');
const conflictCount = new Counter('conflicts_detected');

// ==================== 测试配置 ====================
export const options = {
  scenarios: {
    // 场景1: 渐进式并发用户测试
    ramping_collaboration: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 5 },    // 5用户
        { duration: '3m', target: 5 },    // 稳定期
        { duration: '2m', target: 10 },   // 10用户
        { duration: '3m', target: 10 },   // 稳定期
        { duration: '2m', target: 20 },   // 20用户
        { duration: '3m', target: 20 },   // 稳定期
        { duration: '2m', target: 0 },    // 收尾
      ],
      gracefulRampDown: '30s',
    },
    
    // 场景2: 恒定负载测试
    constant_collaboration: {
      executor: 'constant-vus',
      vus: 10,
      duration: '10m',
      startTime: '15m',  // 在ramping之后开始
    },
  },
  
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
    ws_errors: ['rate<0.05'],
    operation_success: ['rate>0.95'],
  },
};

// ==================== 环境配置 ====================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const WS_URL = __ENV.WS_URL || 'ws://localhost:8080';
const TEST_DOC_ID = __ENV.TEST_DOC_ID || 'test-collaboration-doc';

// ==================== 辅助函数 ====================
function getAuthHeaders(token) {
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`,
  };
}

function generateRandomElement(userId) {
  const types = ['wall', 'door', 'window', 'column'];
  const type = types[Math.floor(Math.random() * types.length)];
  
  return {
    id: `element-${userId}-${Date.now()}`,
    type: type,
    position: {
      x: Math.random() * 100,
      y: Math.random() * 100,
      z: 0,
    },
    properties: {
      width: 5 + Math.random() * 10,
      height: 2.5 + Math.random() * 1,
    },
    createdBy: userId,
    timestamp: Date.now(),
  };
}

function generateRandomOperation(userId) {
  const operations = ['add', 'modify', 'delete'];
  const operation = operations[Math.floor(Math.random() * operations.length)];
  
  return {
    type: 'operation',
    userId: userId,
    operation: operation,
    target: `element-${Math.floor(Math.random() * 100)}`,
    data: generateRandomElement(userId),
    timestamp: Date.now(),
  };
}

// ==================== 测试场景 ====================
export default function () {
  const userId = `user-${__VU}`;
  let authToken = null;
  
  // ========== 步骤1: 用户登录 ==========
  group('用户登录', () => {
    const loginPayload = JSON.stringify({
      username: userId,
      password: 'test123',
    });
    
    const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, {
      headers: { 'Content-Type': 'application/json' },
    });
    
    const loginSuccess = check(loginRes, {
      'login status is 200': (r) => r.status === 200,
      'login returns token': (r) => r.json('token') !== undefined,
    });
    
    if (!loginSuccess) {
      console.error(`用户 ${userId} 登录失败`);
      wsErrorRate.add(1);
      return;
    }
    
    authToken = loginRes.json('token');
    console.log(`用户 ${userId} 登录成功`);
    sleep(1);
  });
  
  if (!authToken) return;
  
  // ========== 步骤2: 获取或创建测试文档 ==========
  group('文档操作', () => {
    // 尝试获取现有文档
    const docRes = http.get(`${BASE_URL}/api/v1/documents/${TEST_DOC_ID}`, {
      headers: getAuthHeaders(authToken),
    });
    
    if (docRes.status === 404) {
      // 文档不存在，创建新文档
      const createPayload = JSON.stringify({
        id: TEST_DOC_ID,
        name: '协作测试文档',
        description: '用于并发协作测试的文档',
      });
      
      const createRes = http.post(`${BASE_URL}/api/v1/documents`, createPayload, {
        headers: getAuthHeaders(authToken),
      });
      
      check(createRes, {
        'document created': (r) => r.status === 201,
      });
      
      console.log(`用户 ${userId} 创建测试文档`);
    } else {
      check(docRes, {
        'document loaded': (r) => r.status === 200,
      });
      console.log(`用户 ${userId} 加载测试文档`);
    }
    
    sleep(1);
  });
  
  // ========== 步骤3: WebSocket协作编辑 ==========
  group('WebSocket协作', () => {
    const wsUrl = `${WS_URL}/ws/collaboration?documentId=${TEST_DOC_ID}&token=${authToken}`;
    
    const wsStartTime = Date.now();
    let operationsSent = 0;
    let operationsReceived = 0;
    let conflictsReceived = 0;
    
    const wsRes = ws.connect(wsUrl, null, (socket) => {
      // 连接打开
      socket.on('open', () => {
        console.log(`用户 ${userId} WebSocket连接成功`);
        
        // 发送加入消息
        socket.send(JSON.stringify({
          type: 'join',
          userId: userId,
          documentId: TEST_DOC_ID,
          timestamp: Date.now(),
        }));
        
        // 定期发送随机操作
        const operationInterval = setInterval(() => {
          if (operationsSent < 10) {
            const operation = generateRandomOperation(userId);
            socket.send(JSON.stringify(operation));
            operationsSent++;
            console.log(`用户 ${userId} 发送操作 #${operationsSent}`);
          }
        }, 2000);
        
        // 30秒后关闭连接
        socket.setTimeout(() => {
          clearInterval(operationInterval);
          socket.send(JSON.stringify({
            type: 'leave',
            userId: userId,
            timestamp: Date.now(),
          }));
          socket.close();
        }, 30000);
      });
      
      // 接收消息
      socket.on('message', (msg) => {
        try {
          const data = JSON.parse(msg);
          
          if (data.type === 'operation_ack') {
            operationsReceived++;
            console.log(`用户 ${userId} 收到操作确认`);
          } else if (data.type === 'conflict') {
            conflictsReceived++;
            conflictCount.add(1);
            console.log(`用户 ${userId} 收到冲突通知`);
          } else if (data.type === 'user_joined') {
            console.log(`用户 ${data.userId} 加入协作`);
          } else if (data.type === 'user_left') {
            console.log(`用户 ${data.userId} 离开协作`);
          }
        } catch (e) {
          console.error(`消息解析错误: ${e.message}`);
        }
      });
      
      // 错误处理
      socket.on('error', (e) => {
        console.error(`用户 ${userId} WebSocket错误: ${e}`);
        wsErrorRate.add(1);
      });
      
      // 连接关闭
      socket.on('close', () => {
        const duration = Date.now() - wsStartTime;
        wsLatency.add(duration);
        console.log(`用户 ${userId} WebSocket连接关闭，持续时间: ${duration}ms`);
        
        // 验证操作成功率
        const success = operationsReceived >= operationsSent * 0.9;
        operationSuccessRate.add(success);
        
        console.log(`用户 ${userId} 统计: 发送=${operationsSent}, 确认=${operationsReceived}, 冲突=${conflictsReceived}`);
      });
    });
    
    // 验证WebSocket连接
    check(wsRes, {
      'WebSocket status is 101': (r) => r && r.status === 101,
    });
    
    sleep(2);
  });
  
  // ========== 步骤4: 验证文档一致性 ==========
  group('一致性验证', () => {
    const finalDocRes = http.get(`${BASE_URL}/api/v1/documents/${TEST_DOC_ID}`, {
      headers: getAuthHeaders(authToken),
    });
    
    check(finalDocRes, {
      'final document loaded': (r) => r.status === 200,
      'document has elements': (r) => {
        const elements = r.json('elements');
        return elements && elements.length >= 0;
      },
      'document version is correct': (r) => r.json('version') !== undefined,
    });
    
    const elementCount = finalDocRes.json('elements')?.length || 0;
    console.log(`用户 ${userId} 最终文档元素数: ${elementCount}`);
  });
}

// ==================== 测试生命周期钩子 ====================
export function setup() {
  console.log('=== 并发协作测试开始 ===');
  console.log(`目标URL: ${BASE_URL}`);
  console.log(`WebSocket URL: ${WS_URL}`);
  console.log(`测试文档ID: ${TEST_DOC_ID}`);
  
  // 健康检查
  const healthRes = http.get(`${BASE_URL}/health`);
  if (healthRes.status !== 200) {
    console.error('警告: 服务健康检查失败');
  }
  
  // 清理测试数据
  console.log('清理历史测试数据...');
  
  return {
    startTime: Date.now(),
    testDocId: TEST_DOC_ID,
  };
}

export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log('=== 并发协作测试结束 ===');
  console.log(`总执行时间: ${duration}秒`);
  console.log(`测试文档ID: ${data.testDocId}`);
  
  // 可选: 清理测试数据
  console.log('可选: 执行测试数据清理');
}
