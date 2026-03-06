/**
 * 半自动化建筑设计平台 - E2E测试
 * 工具: Playwright
 * 用途: 验证端到端协作编辑功能
 */

import { test, expect, Page } from '@playwright/test';

// ==================== 测试配置 ====================
const BASE_URL = process.env.BASE_URL || 'http://localhost:3000';
const API_URL = process.env.API_URL || 'http://localhost:8080';

// ==================== 辅助函数 ====================
async function login(page: Page, username: string, password: string = 'test123') {
  await page.goto(`${BASE_URL}/login`);
  await page.fill('[data-testid="username-input"]', username);
  await page.fill('[data-testid="password-input"]', password);
  await page.click('[data-testid="login-button"]');
  await page.waitForURL(`${BASE_URL}/dashboard`);
}

async function createDocument(page: Page, name: string) {
  await page.click('[data-testid="new-document-button"]');
  await page.fill('[data-testid="document-name-input"]', name);
  await page.click('[data-testid="create-document-button"]');
  await page.waitForSelector('[data-testid="editor-container"]');
}

async function addWallElement(page: Page, startX: number, startY: number, endX: number, endY: number) {
  // 选择墙体工具
  await page.click('[data-testid="tool-wall"]');
  
  // 在画布上绘制墙体
  const canvas = page.locator('[data-testid="design-canvas"]');
  await canvas.click({ position: { x: startX, y: startY } });
  await canvas.click({ position: { x: endX, y: endY } });
  
  // 等待元素创建
  await page.waitForTimeout(500);
}

// ==================== 测试套件 ====================
test.describe('协作编辑功能', () => {
  
  test.beforeEach(async ({ page }) => {
    // 每个测试前登录
    await login(page, 'test-user');
  });

  test('用户应该能够创建新文档', async ({ page }) => {
    // 点击新建文档按钮
    await page.click('[data-testid="new-document-button"]');
    
    // 填写文档信息
    const documentName = `测试文档-${Date.now()}`;
    await page.fill('[data-testid="document-name-input"]', documentName);
    await page.fill('[data-testid="document-description-input"]', '这是一个测试文档');
    
    // 创建文档
    await page.click('[data-testid="create-document-button"]');
    
    // 验证文档创建成功
    await expect(page).toHaveURL(/\/editor\/\w+/);
    await expect(page.locator('[data-testid="document-title"]')).toContainText(documentName);
    
    console.log('✓ 文档创建测试通过');
  });

  test('用户应该能够添加几何元素', async ({ page }) => {
    // 创建测试文档
    await createDocument(page, '几何元素测试');
    
    // 添加墙体
    await addWallElement(page, 100, 100, 300, 100);
    
    // 验证元素被添加
    const elements = page.locator('[data-testid="canvas-element"]');
    await expect(elements).toHaveCount(1);
    
    // 添加门
    await page.click('[data-testid="tool-door"]');
    const canvas = page.locator('[data-testid="design-canvas"]');
    await canvas.click({ position: { x: 200, y: 100 } });
    
    // 验证第二个元素被添加
    await expect(elements).toHaveCount(2);
    
    console.log('✓ 几何元素添加测试通过');
  });

  test('用户应该能够保存文档', async ({ page }) => {
    // 创建并编辑文档
    await createDocument(page, '保存测试文档');
    await addWallElement(page, 100, 100, 200, 100);
    
    // 保存文档
    await page.click('[data-testid="save-button"]');
    
    // 验证保存成功提示
    await expect(page.locator('[data-testid="save-success-toast"]')).toBeVisible();
    
    // 刷新页面验证数据持久化
    await page.reload();
    await page.waitForSelector('[data-testid="editor-container"]');
    
    // 验证元素仍然存在
    const elements = page.locator('[data-testid="canvas-element"]');
    await expect(elements).toHaveCount(1);
    
    console.log('✓ 文档保存测试通过');
  });

  test('用户应该能够撤销和重做操作', async ({ page }) => {
    await createDocument(page, '撤销重做测试');
    
    // 添加元素
    await addWallElement(page, 100, 100, 200, 100);
    await expect(page.locator('[data-testid="canvas-element"]')).toHaveCount(1);
    
    // 撤销
    await page.keyboard.press('Control+z');
    await expect(page.locator('[data-testid="canvas-element"]')).toHaveCount(0);
    
    // 重做
    await page.keyboard.press('Control+Shift+z');
    await expect(page.locator('[data-testid="canvas-element"]')).toHaveCount(1);
    
    console.log('✓ 撤销重做测试通过');
  });
});

test.describe('多用户协作', () => {
  
  test('两个用户应该能够同时编辑同一文档', async ({ browser }) => {
    // 创建两个浏览器上下文
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    
    const page1 = await context1.newPage();
    const page2 = await context2.newPage();
    
    try {
      // 用户1登录并创建文档
      await login(page1, 'user1');
      await createDocument(page1, '协作测试文档');
      
      // 获取文档URL
      const documentUrl = page1.url();
      const documentId = documentUrl.split('/').pop();
      
      // 用户2登录并打开同一文档
      await login(page2, 'user2');
      await page2.goto(documentUrl);
      await page2.waitForSelector('[data-testid="editor-container"]');
      
      // 用户1添加墙体
      await addWallElement(page1, 100, 100, 200, 100);
      
      // 等待同步到用户2
      await page2.waitForTimeout(1000);
      
      // 验证用户2看到用户1的操作
      const user2Elements = page2.locator('[data-testid="canvas-element"]');
      await expect(user2Elements).toHaveCount(1);
      
      // 用户2添加门
      await page2.click('[data-testid="tool-door"]');
      const canvas2 = page2.locator('[data-testid="design-canvas"]');
      await canvas2.click({ position: { x: 150, y: 100 } });
      
      // 等待同步到用户1
      await page1.waitForTimeout(1000);
      
      // 验证用户1看到用户2的操作
      const user1Elements = page1.locator('[data-testid="canvas-element"]');
      await expect(user1Elements).toHaveCount(2);
      
      console.log('✓ 多用户协作测试通过');
      
    } finally {
      await context1.close();
      await context2.close();
    }
  });

  test('应该显示协作用户的光标位置', async ({ browser }) => {
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    
    const page1 = await context1.newPage();
    const page2 = await context2.newPage();
    
    try {
      // 用户1创建文档
      await login(page1, 'user1');
      await createDocument(page1, '光标测试文档');
      const documentUrl = page1.url();
      
      // 用户2加入
      await login(page2, 'user2');
      await page2.goto(documentUrl);
      await page2.waitForSelector('[data-testid="editor-container"]');
      
      // 用户2移动鼠标
      const canvas2 = page2.locator('[data-testid="design-canvas"]');
      await canvas2.hover({ position: { x: 200, y: 200 } });
      
      // 等待同步
      await page1.waitForTimeout(500);
      
      // 验证用户1看到用户2的光标
      await expect(page1.locator('[data-testid="remote-cursor-user2"]')).toBeVisible();
      
      console.log('✓ 光标位置同步测试通过');
      
    } finally {
      await context1.close();
      await context2.close();
    }
  });

  test('应该处理并发冲突', async ({ browser }) => {
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    
    const page1 = await context1.newPage();
    const page2 = await context2.newPage();
    
    try {
      // 用户1创建文档并添加元素
      await login(page1, 'user1');
      await createDocument(page1, '冲突测试文档');
      await addWallElement(page1, 100, 100, 200, 100);
      
      const documentUrl = page1.url();
      
      // 用户2加入
      await login(page2, 'user2');
      await page2.goto(documentUrl);
      await page2.waitForSelector('[data-testid="editor-container"]');
      
      // 同时修改同一元素（模拟冲突）
      // 用户1移动元素
      const element1 = page1.locator('[data-testid="canvas-element"]').first();
      await element1.dragTo(page1.locator('[data-testid="design-canvas"]'), {
        targetPosition: { x: 300, y: 300 }
      });
      
      // 用户2同时修改同一元素
      const element2 = page2.locator('[data-testid="canvas-element"]').first();
      await element2.dragTo(page2.locator('[data-testid="design-canvas"]'), {
        targetPosition: { x: 400, y: 400 }
      });
      
      // 等待冲突处理
      await page1.waitForTimeout(2000);
      await page2.waitForTimeout(2000);
      
      // 验证冲突被处理（可能显示冲突提示）
      const conflictToast = page1.locator('[data-testid="conflict-toast"]');
      if (await conflictToast.isVisible().catch(() => false)) {
        console.log('✓ 冲突检测和提示测试通过');
      } else {
        // 或者验证最终状态一致
        const elements1 = await page1.locator('[data-testid="canvas-element"]').count();
        const elements2 = await page2.locator('[data-testid="canvas-element"]').count();
        expect(elements1).toBe(elements2);
        console.log('✓ 冲突自动解决测试通过');
      }
      
    } finally {
      await context1.close();
      await context2.close();
    }
  });
});

test.describe('脚本执行功能', () => {
  
  test.beforeEach(async ({ page }) => {
    await login(page, 'test-user');
    await createDocument(page, '脚本测试文档');
  });

  test('用户应该能够执行Python脚本', async ({ page }) => {
    // 打开脚本编辑器
    await page.click('[data-testid="script-editor-button"]');
    await page.waitForSelector('[data-testid="script-editor-panel"]');
    
    // 输入脚本
    const script = `
import math

def create_circle(radius):
    points = []
    for i in range(36):
        angle = 2 * math.pi * i / 36
        x = radius * math.cos(angle)
        y = radius * math.sin(angle)
        points.append((x, y))
    return points

result = create_circle(10)
print(f"Created {len(result)} points")
    `;
    
    await page.fill('[data-testid="script-input"]', script);
    
    // 执行脚本
    await page.click('[data-testid="run-script-button"]');
    
    // 验证执行成功
    await expect(page.locator('[data-testid="script-success-output"]')).toBeVisible();
    await expect(page.locator('[data-testid="script-output"]')).toContainText('Created 36 points');
    
    console.log('✓ 脚本执行测试通过');
  });

  test('脚本错误应该被正确捕获', async ({ page }) => {
    await page.click('[data-testid="script-editor-button"]');
    await page.waitForSelector('[data-testid="script-editor-panel"]');
    
    // 输入有错误的脚本
    const script = `
# 故意引发错误
result = 1 / 0
    `;
    
    await page.fill('[data-testid="script-input"]', script);
    await page.click('[data-testid="run-script-button"]');
    
    // 验证错误被捕获
    await expect(page.locator('[data-testid="script-error-output"]')).toBeVisible();
    await expect(page.locator('[data-testid="script-error-message"]')).toContainText('ZeroDivisionError');
    
    console.log('✓ 脚本错误处理测试通过');
  });

  test('危险脚本应该被沙箱阻止', async ({ page }) => {
    await page.click('[data-testid="script-editor-button"]');
    await page.waitForSelector('[data-testid="script-editor-panel"]');
    
    // 输入尝试访问文件系统的脚本
    const script = `
try:
    with open('/etc/passwd', 'r') as f:
        content = f.read()
    result = {"success": True}
except Exception as e:
    result = {"success": False, "error": str(e)}
    `;
    
    await page.fill('[data-testid="script-input"]', script);
    await page.click('[data-testid="run-script-button"]');
    
    // 验证沙箱阻止了文件访问
    await expect(page.locator('[data-testid="script-output"]')).toContainText('False');
    
    console.log('✓ 沙箱安全测试通过');
  });
});

test.describe('版本控制功能', () => {
  
  test.beforeEach(async ({ page }) => {
    await login(page, 'test-user');
  });

  test('用户应该能够查看版本历史', async ({ page }) => {
    // 创建文档并进行多次修改
    await createDocument(page, '版本测试文档');
    await addWallElement(page, 100, 100, 200, 100);
    await page.click('[data-testid="save-button"]');
    await page.waitForSelector('[data-testid="save-success-toast"]');
    
    // 添加更多元素并保存
    await addWallElement(page, 200, 100, 200, 200);
    await page.click('[data-testid="save-button"]');
    
    // 打开版本历史
    await page.click('[data-testid="version-history-button"]');
    await page.waitForSelector('[data-testid="version-list"]');
    
    // 验证版本列表显示
    const versions = page.locator('[data-testid="version-item"]');
    await expect(versions).toHaveCount(2);
    
    console.log('✓ 版本历史查看测试通过');
  });

  test('用户应该能够回滚到历史版本', async ({ page }) => {
    // 创建文档
    await createDocument(page, '回滚测试文档');
    await addWallElement(page, 100, 100, 200, 100);
    await page.click('[data-testid="save-button"]');
    
    // 添加第二个元素
    await addWallElement(page, 200, 100, 200, 200);
    await page.click('[data-testid="save-button"]');
    
    // 验证有两个元素
    await expect(page.locator('[data-testid="canvas-element"]')).toHaveCount(2);
    
    // 打开版本历史
    await page.click('[data-testid="version-history-button"]');
    await page.waitForSelector('[data-testid="version-list"]');
    
    // 选择第一个版本并回滚
    const firstVersion = page.locator('[data-testid="version-item"]').first();
    await firstVersion.click();
    await page.click('[data-testid="rollback-button"]');
    await page.click('[data-testid="confirm-rollback-button"]');
    
    // 验证回滚成功（只剩一个元素）
    await expect(page.locator('[data-testid="canvas-element"]')).toHaveCount(1);
    
    console.log('✓ 版本回滚测试通过');
  });
});

// ==================== 性能测试 ====================
test.describe('性能基准测试', () => {
  
  test('文档加载时间应该小于3秒', async ({ page }) => {
    await login(page, 'test-user');
    await createDocument(page, '性能测试文档');
    
    // 添加一些元素
    for (let i = 0; i < 10; i++) {
      await addWallElement(page, 100 + i * 50, 100, 150 + i * 50, 100);
    }
    await page.click('[data-testid="save-button"]');
    
    // 测量加载时间
    const startTime = Date.now();
    await page.reload();
    await page.waitForSelector('[data-testid="editor-container"]');
    const loadTime = Date.now() - startTime;
    
    expect(loadTime).toBeLessThan(3000);
    console.log(`✓ 文档加载时间: ${loadTime}ms`);
  });

  test('大量元素渲染性能', async ({ page }) => {
    await login(page, 'test-user');
    await createDocument(page, '大量元素测试');
    
    // 添加大量元素
    const elementCount = 50;
    for (let i = 0; i < elementCount; i++) {
      await addWallElement(page, 
        50 + (i % 10) * 60, 
        50 + Math.floor(i / 10) * 60,
        100 + (i % 10) * 60,
        50 + Math.floor(i / 10) * 60
      );
    }
    
    // 验证所有元素渲染
    const elements = page.locator('[data-testid="canvas-element"]');
    await expect(elements).toHaveCount(elementCount);
    
    // 测试交互响应
    const startTime = Date.now();
    await page.click('[data-testid="tool-select"]');
    await elements.first().click();
    const interactionTime = Date.now() - startTime;
    
    expect(interactionTime).toBeLessThan(500);
    console.log(`✓ 大量元素交互响应时间: ${interactionTime}ms`);
  });
});
