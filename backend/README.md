# GengoWatcher Backend

Go backend service for the GengoWatcher SaaS platform. Built with Fiber v2, GORM, and PostgreSQL.

## Running Tests

### Quick Start (with test database setup)

```bash
# Option 1: Using make target
make test-with-setup

# Option 2: Manual setup
make test-db-up      # Start test database
make test            # Run tests
make test-db-down    # Stop test database
```

### Test Database Configuration

The test suite requires PostgreSQL. By default, tests use:
- **Database:** `gengowatcher_test`
- **Host:** `localhost:5433`
- **User:** `gengo`
- **Password:** `devpass`

The test database runs on port 5433 to avoid conflicts with the development database on port 5432.

### Running Individual Tests

```bash
# Run all tests
make test

# Run with verbose output
make test-verbose

# Run with coverage
make test-coverage

# Run specific test file
go test ./tests/auth_test.go -v

# Run specific test function
go test ./tests/auth_test.go -run TestMagicLink -v
```

### Test Requirements

- Docker and Docker Compose (for test database)
- Go 1.25 or later
- PostgreSQL (if not using Docker)

### Environment Variables

Tests use default environment variables for testing. Override with:

```bash
TEST_DB_HOST=localhost \
TEST_DB_PORT=5433 \
TEST_DB_NAME=gengowatcher_test \
make test
```

## Development

### Running the Server

```bash
# Development with hot reload (requires air)
air

# Or standard go run
go run ./cmd/server

# Production build
go build -o server ./cmd/server
./server
```

### Project Structure

```
backend/
├── cmd/
│   ├── server/            # Application entry point
│   └── admin_seed/        # Admin seeding CLI tool
├── internal/              # Private application code
│   ├── auth/              # JWT, password hashing, user service
│   ├── config/            # Environment-based configuration
│   ├── database/          # GORM connection, pooling
│   ├── email/             # Resend email service
│   ├── errors/            # Error definitions
│   ├── handlers/          # HTTP request handlers (routes)
│   ├── middleware/        # Fiber middleware (JWT, CORS, etc.)
│   ├── models/            # GORM models (User, Watcher, Subscription, etc.)
│   ├── oauth/             # OAuth provider logic
│   ├── password/          # Password hashing utilities
│   ├── seeds/             # Admin seeding for development/testing
│   ├── service/           # Token service (verification, magic link, reset)
│   └── watcher/           # RSS/WebSocket monitoring logic
├── tests/                 # Backend integration tests
│   └── helpers.go         # Test utilities
├── scripts/               # Development and test scripts
├── Makefile               # Test commands
├── go.mod                 # Go dependencies
└── main.go                # Server entry point with dependency injection
```

### Adding New Tests

Tests mirror the `internal/` structure under `tests/` directory:

```go
func TestFeature_Behavior(t *testing.T) {
    db := RequireDB(t)
    redisClient := RequireRedis(t)

    // Create test config
    cfg := &config.Config{
        JWTSecret: "test-secret-for-testing-only-32-chars-min",
    }

    // Create test app with Fiber
    app := fiber.New(fiber.Config{
        AppName:               "GengoWatcher Test",
        DisableStartupMessage: true,
    })

    // Register routes
    app.Post("/api/v1/feature", handler.DoThing)

    t.Run("Subtest description", func(t *testing.T) {
        // Arrange
        reqBody := bytes.NewBufferString(`{"email":"test@example.com"}`)
        req := httptest.NewRequest("POST", "/api/v1/feature", reqBody)
        req.Header.Set("Content-Type", "application/json")

        // Act
        resp, err := app.Test(req)
        require.NoError(t, err)

        // Assert
        assert.Equal(t, 200, resp.StatusCode)
    })
}
```

## API Routes

- **Auth:** `/api/v1/auth/*` - Register, login, magic links, OAuth
- **User:** `/api/v1/me/*` - User profile, password change
- **Watcher:** `/api/v1/watcher/*` - Configuration and state management
- **Admin:** `/api/v1/admin/*` - Admin-only endpoints (requires admin role)

## Configuration

Environment variables are loaded from `.env` file or system environment:

```bash
# Server
PORT=8000

# Database
DATABASE_URL=postgres://gengo:devpass@localhost:5432/gengowatcher

# Redis
REDIS_URL=redis://localhost:6379

# Security
JWT_SECRET=                   # Required (32+ chars recommended)
COOKIE_SECURE=false           # Set true in production

# Email (Resend)
RESEND_API_KEY=
FROM_EMAIL=
FROM_NAME=

# OAuth
OAUTH_REDIRECT_URL=
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=

# Billing (LemonSqueezy)
LEMONSQUEEZY_WEBHOOK_SECRET=
LEMONSQUEEZY_API_KEY=
LEMONSQUEEZY_STORE_ID=
```
