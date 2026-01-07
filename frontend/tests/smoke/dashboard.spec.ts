import { test, expect } from '@playwright/test';
import { shouldFilterConsoleError } from './helpers';

test.describe('Dashboard @smoke @critical', () => {
  test.beforeEach(async ({ page }) => {
    // Create and login test user (inline for simplicity)
    const timestamp = Date.now();
    const email = `smoke-test-${timestamp}@example.com`;
    const password = 'TestPassword123!';

    // Register via API
    await fetch('http://localhost:8000/api/v1/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });

    // Login via UI
    await page.goto('/auth/login');
    await page.fill('input[name="email"]', email);
    await page.fill('input[name="password"]', password);
    await page.click('button[type="submit"]');
    await page.waitForURL('/dashboard', { timeout: 10000 });
  });

  test('should display dashboard with main elements', async ({ page }) => {
    // Main heading should be visible
    await expect(page.locator('h2')).toContainText('Dashboard');

    // User email should be displayed
    await expect(page.locator('text=/smoke-test-/')).toBeVisible();

    // Settings link should be present
    await expect(page.locator('a[href="/settings"]')).toBeVisible();

    // Sign out button should be present
    await expect(page.locator('button:has-text("Sign Out")')).toBeVisible();
  });

  test('should display watcher status card', async ({ page }) => {
    // Watcher Status heading should be visible
    await expect(page.locator('text=Watcher Status')).toBeVisible();

    // Status should be one of: Running, Stopped, Error, or Loading
    const statusElement = page.locator('[role="status"]').first();
    await expect(statusElement).toBeVisible();
  });

  test('should have watcher control buttons', async ({ page }) => {
    // Start Watcher button
    const startButton = page.locator('button:has-text("Start Watcher")');
    await expect(startButton).toBeVisible();

    // Stop Watcher button
    const stopButton = page.locator('button:has-text("Stop Watcher")');
    await expect(stopButton).toBeVisible();

    // Configure button
    const configButton = page.locator('button:has-text("Configure")');
    await expect(configButton).toBeVisible();
  });

  test('should have no application console errors', async ({ page }) => {
    const errors: string[] = [];

    page.on('console', msg => {
      if (msg.type() === 'error') {
        const text = msg.text();
        if (!shouldFilterConsoleError(text)) {
          errors.push(text);
        }
      }
    });

    await page.reload();
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should load in under 2 seconds', async ({ page }) => {
    const startTime = Date.now();
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');
    const duration = Date.now() - startTime;

    expect(duration).toBeLessThan(2000);
  });

  test('should open configuration modal with Ctrl+K shortcut', async ({ page }) => {
    // Press Ctrl+K to open config modal
    await page.keyboard.press('Control+k');

    // Modal should appear
    await expect(page.locator('text=Watcher Configuration')).toBeVisible({ timeout: 3000 });
  });

  test('should navigate to settings page', async ({ page }) => {
    await page.click('a[href="/settings"]');

    // Should navigate to settings
    await page.waitForURL('/settings');
    await expect(page.locator('h1, h2')).toBeVisible();
  });
});
