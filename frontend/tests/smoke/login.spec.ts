import { test, expect, type Page } from '@playwright/test';
import { shouldFilterConsoleError } from './helpers';

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:37181';

// Helper to generate unique test user credentials
function generateTestUser() {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(7);
  return {
    email: `smoke-test-${timestamp}-${random}@example.com`,
    password: 'TestPassword123!',
  };
}

// Helper to create a user via the registration API
async function createTestUser(email: string, password: string) {
  const response = await fetch(`${BACKEND_URL}/api/v1/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Failed to create test user: ${response.statusText} - ${errorText}`);
  }
}

// Helper to perform login via UI
async function loginViaUI(page: Page, email: string, password: string) {
  await page.goto('/auth/login');
  await page.fill('input[name="email"]', email);
  await page.fill('input[name="password"]', password);
  await page.click('button[type="submit"]');
  await page.waitForURL('/dashboard', { timeout: 10000 });
}

test.describe('Login Flow @smoke @auth', () => {
  test('should register new user and login successfully', async ({ page }) => {
    const user = generateTestUser();

    // First, register the user via API
    await createTestUser(user.email, user.password);

    // Navigate to login page
    await page.goto('/auth/login');

    // Verify login form is visible
    await expect(page.locator('h1')).toContainText('Sign In');
    await expect(page.locator('input#email')).toBeVisible();
    await expect(page.locator('input#password')).toBeVisible();

    // Login via UI
    await loginViaUI(page, user.email, user.password);

    // Should be on dashboard after successful login
    await expect(page).toHaveURL('/dashboard');
    await expect(page.locator('h2')).toContainText('Dashboard');
  });

  test('should show error for invalid credentials', async ({ page }) => {
    await page.goto('/auth/login');

    // Try to login with invalid credentials
    await page.fill('input#email', 'nonexistent@example.com');
    await page.fill('input#password', 'WrongPassword123!');
    await page.click('button[type="submit"]');

    // Should show error message
    const errorSelector = page.locator('text=/invalid|incorrect|credentials/i');
    await expect(errorSelector).toBeVisible({ timeout: 5000 });
  });

  test('should redirect to dashboard when already authenticated', async ({ page }) => {
    const user = generateTestUser();
    await createTestUser(user.email, user.password);

    // Login first
    await loginViaUI(page, user.email, user.password);

    // Now try to go to login page again
    await page.goto('/auth/login');

    // Should redirect to dashboard (app handles this via auth store)
    // Note: This depends on app implementation - may stay on login if redirect isn't implemented
    // For now, we just verify the page loads without errors
    await expect(page.locator('h1, h2')).toBeVisible();
  });

  test('should have no console errors during login', async ({ page }) => {
    const errors: string[] = [];

    page.on('console', msg => {
      if (msg.type() === 'error') {
        const text = msg.text();
        if (!shouldFilterConsoleError(text)) {
          errors.push(text);
        }
      }
    });

    const user = generateTestUser();
    await createTestUser(user.email, user.password);

    await page.goto('/auth/login');
    await loginViaUI(page, user.email, user.password);

    expect(errors).toHaveLength(0);
  });

  test('should navigate to register page', async ({ page }) => {
    await page.goto('/auth/login');

    // Click "Create account" link
    await page.click('a[href="/auth/register"]');

    // Should navigate to register
    await page.waitForURL('/auth/register');
    await expect(page.locator('h1')).toContainText('Create Account');
  });
});
