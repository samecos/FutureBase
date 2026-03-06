/**
 * 半自动化建筑设计平台 - API响应时间测试脚本
 * 工具: k6
 * 用途: 验证关键API的响应时间性能
 */

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// ==================== 自定义指标 ====================
const errorRate = new Rate('errors');
const apiResponseTime = new Trend('api_response_time');
const geometryCalcTime = new Trend('geometry_calc_time');
const scriptExecTime = new Trend('script_exec_time');

// ==================== 测试配置 ====================
export const options = {
  stages: [
    { duration: '2m', target: 10 },   // 逐步增加到10用户
    { duration: '5m', target: 10 },   // 保持10用户稳定期
    { duration: '2m', target: 20 },   // 增加到20用户
    { duration: '5m', target: 20 },   // 保持20用户稳定期
    { duration: '2m', target: 0 },    // 逐步减少
  ],
  thresholds: {
    // 全局阈值
    http_req_duration: ['p(50)<200', 'p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.01'],
    errors: ['rate<0.05'],
    
    // API特定阈值
    'api_response_time': ['p(95)<300'],
    'geometry_calc_time': ['p(95)<500'],
    'script_exec_time': ['p(95)<1000'],
  },
};

// ==================== 环境配置 ====================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_VERSION = __ENV.API_VERSION || 'v1';

// ==================== 辅助函数 ====================
function getAuthHeaders() {
  return {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${__ENV.TEST_TOKEN || 'test-token'}`,
  };
}

function logResponse(response, name) {
  console.log(`${name}: status=${response.status}, time=${response.timings.duration}ms`);
}

// ==================== 测试场景 ====================
export default function () {
  
  // ========== 场景1: 文档管理API测试 ==========
  group('文档管理API', () => {
    
    // 1.1 获取文档列表
    const listStart = Date.now();
    const listRes = http.get(`${BASE_URL}/api/${API_VERSION}/documents`, {
      headers: getAuthHeaders(),
    });
    apiResponseTime.add(Date.now() - listStart);
    
    const listSuccess = check(listRes, {
      'list status is 200': (r) => r.status === 200,
      'list response time < 200ms': (r) => r.timings.duration < 200,
      'list has data': (r) => r.json('data') !== undefined,
    });
    errorRate.add(!listSuccess);
    logResponse(listRes, 'GET /documents');
    
    sleep(1);
    
    // 1.2 创建新文档
    const createPayload = JSON.stringify({
      name: `Test Document ${Date.now()}`,
      description: 'Performance test document',
      templateId: 'template-001',
    });
    
    const createStart = Date.now();
    const createRes = http.post(`${BASE_URL}/api/${API_VERSION}/documents`, createPayload, {
      headers: getAuthHeaders(),
    });
    apiResponseTime.add(Date.now() - createStart);
    
    const createSuccess = check(createRes, {
      'create status is 201': (r) => r.status === 201,
      'create response time < 400ms': (r) => r.timings.duration < 400,
      'create returns document id': (r) => r.json('id') !== undefined,
    });
    errorRate.add(!createSuccess);
    logResponse(createRes, 'POST /documents');
    
    const documentId = createSuccess ? createRes.json('id') : null;
    sleep(1);
    
    // 1.3 获取文档详情
    if (documentId) {
      const detailStart = Date.now();
      const detailRes = http.get(`${BASE_URL}/api/${API_VERSION}/documents/${documentId}`, {
        headers: getAuthHeaders(),
      });
      apiResponseTime.add(Date.now() - detailStart);
      
      const detailSuccess = check(detailRes, {
        'detail status is 200': (r) => r.status === 200,
        'detail response time < 300ms': (r) => r.timings.duration < 300,
        'detail has correct id': (r) => r.json('id') === documentId,
      });
      errorRate.add(!detailSuccess);
      logResponse(detailRes, `GET /documents/${documentId}`);
    }
    
    sleep(1);
  });
  
  // ========== 场景2: 几何计算API测试 ==========
  group('几何计算API', () => {
    
    const geometryPayload = JSON.stringify({
      operation: 'intersection',
      shapes: [
        {
          type: 'rectangle',
          x: 0,
          y: 0,
          width: 10,
          height: 10,
        },
        {
          type: 'rectangle',
          x: 5,
          y: 5,
          width: 10,
          height: 10,
        },
      ],
    });
    
    const geoStart = Date.now();
    const geoRes = http.post(`${BASE_URL}/api/${API_VERSION}/geometry/calculate`, geometryPayload, {
      headers: getAuthHeaders(),
    });
    geometryCalcTime.add(Date.now() - geoStart);
    
    const geoSuccess = check(geoRes, {
      'geometry status is 200': (r) => r.status === 200,
      'geometry response time < 500ms': (r) => r.timings.duration < 500,
      'geometry has result': (r) => r.json('result') !== undefined,
    });
    errorRate.add(!geoSuccess);
    logResponse(geoRes, 'POST /geometry/calculate');
    
    sleep(1);
    
    // 复杂几何运算测试
    const complexGeometryPayload = JSON.stringify({
      operation: 'boolean_union',
      shapes: generateComplexShapes(10),
    });
    
    const complexGeoStart = Date.now();
    const complexGeoRes = http.post(`${BASE_URL}/api/${API_VERSION}/geometry/calculate`, complexGeometryPayload, {
      headers: getAuthHeaders(),
    });
    geometryCalcTime.add(Date.now() - complexGeoStart);
    
    const complexGeoSuccess = check(complexGeoRes, {
      'complex geometry status is 200': (r) => r.status === 200,
      'complex geometry response time < 1000ms': (r) => r.timings.duration < 1000,
    });
    errorRate.add(!complexGeoSuccess);
    logResponse(complexGeoRes, 'POST /geometry/calculate (complex)');
    
    sleep(1);
  });
  
  // ========== 场景3: 脚本执行API测试 ==========
  group('脚本执行API', () => {
    
    const scriptPayload = JSON.stringify({
      script: `
import math

def create_circle_points(radius, num_points=36):
    points = []
    for i in range(num_points):
        angle = 2 * math.pi * i / num_points
        x = radius * math.cos(angle)
        y = radius * math.sin(angle)
        points.append({'x': x, 'y': y})
    return points

result = create_circle_points(10)
print(f"Created {len(result)} points")
      `,
      timeout: 5000,
    });
    
    const scriptStart = Date.now();
    const scriptRes = http.post(`${BASE_URL}/api/${API_VERSION}/scripts/execute`, scriptPayload, {
      headers: getAuthHeaders(),
    });
    scriptExecTime.add(Date.now() - scriptStart);
    
    const scriptSuccess = check(scriptRes, {
      'script status is 200': (r) => r.status === 200,
      'script response time < 1000ms': (r) => r.timings.duration < 1000,
      'script executed successfully': (r) => r.json('success') === true,
    });
    errorRate.add(!scriptSuccess);
    logResponse(scriptRes, 'POST /scripts/execute');
    
    sleep(1);
  });
  
  // ========== 场景4: 用户和权限API测试 ==========
  group('用户权限API', () => {
    
    const userStart = Date.now();
    const userRes = http.get(`${BASE_URL}/api/${API_VERSION}/users/me`, {
      headers: getAuthHeaders(),
    });
    apiResponseTime.add(Date.now() - userStart);
    
    const userSuccess = check(userRes, {
      'user status is 200': (r) => r.status === 200,
      'user response time < 200ms': (r) => r.timings.duration < 200,
      'user has id': (r) => r.json('id') !== undefined,
    });
    errorRate.add(!userSuccess);
    logResponse(userRes, 'GET /users/me');
    
    sleep(1);
  });
}

// ==================== 辅助函数 ====================
function generateComplexShapes(count) {
  const shapes = [];
  for (let i = 0; i < count; i++) {
    shapes.push({
      type: 'rectangle',
      x: i * 10,
      y: i * 5,
      width: 10 + i,
      height: 10 + i,
    });
  }
  return shapes;
}

// ==================== 测试生命周期钩子 ====================
export function setup() {
  console.log('=== API响应时间测试开始 ===');
  console.log(`目标URL: ${BASE_URL}`);
  console.log(`API版本: ${API_VERSION}`);
  
  // 验证环境连通性
  const healthRes = http.get(`${BASE_URL}/health`);
  if (healthRes.status !== 200) {
    console.error('警告: 健康检查失败，请确认服务已启动');
  } else {
    console.log('健康检查通过');
  }
  
  return { startTime: Date.now() };
}

export function teardown(data) {
  const duration = (Date.now() - data.startTime) / 1000;
  console.log('=== API响应时间测试结束 ===');
  console.log(`总执行时间: ${duration}秒`);
}
