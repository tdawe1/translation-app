# Test Environment Setup Fix

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Unblock CI/CD by fixing test environment - tests currently fail due to missing `JWT_SECRET` environment variable.

**Architecture:** Create a test environment configuration package that provides default values for required environment variables during testing, with explicit warnings to prevent production usage.

**Tech Stack:** Go 1.25, testing package, environment variable mocking

**Documentation:** `docs/contributing/testing.md`

---

## Overview

Current problem: Tests fail with this error:
```
FATAL: JWT_SECRET environment variable is not set.
```

This happens in `internal/middleware/jwt.go:37-38` which uses `log.Fatal()` during package initialization.

We'll fix this by:
1. Moving fatal checks from `init()` to runtime
2. Creating a test config package
3. Adding test environment setup

---

## Task 1: Fix JWT Middleware Fatal Initialization

**Files:**
- Modify: `backend/internal/middleware/jwt.go`

**Problem:** The JWT middleware has `log.Fatal()` calls at package level that execute before tests can set environment variables.

**Step 1: Read the current implementation**

Read: `backend/internal/middleware/jwt.go`

Note the problematic code at lines 36-73.

**Step 2: Write failing test**

Create: `backend/internal/middleware/jwt_test.go`

```go
package middleware

import (
    "os"
    "testing"

    "github.com/gofiber/fiber/v2"
)

func TestJWTConfig_InTestEnvironment(t *testing.T) {
    // This test should pass even without JWT_SECRET set
    app := fiber.New()

    // Add JWT middleware - it should not panic in tests
    app.Use(New())

    // Create a simple route
    app.Get("/health", func(c *fiber.Ctx) error {
        return c.SendString("OK")
    })

    // The middleware should not panic during test setup
    req, _ := http.NewRequest("GET", "/health", nil)
    resp, err := app.Test(req)

    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }

    if resp.StatusCode != fiber.StatusOK {
        t.Errorf("Expected 200, got %d", resp.StatusCode)
    }
}
```

**Step 3: Run test to verify it fails**

Run: `cd backend && go test ./internal/middleware/... -v -run TestJWTConfig`

Expected: FAIL with "FATAL: JWT_SECRET..." or panic

**Step 4: Refactor JWT middleware to use lazy initialization**

Edit: `backend/internal/middleware/jwt.go`

Change from immediate fatal checks to validation function:

```go
package middleware

import (
    "log"
    "os"

    "github.com/gofiber/fiber/v2"
    "github.com/golang-jwt/jwt/v5"
)

const (
    minSecretLength = 32 // 256 bits for HS256
)

// Config holds JWT middleware configuration
type Config struct {
    Secret string
}

// New creates a new JWT middleware with the given options
func New(opts ...Option) fiber.Handler {
    cfg := Config{
        Secret: os.Getenv("JWT_SECRET"),
    }

    // Apply options
    for _, opt := range opts {
        opt(&cfg)
    }

    // Validate configuration
    if err := validateConfig(&cfg); err != nil {
        log.Fatal(err) // Only fatal in server startup, not in tests
    }

    return jwtMiddleware(cfg)
}

// Option is a function that configures the JWT middleware
type Option func(*Config)

// WithSecret sets a custom JWT secret (useful for testing)
func WithSecret(secret string) Option {
    return func(cfg *Config) {
        cfg.Secret = secret
    }
}

// validateConfig checks if the configuration is valid
func validateConfig(cfg *Config) error {
    if cfg.Secret == "" {
        return fmt.Errorf("JWT_SECRET environment variable is not set. "+
            "Authentication cannot function without a secure secret. "+
            "Please set JWT_SECRET to a random string of at least 32 characters")
    }

    if len(cfg.Secret) < minSecretLength {
        return fmt.Errorf("JWT_SECRET must be at least %d characters (256 bits for HS256). "+
            "Current length: %d. Please generate a stronger secret.",
            minSecretLength, len(cfg.Secret))
    }

    return nil
}

// jwtMiddleware returns the actual Fiber middleware
func jwtMiddleware(cfg Config) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // ... existing middleware logic
        authHeader := c.Get("Authorization")
        if authHeader == "" {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "Missing authorization header",
            })
        }

        // ... rest of JWT validation
        return c.Next()
    }
}
```

**Step 5: Update test to use test secret**

Edit: `backend/internal/middleware/jwt_test.go`

```go
package middleware

import (
    "net/http"
    "os"
    "testing"

    "github.com/gofiber/fiber/v2"
)

func TestJWTConfig_WithTestSecret(t *testing.T) {
    app := fiber.New()

    // Use test secret to avoid env var requirement
    app.Use(New(WithSecret("test-secret-for-testing-only-32-chars-long!!")))

    app.Get("/health", func(c *fiber.Ctx) error {
        return c.SendString("OK")
    })

    req, _ := http.NewRequest("GET", "/health", nil)
    resp, err := app.Test(req)

    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }

    if resp.StatusCode != fiber.StatusOK {
        t.Errorf("Expected 200, got %d", resp.StatusCode)
    }
}

func TestJWTConfig_MissingSecretInProduction(t *testing.T) {
    // Clear JWT_SECRET to simulate missing env var
    oldSecret := os.Getenv("JWT_SECRET")
    os.Unsetenv("JWT_SECRET")
    defer func() {
        if oldSecret != "" {
            os.Setenv("JWT_SECRET", oldSecret)
        }
    }()

    // This should panic/log.Fatal during server startup
    // In tests, we can verify the validation error
    cfg := Config{Secret: ""}
    err := validateConfig(&cfg)

    if err == nil {
        t.Error("Expected error when JWT_SECRET is empty")
    }
}
```

**Step 6: Run tests to verify they pass**

Run: `cd backend && go test ./internal/middleware/... -v -run TestJWTConfig`

Expected: PASS

**Step 7: Commit**

```bash
cd backend
git add internal/middleware/jwt.go internal/middleware/jwt_test.go
git commit -m "refactor(middleware): make JWT middleware test-friendly with lazy validation"
```

---

## Task 2: Create Test Config Package

**Files:**
- Create: `backend/internal/testing/config.go`
- Create: `backend/internal/testing/testing.go`

**Step 1: Create testing package helpers**

Create: `backend/internal/testing/testing.go`

```go
// Package testing provides test utilities for the application
package testing

import (
    "os"
    "testing"

    "github.com/tdawe1/translation-app/internal/config"
)

// SetupTestEnvironment initializes environment variables for testing.
// Call this in TestMain or at the start of tests that require config.
func SetupTestEnvironment(t *testing.T) {
    t.Helper()

    // Set required environment variables with safe defaults for testing
    envVars := map[string]string{
        "JWT_SECRET":           "test-secret-key-32-characters-long-for-hs256!",
        "DATABASE_URL":         ":memory:",
        "REDIS_ADDR":           "localhost:6379",
        "EMAIL_FROM":           "test@example.com",
        "RESEND_API_KEY":       "test-key",
        "GOOGLE_CLIENT_ID":      "test-client-id",
        "GOOGLE_CLIENT_SECRET": "test-client-secret",
        "GITHUB_CLIENT_ID":      "test-client-id",
        "GITHUB_CLIENT_SECRET": "test-client-secret",
        "LEMONSQUEEZY_API_KEY":  "test-api-key",
        "LEMONSQUEEZY_STORE_ID": "test-store-id",
        "BASE_URL":             "http://localhost:8080",
        "FRONTEND_URL":         "http://localhost:3000",
    }

    for key, value := range envVars {
        if os.Getenv(key) == "" {
            os.Setenv(key, value)
            t.Cleanup(func() {
                os.Unsetenv(key)
            })
        }
    }
}

// TestConfig returns a config.Config instance for testing
func TestConfig(t *testing.T) *config.Config {
    t.Helper()
    SetupTestEnvironment(t)
    cfg, err := config.Load()
    if err != nil {
        t.Fatalf("Failed to load test config: %v", err)
    }
    return cfg
}
```

**Step 2: Create test helpers file**

Create: `backend/internal/testing/helpers.go`

```go
package testing

import (
    "github.com/gofiber/fiber/v2"
    "github.com/tdawe1/translation-app/internal/middleware"
)

// TestApp creates a Fiber app configured for testing
func TestApp(opts ...fiber.Config) *fiber.App {
    // Default test config
    cfg := fiber.Config{
        DisableStartupMessage: true,
        ErrorHandler: nil, // Use default error handler
        AppName:            "TestApp",
    }

    // Apply custom options
    for _, opt := range opts {
        opt(&cfg)
    }

    app := fiber.New(cfg)

    // Add JWT middleware with test secret
    app.Use(middleware.New(middleware.WithSecret("test-secret-key-32-characters-long-for-hs256!")))

    return app
}
```

**Step 3: Create tests for the testing package**

Create: `backend/internal/testing/testing_test.go`

```go
package testing

import (
    "testing"

    "github.com/tdawe1/translation-app/internal/config"
)

func TestSetupTestEnvironment(t *testing.T) {
    // Verify test environment is set up
    SetupTestEnvironment(t)

    // Load config to verify it works
    cfg, err := config.Load()
    if err != nil {
        t.Fatalf("Failed to load config: %v", err)
    }

    if cfg.JWTSecret == "" {
        t.Error("JWT_SECRET should be set in test environment")
    }

    if cfg.JWTSecret == "test-secret-key-32-characters-long-for-hs256!" {
        t.Log("Test environment properly configured")
    }
}

func TestTestConfig(t *testing.T) {
    cfg := TestConfig(t)

    if cfg == nil {
        t.Fatal("TestConfig returned nil")
    }

    if cfg.JWTSecret == "" {
        t.Error("JWT_SECRET not set in TestConfig")
    }
}
```

**Step 4: Run tests**

Run: `cd backend && go test ./internal/testing/... -v`

Expected: PASS

**Step 5: Commit**

```bash
cd backend
git add internal/testing/
git commit -m "test(testing): add test environment setup helpers"
```

---

## Task 3: Update Existing Tests to Use Test Helpers

**Files:**
- Modify: `backend/tests/auth_test.go`
- Modify: `backend/tests/watcher_test.go`
- Modify: `backend/tests/helpers.go`

**Step 1: Update test helpers**

Edit: `backend/tests/helpers.go`

Add import:
```go
    "github.com/tdawe1/translation-app/internal/testing"
```

Update setup functions to use `testing.SetupTestEnvironment()`.

**Step 2: Update auth_test.go**

Edit: `backend/tests/auth_test.go`

Add at top of test file:
```go
func TestMain(m *testing.M) {
    testing.SetupTestEnvironment(nil)
    os.Exit(m.Run())
}
```

**Step 3: Run all tests**

Run: `cd backend && go test ./... -v 2>&1 | head -50`

Expected: Tests should no longer fail with JWT_SECRET errors

**Step 4: Commit**

```bash
cd backend
git add tests/
git commit -m "test: update tests to use test environment helpers"
```

---

## Task 4: Add Test Documentation

**Files:**
- Create: `docs/contributing/testing.md`

**Step 1: Create testing documentation**

Create: `docs/contributing/testing.md`

```markdown
# Testing Guide

## Running Tests

All tests can be run with:
\`\`\`bash
cd backend && go test ./... -v
\`\`\`

To run tests for a specific package:
\`\`\`bash
go test ./internal/handlers/... -v
\`\`\`

## Test Environment

Tests automatically set up required environment variables. The `internal/testing` package provides helpers:

\`\`\`go
import "github.com/tdawe1/translation-app/internal/testing"

func TestMyFunction(t *testing.T) {
    cfg := testing.TestConfig(t)
    // Use cfg for testing
}
\`\`\`

## Writing Tests

1. Use `testing.SetupTestEnvironment(t)` at the start of your test
2. Use `testing.TestApp()` to create a test Fiber app
3. Follow TDD: write failing test first, then implementation

## Environment Variables

The following are automatically set for testing:
- JWT_SECRET: Test key for JWT validation
- DATABASE_URL: In-memory SQLite
- REDIS_ADDR: localhost:6379
- EMAIL_FROM: test@example.com
- All OAuth keys: test values
\`\`\`
```

**Step 2: Run documentation build**

Run: `cd docs && mkdocs build 2>/dev/null || echo "MkDocs not configured - skipping"`

**Step 3: Commit**

```bash
git add docs/contributing/testing.md
git commit -m "docs(testing): add comprehensive testing guide"
```

---

## Task 5: Add CI/CD Configuration

**Files:**
- Create: `.github/workflows/test.yml`

**Step 1: Create GitHub Actions workflow**

Create: `.github/workflows/test.yml`

```yaml
name: Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: testdb
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 6379:6379

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Download dependencies
        working-directory: backend
        run: go mod download

      - name: Run tests
        working-directory: backend
        run: go test ./... -v -cover

      - name: Run tests with race detector
        working-directory: backend
        run: go test ./... -race -short

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./backend/coverage.out
```

**Step 2: Create workflow directory if needed**

Run: `mkdir -p .github/workflows`

**Step 3: Validate workflow syntax**

Run: `cat .github/workflows/test.yml` | grep -E "name:|run:|uses:"`

Expected: Valid YAML structure

**Step 4: Commit**

```bash
git add .github/workflows/test.yml
git commit -m "ci: add GitHub Actions workflow for automated testing"
```

---

## Task 6: Final Verification

**Step 1: Run full test suite**

Run: `cd backend && go test ./... -v`

Expected: All tests pass, no JWT_SECRET fatal errors

**Step 2: Check test coverage**

Run: `cd backend && go test ./... -cover | grep -E "coverage:|ok"`

Expected: Coverage percentages displayed for all packages

**Step 3: Verify tests work without env vars**

Run: `unset JWT_SECRET && cd backend && go test ./internal/testing/... -v`

Expected: Tests still pass (test helpers provide defaults)

**Step 4: Final commit**

```bash
git add -A
git commit -m "test(testing): complete test environment setup - unblocks CI/CD"
```

---

## Success Criteria

- [ ] Tests run without requiring environment variables to be set
- [ ] `go test ./...` passes with 0 fatal errors
- [ ] Test coverage is tracked and reported
- [ ] CI/CD workflow created and tested
- [ ] Documentation updated

---

## Next Steps

After completing this plan:

1. **Auth Middleware Helper** - Eliminate remaining auth boilerplate
2. **Config Update Helper** - Generic ApplyPartialUpdate() function
3. **Handler Test Coverage** - Add unit tests for all handlers

---

**Estimated Time**: 4 hours
**Impact**: Unblocks CI/CD, enables 20+ hours/month savings in testing
