/**
 * 半自动化建筑设计平台 - 负载测试脚本
 * 工具: k6
 * 用途: 验证系统在不同负载下的性能表现和稳定性
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// ==================== 自定义指标 ====================
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');
const throughput = new Counter('requests_count');
const activeUsers = new Counter('active_users');

// ==================== 测试配置 ====================
export const options = {
  scenarios: {
    // 场景1: 渐进式负载测试 - 找到性能拐点
    ramp_up_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 10 },    // 10用户
        { duration: '2m', target: 20 },    // 20用户
        { duration: '2m', target: 30 },    // 30用户
        { duration: '2m', target: 40 },    // 40用户
        { duration: '2m', target: 50 },    // 50用户
        { duration: '2m', target: 75 },    // 75用户
        { duration: '2m', target: 100 },   // 100用户
        { duration: '5m', target: 100 },   // 保持峰值
        { duration: '2m', target: 0 },     // 收尾
      ],
      gracefulRampDown: '30s',
    },
    
    // 场景2: 峰值测试 - 突发流量
    spike_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 20 },    // 正常负载
        { duration: '30s', target: 100 },  // 突发到100用户
        { duration: '3m', target: 100 },   // 保持峰值
        { duration: '30s', target: 20 },   // 恢复正常
        { duration: '2m', target: 20 },    // 观察恢复
        { duration: '1m', target: 0 },     // 收尾
      ],
      startTime: '25m',  // 在ramp_up_test之后开始
    },
    
    // 场景3: 稳定性测试 - 长时间运行
    stability_test: {
      executor: 'constant-vus',
      vus: 30,
      duration: '30m',
      startTime: '35m',
    },
  },
  
  thresholds: {
    // 响应时间阈值
    http_req_duration: ['p(50)<200', 'p(95)<500', 'p(99)<1000'],
    
    // 错误率阈值
    http_req_failed: ['rate<0.01'],
    errors: ['rate<0.05'],
    
    // 自定义阈值
    'response_time': ['p(95)<500'],
  },
  
  // 丢弃部分响应以节省内存
  discardResponseBodies: true,
};

// ==================== 环境配置 ====================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_VERSION = __ENV.API_VERSION || 'v1';

// ==================== 辅助函数 ====================
function getAuthHeaders() {
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${__ENV.TEST_TOKEN || 'load-test-token'}`,
  };
}

function logMetrics(response, name, threshold = 500) {
  const duration = response.timings.duration;
  responseTime.add(duration);
  throughput.add(1);
  
  if (duration > threshold) {
    console.warn(`SLOW: ${name} took ${duration}ms (threshold: ${threshold}ms)`);
  }
  
  return response;
}

// ==================== 测试场景 ====================
export default function () {
  activeUsers.add(1);
  
  // ========== 场景1: 文档浏览流程 ==========
  group('文档浏览', () => {
    // 1.1 获取文档列表
    const listRes = http.get(`${BASE_URL}/api/${API_VERSION}/documents?page=1&size=20`, {
      headers: getAuthHeaders(),
    });
    
    logMetrics(listRes, 'GET /documents');
    
    const listSuccess = check(listRes, {
      'list status is 200': (r) => r.status === 200,
      'list response time < 300ms': (r) => r.timings.duration < 300,
    });
    errorRate.add(!listSuccess);
    
    sleep(randomIntBetween(1, 3));
    
    // 1.2 随机查看文档详情
    const docId = randomIntBetween(1, 100);
    const detailRes = http.get(`${BASE_URL}/api/${API_VERSION}/documents/doc-${docId}`, {
      headers: getAuthHeaders(),
    });
    
    logMetrics(detailRes, `GET /documents/doc-${docId}`);
    
    const detailSuccess = check(detailRes, {
      'detail status is 200 or 404': (r) => r.status === 200 || r.status === 404,
      'detail response time < 400ms': (r) => r.timings.duration < 400,
    });
    errorRate.add(!detailSuccess);
    
    sleep(randomIntBetween(2, 5));
  });
  
  // ========== 场景2: 几何计算 ==========
  group('几何计算', () => {
    const operations = ['intersection', 'union', 'difference', 'buffer'];
    const operation = operations[randomIntBetween(0, operations.length - 1)];
    
    const complexity = randomIntBetween(1, 3);  // 1=简单, 2=中等, 3=复杂
    const shapes = generateShapes(complexity);
    
    const geoPayload = JSON.stringify({
      operation: operation,
      shapes: shapes,
      precision: 0.001,
    });
    
    const geoRes = http.post(`${BASE_URL}/api/${API_VERSION}/geometry/calculate`, geoPayload, {
      headers: getAuthHeaders(),
    });
    
    logMetrics(geoRes, `POST /geometry/calculate (${operation})`, 1000);
    
    const geoSuccess = check(geoRes, {
      'geometry status is 200': (r) => r.status === 200,
      'geometry response time < 1000ms': (r) => r.timings.duration < 1000,
    });
    errorRate.add(!geoSuccess);
    
    sleep(randomIntBetween(1, 3));
  });
  
  // ========== 场景3: 脚本执行 ==========
  group('脚本执行', () => {
    const scriptComplexity = randomIntBetween(1, 3);
    const script = generateScript(scriptComplexity);
    
    const scriptPayload = JSON.stringify({
      script: script,
      timeout: 10000,
      memoryLimit: 128 * 1024 * 1024,  // 128MB
    });
    
    const scriptRes = http.post(`${BASE_URL}/api/${API_VERSION}/scripts/execute`, scriptPayload, {
      headers: getAuthHeaders(),
    });
    
    logMetrics(scriptRes, 'POST /scripts/execute', 2000);
    
    const scriptSuccess = check(scriptRes, {
      'script status is 200': (r) => r.status === 200,
      'script response time < 2000ms': (r) => r.timings.duration < 2000,
    });
    errorRate.add(!scriptSuccess);
    
    sleep(randomIntBetween(3, 7));
  });
  
  // ========== 场景4: 用户操作 ==========
  group('用户操作', () => {
    // 4.1 获取当前用户信息
    const userRes = http.get(`${BASE_URL}/api/${API_VERSION}/users/me`, {
      headers: getAuthHeaders(),
    });
    
    logMetrics(userRes, 'GET /users/me');
    
    check(userRes, {
      'user status is 200': (r) => r.status === 200,
      'user response time < 200ms': (r) => r.timings.duration < 200,
    });
    
    sleep(randomIntBetween(1, 2));
    
    // 4.2 获取用户文档列表
    const userDocsRes = http.get(`${BASE_URL}/api/${API_VERSION}/users/me/documents`, {
      headers: getAuthHeaders(),
    });
    
    logMetrics(userDocsRes, 'GET /users/me/documents');
    
    check(userDocsRes, {
      'user docs status is 200': (r) => r.status === 200,
      'user docs response time < 300ms': (r) => r.timings.duration < 300,
    });
    
    sleep(randomIntBetween(2, 4));
  });
  
  // ========== 场景5: 搜索操作 ==========
  group('搜索操作', () => {
    const searchTerms = ['wall', 'door', 'window', 'room', 'floor', 'building'];
    const searchTerm = searchTerms[randomIntBetween(0, searchTerms.length - 1)];
    
    const searchRes = http.get(
      `${BASE_URL}/api/${API_VERSION}/search?q=${searchTerm}&type=document`,
      { headers: getAuthHeaders() }
    );
    
    logMetrics(searchRes, `GET /search?q=${searchTerm}`);
    
    check(searchRes, {
      'search status is 200': (r) => r.status === 200,
      'search response time < 500ms': (r) => r.timings.duration < 500,
    });
    
    sleep(randomIntBetween(1, 3));
  });
}

// ==================== 辅助函数 ====================
function generateShapes(complexity) {
  const shapes = [];
  const count = complexity === 1 ? 2 : complexity === 2 ? 5 : 10;
  
  for (let i = 0; i < count; i++) {
    const shapeType = randomIntBetween(0, 2);
    
    if (shapeType === 0) {
      // 矩形
      shapes.push({
        type: 'rectangle',
        x: randomIntBetween(0, 100),
        y: randomIntBetween(0, 100),
        width: randomIntBetween(5, 20),
        height: randomIntBetween(5, 20),
      });
    } else if (shapeType === 1) {
      // 圆形
      shapes.push({
        type: 'circle',
        x: randomIntBetween(0, 100),
        y: randomIntBetween(0, 100),
        radius: randomIntBetween(3, 15),
      });
    } else {
      // 多边形
      const points = [];
      const numPoints = randomIntBetween(3, 6);
      for (let j = 0; j < numPoints; j++) {
        points.push({
          x: randomIntBetween(0, 100),
          y: randomIntBetween(0, 100),
        });
      }
      shapes.push({
        type: 'polygon',
        points: points,
      });
    }
  }
  
  return shapes;
}

function generateScript(complexity) {
  if (complexity === 1) {
    return `
# 简单脚本
result = {"status": "ok", "data": [1, 2, 3]}
print("Simple script executed")
    `;
  } else if (complexity === 2) {
    return `
import math

def calculate_area(radius):
    return math.pi * radius ** 2

areas = []
for r in range(1, 100):
    areas.append(calculate_area(r))

result = {"areas": areas, "count": len(areas)}
print(f"Calculated {len(areas)} areas")
    `;
  } else {
    return `
import math
import random

def complex_calculation(n):
    results = []
    for i in range(n):
        x = random.random() * 100
        y = random.random() * 100
        dist = math.sqrt(x**2 + y**2)
        angle = math.atan2(y, x)
        results.append({"x": x, "y": y, "dist": dist, "angle": angle})
    return results

# 执行复杂计算
data = complex_calculation(1000)

# 数据分析
avg_dist = sum(d["dist"] for d in data) / len(data)
max_dist = max(d["dist"] for d in data)

result = {
    "count": len(data),
    "avg_distance": avg_dist,
    "max_distance": max_dist,
    "sample": data[:10]
}
print(f"Complex calculation completed: {len(data)} points")
    `;
  }
}

// ==================== 测试生命周期钩子 ====================
export function setup() {
  console.log('=== 负载测试开始 ===');
  console.log(`目标URL: ${BASE_URL}`);
  console.log(`API版本: ${API_VERSION}`);
  console.log('');
  console.log('测试场景:');
  console.log('1. 渐进式负载测试 (0→100用户)');
  console.log('2. 峰值测试 (突发流量)');
  console.log('3. 稳定性测试 (30用户, 30分钟)');
  console.log('');
  
  // 健康检查
  const healthRes = http.get(`${BASE_URL}/health`);
  if (healthRes.status === 200) {
    console.log('✓ 服务健康检查通过');
  } else {
    console.error('✗ 服务健康检查失败');
  }
  
  return { startTime: Date.now() };
}

export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000 / 60;  // 转换为分钟
  console.log('');
  console.log('=== 负载测试结束 ===');
  console.log(`总执行时间: ${duration.toFixed(1)}分钟`);
  console.log('');
  console.log('关键指标检查:');
  console.log('- 检查响应时间P95是否超过阈值');
  console.log('- 检查错误率是否超过1%');
  console.log('- 检查系统资源使用情况');
}
