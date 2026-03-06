import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 100 }, // Ramp up to 100 users
    { duration: '5m', target: 100 }, // Stay at 100 users
    { duration: '2m', target: 200 }, // Ramp up to 200 users
    { duration: '5m', target: 200 }, // Stay at 200 users
    { duration: '2m', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests must complete within 500ms
    http_req_failed: ['rate<0.1'],    // Error rate must be below 10%
    errors: ['rate<0.1'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8000';

export function setup() {
  // Login and get token
  const loginRes = http.post(`${BASE_URL}/api/v1/auth/login`, JSON.stringify({
    username: 'loadtestuser',
    password: 'LoadTest123',
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  const success = check(loginRes, {
    'login successful': (r) => r.status === 200,
    'token received': (r) => r.json('accessToken') !== undefined,
  });

  errorRate.add(!success);

  return {
    token: loginRes.json('accessToken'),
  };
}

export default function (data) {
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${data.token}`,
    },
  };

  // Test 1: Get current user
  {
    const res = http.get(`${BASE_URL}/api/v1/users/me`, params);
    const success = check(res, {
      'get me status is 200': (r) => r.status === 200,
      'get me response time < 500ms': (r) => r.timings.duration < 500,
    });
    errorRate.add(!success);
  }

  sleep(1);

  // Test 2: List projects
  {
    const res = http.get(`${BASE_URL}/api/v1/projects?page=0&size=20`, params);
    const success = check(res, {
      'list projects status is 200': (r) => r.status === 200,
      'list projects has content': (r) => r.json('content') !== undefined,
    });
    errorRate.add(!success);
  }

  sleep(1);

  // Test 3: Search
  {
    const res = http.get(`${BASE_URL}/api/v1/search?q=test&type=all`, params);
    const success = check(res, {
      'search status is 200': (r) => r.status === 200,
    });
    errorRate.add(!success);
  }

  sleep(2);
}

export function teardown(data) {
  // Cleanup if needed
  console.log('Load test completed');
}
