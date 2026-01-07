import { z } from 'zod';

/**
 * Test user credentials for smoke testing
 * Each test run generates unique credentials to avoid conflicts
 */
export interface TestUser {
  email: string;
  password: string;
  userId: string;
}

/**
 * Console error filter for smoke tests
 * Excludes benign errors that don't indicate real problems
 *
 * Filtered patterns:
 * - favicon.ico 404s (browser automatic request)
 * - third-party resources that may fail during tests
 * - DevTools warnings that appear in headless mode
 */
export function shouldFilterConsoleError(message: string): boolean {
  const filteredPatterns = [
    'favicon.ico',
    'third-party-domain.com',
    'DevTools failed to load',
  ];

  return filteredPatterns.some(pattern => message.includes(pattern));
}

/**
 * Service health check result schema
 * Used to verify all dependencies are ready before running tests
 */
export const ServiceHealthSchema = z.object({
  postgresql: z.boolean(),
  redis: z.boolean(),
  backend: z.boolean(),
  frontend: z.boolean(),
});

export type ServiceHealth = z.infer<typeof ServiceHealthSchema>;

/**
 * Test user generator configuration
 */
export interface TestUserConfig {
  passwordStrength: {
    minLength: number;
    requireUppercase: boolean;
    requireLowercase: boolean;
    requireNumber: boolean;
    requireSpecial: boolean;
  };
}

/**
 * Default test password that meets backend strength requirements
 * - At least 12 characters
 * - Contains uppercase, lowercase, number, and special character
 */
export const DEFAULT_TEST_PASSWORD = 'TestPassword123!';

/**
 * API response types for test assertions
 */
export interface AuthResponse {
  access_token: string;
  refresh_token?: string;
  user_id: string;
  email: string;
}

export interface ErrorResponse {
  error: string;
  code: string;
  details?: Record<string, unknown>;
}
