import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/smoke',
  fullyParallel: false, // Disabled to prevent DB race conditions
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1, // Single worker for test isolation

  reporter: [
    ['html', { open: 'never', outputFolder: 'playwright-report' }],
    ['list']
  ],

  use: {
    baseURL: 'http://localhost:3001',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    navigationTimeout: 10000, // 10s timeout (not 30s default)
  },

  // Global setup handles service health check
  globalSetup: require.resolve('./tests/smoke/global-setup'),
});
