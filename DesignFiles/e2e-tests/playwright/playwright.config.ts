/**
 * 半自动化建筑设计平台 - Playwright配置
 */

import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright测试配置
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  // 测试目录
  testDir: './',
  
  // 完全并行运行测试
  fullyParallel: true,
  
  // 禁止在CI中并行测试时重复使用文件
  forbidOnly: !!process.env.CI,
  
  // 重试次数
  retries: process.env.CI ? 2 : 0,
  
  // 并行工作进程数
  workers: process.env.CI ? 1 : undefined,
  
  // 报告器配置
  reporter: [
    ['html', { open: 'never' }],
    ['json', { outputFile: 'test-results.json' }],
    ['junit', { outputFile: 'test-results.xml' }],
    ['list'],
  ],
  
  // 共享配置
  use: {
    // 基础URL
    baseURL: process.env.BASE_URL || 'http://localhost:3000',
    
    // 收集跟踪信息
    trace: 'on-first-retry',
    
    // 截图配置
    screenshot: 'only-on-failure',
    
    // 视频录制配置
    video: 'on-first-retry',
    
    // 视口大小
    viewport: { width: 1280, height: 720 },
    
    // 动作超时
    actionTimeout: 15000,
    
    // 导航超时
    navigationTimeout: 30000,
  },

  // 项目配置
  projects: [
    // 安装项目
    {
      name: 'setup',
      testMatch: /.*\.setup\.ts/,
    },
    
    // Chrome浏览器
    {
      name: 'chromium',
      use: { 
        ...devices['Desktop Chrome'],
        storageState: 'playwright/.auth/user.json',
      },
      dependencies: ['setup'],
    },

    // Firefox浏览器
    {
      name: 'firefox',
      use: { 
        ...devices['Desktop Firefox'],
        storageState: 'playwright/.auth/user.json',
      },
      dependencies: ['setup'],
    },

    // WebKit浏览器
    {
      name: 'webkit',
      use: { 
        ...devices['Desktop Safari'],
        storageState: 'playwright/.auth/user.json',
      },
      dependencies: ['setup'],
    },

    /* 测试移动端视口 */
    // {
    //   name: 'Mobile Chrome',
    //   use: { ...devices['Pixel 5'] },
    // },
    // {
    //   name: 'Mobile Safari',
    //   use: { ...devices['iPhone 12'] },
    // },

    /* 测试品牌浏览器 */
    // {
    //   name: 'Microsoft Edge',
    //   use: { ...devices['Desktop Edge'], channel: 'msedge' },
    // },
    // {
    //   name: 'Google Chrome',
    //   use: { ...devices['Desktop Chrome'], channel: 'chrome' },
    // },
  ],

  // 本地开发服务器配置
  webServer: [
    // 前端开发服务器
    {
      command: 'npm run dev',
      url: 'http://localhost:3000',
      reuseExistingServer: !process.env.CI,
      timeout: 120 * 1000,
    },
    // 后端API服务器
    {
      command: 'cd ../backend && ./mvnw spring-boot:run',
      url: 'http://localhost:8080/health',
      reuseExistingServer: !process.env.CI,
      timeout: 180 * 1000,
    },
  ],
});
