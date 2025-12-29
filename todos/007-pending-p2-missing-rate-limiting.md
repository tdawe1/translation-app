---
status: resolved
priority: p2
issue_id: "007"
tags:
  - security
  - api
  - rate-limiting
  - code-review
dependencies: []
---

# P2: Missing Rate Limiting on Auth Endpoints

## Problem Statement

Authentication endpoints (`/api/v1/auth/login`, `/api/v1/auth/register`) have no rate limiting, allowing credential stuffing and account enumeration attacks.

**Files**: `backend/internal/handlers/auth.go` (all endpoints)

## Findings

### OWASP Classification
- **OWASP**: A07:2021 - Identification and Authentication Failures
- **CWE**: CWE-307 (Improper Restriction of Excessive Authentication Attempts)

### Attack Scenarios

1. **Credential Stuffing**: Attacker tries thousands of password/email combinations
2. **Account Enumeration**: Attacker determines which emails are registered
3. **DoS**: Attacker registers thousands of fake accounts
4. **Resource Exhaustion**: High-volume requests consume database connections

### Impact
- **Severity**: IMPORTANT - Enables automated attacks
- **Confidence**: 95/100 - No rate limiting found in code

### Evidence
- No rate limiting middleware in `cmd/server/main.go`
- No rate limiting in auth handlers
- Standard Fiber app without rate limiting plugins

## Proposed Solutions

### Option 1: Redis-Based Rate Limiting (Recommended for Production)

```go
import (
    "github.com/redis/go-redis/v9"
    "golang.org/x/time/rate"
)

type RateLimiter struct {
    redis *redis.Client
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
    return &RateLimiter{redis: redisClient}
}

func (r *RateLimiter) Allow(ctx context.Context, key string, limit rate.Limit, window time.Duration) (bool, error) {
    // Use Redis INCR with expiration
    countKey := fmt.Sprintf("ratelimit:%s:%s", key, time.Now().Truncate(window).Format("2006-01-02T15:04:00"))

    count, err := r.redis.Incr(ctx, countKey).Result()
    if err != nil {
        return false, err
    }

    if count == 1 {
        r.redis.Expire(ctx, countKey, window)
    }

    return count <= int(limit), nil
}

// Middleware
func (r *RateLimiter) Middleware(endpoint string, requests int, window time.Duration) fiber.Handler {
    return func(c *fiber.Ctx) error {
        allowed, err := r.Allow(c.Context(), endpoint, rate.Limit(requests), window)
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": "Rate limit error"})
        }
        if !allowed {
            return c.Status(429).JSON(fiber.Map{
                "error": "Too many requests",
                "code":  "RATE_LIMITED",
                "retry_after": window.Seconds(),
            })
        }
        return c.Next()
    }
}
```

**Usage**:
```go
// In main.go
loginLimiter := ratelimit.Middleware("login", 10, time.Minute)
registerLimiter := ratelimit.Middleware("register", 3, time.Minute)

authGroup.Post("/register", loginLimiter, authHandler.Register)
authGroup.Post("/login", registerLimiter, authHandler.Login)
```

**Pros**:
- Distributed rate limiting (works across multiple servers)
- Persistent across restarts
- Configurable per endpoint
- Returns proper 429 status

**Cons**:
- Requires Redis
- More complex than simple in-memory limiter
- Additional dependency

**Effort**: Medium
**Risk**: Low

### Option 2: In-Memory Rate Limiting (Simpler)

```go
import "golang.org/x/time/rate"

var (
    loginLimiter    = rate.NewLimiter(rate.Every(time.Second), 10)
    registerLimiter = rate.NewLimiter(rate.Every(time.Minute), 3)
)

func RateLimit(limiter *rate.Limiter) fiber.Handler {
    return func(c *fiber.Ctx) error {
        if !limiter.Allow() {
            return c.Status(429).JSON(fiber.Map{
                "error": "Too many requests",
                "code":  "RATE_LIMITED",
            })
        }
        return c.Next()
    }
}

// Usage
authGroup.Post("/register", RateLimit(registerLimiter), authHandler.Register)
authGroup.Post("/login", RateLimit(loginLimiter), authHandler.Login)
```

**Pros**:
- Simple implementation
- No external dependencies (beyond x/time/rate)
- Fast

**Cons**:
- Not distributed (each server has its own counter)
- Resets on restart
- Can be bypassed by rotating through servers

**Effort**: Small
**Risk**: Medium (not distributed)

### Option 3: Fiber Rate Limiter Plugin

```go
import "github.com/gofiber/fiber/v2/middleware/limiter"

// Config
limiterConfig := limiter.Config{
    Max:        10,
    Expiration: 1 * time.Minute,
    KeyGenerator: func(c *fiber.Ctx) string {
        return c.IP() // or c.Get("X-Forwarded-For")
    },
    LimitReached: func(c *fiber.Ctx) error {
        return c.Status(429).JSON(fiber.Map{
            "error": "Too many requests",
            "code":  "RATE_LIMITED",
        })
    },
}

// Usage
authGroup.Post("/login", limiter.New(limiterConfig), authHandler.Login)
```

**Pros**:
- Drop-in Fiber middleware
- Well-tested
- Supports Redis backend

**Cons**:
- External plugin dependency
- Less flexible than custom implementation

**Effort**: Small
**Risk**: Low

## Recommended Action

**Implement Option 1** (Redis-based) for production, or **Option 3** (Fiber plugin) for quick implementation.

## Technical Details

### Affected Files
- `backend/cmd/server/main.go` (add middleware setup)
- `backend/internal/ratelimit/limiter.go` (new file)
- `backend/go.mod` (add dependencies)

### Dependencies
- `github.com/redis/go-redis/v9` (already in use)
- `golang.org/x/time/rate`

### Database Changes
None (uses Redis for rate limit storage)

## Acceptance Criteria

- [ ] Rate limiting implemented on `/register` endpoint (3/minute)
- [ ] Rate limiting implemented on `/login` endpoint (10/second)
- [ ] Returns 429 status code when limit exceeded
- [ ] Includes `Retry-After` header
- [ ] Uses Redis for distributed limiting
- [ ] IP-based limiting (with X-Forwarded-For support)
- [ ] Tests verify rate limits work correctly
- [ ] Documentation updated

## Work Log

### 2025-12-29
- **Finding**: Security audit identified missing rate limiting
- **Analysis**: Confirmed auth endpoints are unprotected
- **Decision**: Selected Redis-based approach for production
- **Status**: Pending implementation

## Resources

- [OWASP Rate Limiting](https://cheatsheetseries.owasp.org/cheatsheets/Rate_Limiting_Cheat_Sheet.html)
- [OWASP A07:2021](https://owasp.org/Top10/A07_2021-Identification_and_Authentication_Failures/)
- [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate)
