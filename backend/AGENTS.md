# AGENTS.md - Go Backend

> **Parent**: [../AGENTS.md](../AGENTS.md) | **Child**: [cmd/translation-worker/AGENTS.md](cmd/translation-worker/AGENTS.md)

## Overview

Go 1.25 backend using Fiber v2, GORM, and JWT auth. Handles API routes, watcher orchestration, and billing.

## Quick Commands

```bash
go run ./cmd/server           # Run server
make test                     # Run all tests
make test-coverage            # Generate coverage.html
go vet ./...                  # Type checking
go build -o server ./cmd/server  # Production build
```

## Directory Structure

```
backend/
├── cmd/
│   ├── server/              # Entry point (main.go, dependency injection)
│   ├── admin_seed/          # Admin seeding CLI
│   └── translation-worker/  # Python subsystem → See AGENTS.md
├── internal/
│   ├── handlers/            # HTTP handlers (18 files, highest density)
│   ├── watcher/             # RSS/WebSocket monitoring (8 files)
│   ├── middleware/          # JWT, CORS, rate limiting (8 files)
│   ├── models/              # GORM models (User, Watcher, Subscription)
│   ├── auth/                # JWT, password hashing, user service
│   ├── oauth/               # OAuth provider logic
│   └── config/              # Environment-based config
├── tests/                   # Integration tests (mirrors internal/)
├── Makefile                 # Test commands
└── go.mod                   # Dependencies
```

## High-Density Modules

| Directory | Files | Purpose |
|-----------|-------|---------|
| `internal/handlers/` | 18 | HTTP request handlers |
| `internal/watcher/` | 8 | Watcher core logic |
| `internal/middleware/` | 8 | Auth/CORS middleware |
| `tests/` | 14 | Integration tests |

## Code Patterns

### Error Handling
```go
import apperrors "github.com/tdawe1/translation-app/internal/errors"

// Always use typed errors
err := apperrors.New(apperrors.ErrUserNotFound, "user not found")
handlers.RespondWithError(c, 404, apperrors.ErrUserNotFound, "User not found")
```

### Handler Pattern
```go
func (h *MyHandler) DoThing(c *fiber.Ctx) error {
    userID, err := handlers.ParseUserID(c.Params("id"))
    if err != nil {
        return handlers.RespondWithError(c, 400, handlers.ErrInvalidUserID, "invalid id")
    }
    return c.JSON(fiber.Map{"data": result})
}
```

### UUID Handling
```go
import "github.com/google/uuid"

userID, err := uuid.Parse(c.Params("id"))
return c.JSON(fiber.Map{"id": user.ID.String()})
```

### Memory Management
```go
// Use LRU for in-memory deduplication
cache := watcher.NewLRUCache(1000)
if exists := cache.Add(jobID); exists {
    return nil // Skip duplicate
}

// Redis sets with TTL
_ = redisClient.Expire(ctx, "seen_jobs:"+userID, 24*time.Hour).Err()
```

## Test Pattern

```go
func TestFeature_Behavior(t *testing.T) {
    db := RequireDB(t)           // PostgreSQL on localhost:5433
    redisClient := RequireRedis(t) // Redis DB 15

    app := fiber.New(fiber.Config{DisableStartupMessage: true})
    // Register routes, run assertions
}
```

## MUST NOT

- Use `as any` or type error suppression
- Use global `models.DB` - inject dependencies
- Return specific email-exists errors (account enumeration)
- Skip JWT validation for any auth flow
- Use URLs with localhost/private IPs (SSRF)
- Ignore errors from `blocklist.Add()` (silent logout failure)

## API Routes

All routes: `/api/v1/*`

| Route | Handler | Purpose |
|-------|---------|---------|
| `/auth/*` | `auth_handler.go` | Registration, login, tokens |
| `/oauth/*` | `oauth.go` | Google/GitHub OAuth |
| `/watcher/*` | `watcher_handler.go` | Watcher config/state |
| `/dev/*` | `dev_handler.go` | Dev-only endpoints |

## Auth System

### JWT Blocklist
```go
// Token revocation on logout - backend/internal/auth/blocklist.go
blocklist := auth.NewTokenBlocklist(redisClient)
blocklist.Add(ctx, userID, tokenJTI, expiry)  // MUST check error
blocklist.IsBlocked(ctx, userID, tokenJTI)    // Check in JWT middleware
```

Redis key pattern: `user:{user_id}:blocklist:{token_id}` with TTL matching token expiry.

### Fail-Fast Secrets
Production REQUIRES (panics if missing):
- `JWT_SECRET` (32+ chars)
- `REDIS_URL`
- `DB_PASSWORD`
- `RESEND_API_KEY`

Test mode: Set `TEST_ENV=true` or `ENV=test` to bypass.

---

*For Python translation-worker, see [cmd/translation-worker/AGENTS.md](cmd/translation-worker/AGENTS.md)*
