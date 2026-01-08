import { test, expect, shouldFilterConsoleError } from './helpers';

test.describe('Dashboard @smoke @critical', () => {
  test('should display dashboard with main elements', async ({ authenticatedPage: page }) => {
    // Main heading should be visible
    await expect(page.getByTestId('dashboard-heading')).toContainText('Dashboard');

    // User email should be displayed
    await expect(page.getByTestId('user-email')).toBeVisible();

    // Settings link should be present
    await expect(page.getByTestId('settings-link')).toBeVisible();

    // Sign out button should be present
    await expect(page.getByTestId('sign-out-button')).toBeVisible();
  });

  test('should display watcher status card', async ({ authenticatedPage: page }) => {
    // Watcher Status heading should be visible
    await expect(page.locator('text=Watcher Status')).toBeVisible();

    // Status should be one of: Running, Stopped, Error, or Loading
    const statusElement = page.getByTestId('watcher-status');
    await expect(statusElement).toBeVisible();
  });

  test('should have watcher control buttons', async ({ authenticatedPage: page }) => {
    // Start Watcher button
    const startButton = page.getByTestId('start-watcher-button');
    await expect(startButton).toBeVisible();

    // Stop Watcher button
    const stopButton = page.getByTestId('stop-watcher-button');
    await expect(stopButton).toBeVisible();

    // Configure button
    const configButton = page.getByTestId('configure-button');
    await expect(configButton).toBeVisible();
  });

  test('should have no application console errors', async ({ authenticatedPage: page }) => {
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
    await page.waitForLoadState('domcontentloaded');

    expect(errors).toHaveLength(0);
  });

  test('should load in under 2 seconds', async ({ authenticatedPage: page }) => {
    const startTime = Date.now();
    await page.goto('/dashboard');
    await page.waitForLoadState('domcontentloaded');
    const duration = Date.now() - startTime;

    expect(duration).toBeLessThan(2000);
  });

  test('should open configuration modal with Ctrl+K shortcut', async ({ authenticatedPage: page }) => {
    // Press Ctrl+K to open config modal
    await page.keyboard.press('Control+k');

    // Modal should appear - use .first() to avoid strict mode violation with multiple headings
    await expect(page.locator('text=Watcher Configuration').first()).toBeVisible({ timeout: 3000 });
  });

  test('should navigate to settings page', async ({ authenticatedPage: page }) => {
    await page.click('a[href="/settings"]');

    // Should navigate to settings
    await page.waitForURL('/settings');
    // Look for the specific Settings heading (h1 with class containing "Settings")
    await expect(page.locator('h1:has-text("Settings")')).toBeVisible();
  });
});
