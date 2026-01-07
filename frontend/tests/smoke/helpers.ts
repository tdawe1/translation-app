import { test as base, Page } from '@playwright/test';
import type { TestUser } from './types';
import { DEFAULT_TEST_PASSWORD } from './types';

/**
 * Generate a unique test user per test run
 * Uses timestamp + random suffix to prevent email conflicts
 */
export function generateTestUser(): TestUser {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(7);
  return {
    email: `smoke-test-${timestamp}-${random}@example.com`,
    password: DEFAULT_TEST_PASSWORD,
    userId: '',
  };
}

/**
 * Create a test user via the backend API
 * Must be called before attempting to login in tests
 *
 * @param user - TestUser object with email and password
 * @throws Error if registration fails
 */
export async function createTestUser(user: TestUser): Promise<void> {
  const response = await fetch('http://localhost:8000/api/v1/auth/register', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: user.email,
      password: user.password,
    }),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Failed to create test user: ${response.statusText} - ${errorText}`);
  }

  const data = await response.json();
  user.userId = data.user_id || data.id || '';
}

/**
 * Delete a test user via the backend API
 * Attempts cleanup even if it fails (logs error but doesn't throw)
 *
 * @param userId - UUID of the user to delete
 * @param accessToken - Valid JWT for authentication
 */
export async function deleteTestUser(userId: string, accessToken: string): Promise<void> {
  try {
    await fetch(`http://localhost:8000/api/v1/users/${userId}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${accessToken}`,
      },
    });
  } catch (error) {
    // Log but don't throw - cleanup shouldn't fail the test
    console.warn(`Failed to cleanup test user ${userId}:`, error);
  }
}

/**
 * Perform login via UI and verify success
 * Handles the full login flow including redirect verification
 *
 * @param page - Playwright Page object
 * @param email - User email
 * @param password - User password
 */
export async function loginViaUI(
  page: Page,
  email: string,
  password: string
): Promise<void> {
  // Navigate to login page
  await page.goto('/auth/login');

  // Fill credentials
  await page.fill('input[name="email"]', email);
  await page.fill('input[name="password"]', password);

  // Submit form
  await page.click('button[type="submit"]');

  // Wait for redirect to dashboard
  await page.waitForURL('/dashboard', { timeout: 10000 });

  // Verify login succeeded (no error toast visible)
  const errorToast = page.locator('[data-testid="error-toast"], [role="alert"]').first();
  if (await errorToast.isVisible({ timeout: 1000 }).catch(() => false)) {
    const errorText = await errorToast.textContent();
    throw new Error(`Login failed: error toast displayed - ${errorText}`);
  }
}

/**
 * Extended test fixture with authenticated page
 * Automatically creates a test user and logs in before each test
 */
export const test = base.extend<{ authenticatedPage: Page }>({
  authenticatedPage: async ({ page }, use) => {
    const user = generateTestUser();
    await createTestUser(user);

    // Login via UI
    await loginViaUI(page, user.email, user.password);

    // Store user info on page for potential cleanup
    (page as any).__testUser = user;

    await use(page);

    // Cleanup: Attempt to delete test user after test
    // Note: This is best-effort cleanup - tests should be isolated via unique emails
    const accessToken = await page.evaluate(() =>
      sessionStorage.getItem('access_token')
    );
    if (accessToken && user.userId) {
      await deleteTestUser(user.userId, accessToken);
    }
  },
});

/**
 * Re-export base test for non-authenticated tests
 */
export { expect } from '@playwright/test';
