# Testing Guide

## Running Tests

All tests can be run with:
```bash
cd backend && go test ./... -v
```

To run tests for a specific package:
```bash
go test ./internal/handlers/... -v
```

To run tests with coverage:
```bash
go test ./... -cover
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

To run tests with race detector:
```bash
go test ./... -race -short
```

## Test Environment

Tests automatically set up required environment variables. The `internal/testing` package provides helpers:

```go
import "github.com/tdawe1/translation-app/internal/testing"

func TestMyFunction(t *testing.T) {
    cfg := testing.TestConfig()
    // Use cfg for testing
}
```

## Writing Tests

1. Use `testing.SetupTestEnvironment()` at the start of your test
2. Use `testing.TestApp()` to create a test Fiber app
3. Follow TDD: write failing test first, then implementation

### Example Test

```go
package handlers

import (
    "testing"
    "github.com/tdawe1/translation-app/internal/testing"
)

func TestHealthCheck(t *testing.T) {
    app := testing.TestApp()
    cfg := testing.TestConfig()

    // Setup your handlers...
    // app.Get("/health", handlers.HealthCheck(cfg))

    // Make test requests...
}
```

## Environment Variables

The following are automatically set for testing:
- JWT_SECRET: Test key for JWT validation
- DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME: Test database credentials
- RESEND_API_KEY: test-key
- FROM_EMAIL: test@example.com
- GOOGLE_OAUTH_CLIENT_ID, GOOGLE_OAUTH_CLIENT_SECRET: test values
- GITHUB_OAUTH_CLIENT_ID, GITHUB_OAUTH_CLIENT_SECRET: test values
- ENV: test

## Test Helpers

| Function | Description |
|----------|-------------|
| `SetupTestEnvironment()` | Sets all required environment variables |
| `TestConfig()` | Returns a configured `*config.Config` for testing |
| `TestApp()` | Creates a Fiber app configured for testing |
| `CleanupTestEnvironment()` | Clears test environment variables |

## CI/CD

Tests run automatically on push/PR to main and develop branches via GitHub Actions. See `.github/workflows/test.yml`.
