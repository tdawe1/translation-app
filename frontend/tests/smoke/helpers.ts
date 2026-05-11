import { test as base, Page } from '@playwright/test';
import type { TestUser } from './types';
import { DEFAULT_TEST_PASSWORD, shouldFilterConsoleError } from './types';

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:37181';

/**
 * Generate a unique test admin user per test run
 * Uses timestamp + random suffix to prevent email conflicts.
 * Uses a fixed domain for admin users.
 */
export function generateTestAdmin(): TestUser {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(7);
  return {
    email: `smoke-admin-${timestamp}-${random}@admin.local`,
    password: DEFAULT_TEST_PASSWORD,
    userId: '',
  };
}

/**
 * Create (or update) an admin user via the dev seed endpoint.
 * This endpoint is only available in development and returns a valid JWT token.
 *
 * This is faster and more reliable than email/password registration for tests.
 *
 * @param user - TestUser object with email and password
 * @returns JWT access token
 * @throws Error if seeding fails
 */
export async function seedAdminUser(user: TestUser): Promise<string> {
  const response = await fetch(`${BACKEND_URL}/dev/seed-admin`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: user.email,
      password: user.password,
    }),
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Failed to seed admin user: ${response.statusText} - ${errorText}`);
  }

  const data = await response.json();
  user.userId = data.user_id || data.id || '';
  return data.token as string;
}

/**
 * Set the JWT token as an httpOnly cookie and wait for auth state to propagate.
 * This bypasses the UI login flow for faster, more reliable tests.
 *
 * @param page - Playwright Page object
 * @param token - JWT access token
 */
export async function authenticateWithToken(
  page: Page,
  token: string
): Promise<void> {
  await page.context().addCookies([
    {
      name: 'session_token',
      value: token,
      url: BACKEND_URL,
      httpOnly: true,
      sameSite: 'Lax',
      path: '/',
    },
  ]);

  await page.goto('/');
  await page.reload();

  // Navigate to dashboard to verify authentication worked
  await page.goto('/dashboard');

  // Wait for dashboard heading by data-testid (more specific than h2)
  // Using getByTestId is more reliable as it won't match unexpected elements
  await page.waitForSelector('[data-testid="dashboard-heading"]', { timeout: 10000 });
  await page.waitForLoadState('domcontentloaded');
}

/**
 * Extended test fixture with authenticated page using admin token.
 * Automatically seeds an admin user and authenticates before each test.
 *
 * This uses the dev-only seed endpoint which is faster than UI-based login
 * and provides a consistent admin user for all smoke tests.
 */
export const test = base.extend<{ authenticatedPage: Page }>({
  authenticatedPage: async ({ page }, use) => {
    const adminUser = generateTestAdmin();
    const token = await seedAdminUser(adminUser);

    // Authenticate by setting token directly (bypasses UI login)
    await authenticateWithToken(page, token);

    // Store user info on page for potential cleanup
    (page as any).__testUser = adminUser;

    await use(page);

    // Note: We don't delete admin users after tests since they're
    // created with unique emails each run and may be useful for debugging.
    // The database can be reset via `docker-compose down -v` if needed.
  },
});

/**
 * Re-export base test for non-authenticated tests
 */
export { expect } from '@playwright/test';

// Re-export console error filter from types for convenience
export { shouldFilterConsoleError };
