import { FullConfig } from '@playwright/test';

/**
 * Global setup for smoke tests
 * Runs once before all tests to verify services are healthy
 *
 * This prevents test failures due to services not being ready
 * and provides clearer error messages when dependencies are missing.
 */
export default async function globalSetup(config: FullConfig): Promise<void> {
  const services = {
    backend: 'http://localhost:8000',
    frontend: 'http://localhost:3001',
  };

  const errors: string[] = [];

  // Check backend health
  try {
    const backendResponse = await fetch(`${services.backend}/health`, {
      signal: AbortSignal.timeout(5000),
    });
    if (!backendResponse.ok) {
      errors.push(`Backend health check failed: ${backendResponse.status}`);
    }
  } catch (error) {
    errors.push(`Backend not reachable at ${services.backend}. Run: ./scripts/dev.sh backend start`);
  }

  // Check frontend is responding
  try {
    const frontendResponse = await fetch(services.frontend, {
      signal: AbortSignal.timeout(5000),
    });
    if (!frontendResponse.ok) {
      errors.push(`Frontend returned error: ${frontendResponse.status}`);
    }
  } catch (error) {
    errors.push(`Frontend not reachable at ${services.frontend}. Run: ./scripts/dev.sh frontend start`);
  }

  // If any checks failed, abort with helpful message
  if (errors.length > 0) {
    console.error('\n❌ Smoke test prerequisites not met:\n');
    errors.forEach((err) => console.error(`  - ${err}`));
    console.error('\n💡 Quick fix: Run ./scripts/dev.sh up\n');
    throw new Error('Services not ready. Aborting tests.');
  }

  console.log('✅ Smoke test global setup complete - all services healthy');
}
