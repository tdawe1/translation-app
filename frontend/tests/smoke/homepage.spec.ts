import { test, expect } from '@playwright/test';
import { shouldFilterConsoleError } from './helpers';

test.describe('Homepage @smoke', () => {
  test('should load and display main content', async ({ page }) => {
    await page.goto('/');

    // Main heading should be visible
    await expect(page.locator('h1')).toContainText('GengoWatcher SaaS');

    // Description should be visible
    await expect(page.locator('text=Multi-tenant job monitoring')).toBeVisible();

    // CTA buttons should be present
    await expect(page.locator('a', { hasText: 'Get Started' })).toBeVisible();
    await expect(page.locator('a', { hasText: 'Sign In' })).toHaveAttribute('href', '/auth/login');
  });

  test('should have no application console errors', async ({ page }) => {
    const errors: string[] = [];

    page.on('console', msg => {
      if (msg.type() === 'error') {
        const text = msg.text();
        // Filter out benign errors
        if (!shouldFilterConsoleError(text)) {
          errors.push(text);
        }
      }
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should load in under 2 seconds', async ({ page }) => {
    const startTime = Date.now();
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    const duration = Date.now() - startTime;

    expect(duration).toBeLessThan(2000);
  });

  test('should navigate to register page', async ({ page }) => {
    await page.goto('/');

    // Click Get Started button
    await page.click('a:has-text("Get Started")');

    // Should navigate to register
    await page.waitForURL('/auth/register');
  });
});
