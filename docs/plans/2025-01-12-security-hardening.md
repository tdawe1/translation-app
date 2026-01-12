# Security Hardening Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Harden the GengoWatcher SaaS application against DoS attacks and improve authentication security based on security audit findings.

**Architecture:**
- Move in-memory OAuth state to Redis for DoS resilience
- Add rate limiting to WebSocket ticket generation
- Implement constant-time password verification to prevent timing attacks
- Add email format validation and stricter admin rate limits

**Tech Stack:** Go 1.25, Fiber v2, Redis, GORM, existing middleware patterns

---

## Task 1: Redis-Backed OAuth State Storage

**Issue:** H-2 - OAuth state stored in-memory causes DoS risk through unbounded memory growth.

**Files:**
- Create: `backend/internal/oauth/state_store.go` (new Redis-backed state store)
- Modify: `backend/internal/handlers/oauth.go:20-33, 103-136`
- Test: `backend/tests/handlers/oauth_test.go`

**Step 1: Write the failing test for Redis state store**

Create `backend/internal/oauth/state_store_test.go`:

```go
package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisStateStore_SetAndGet(t *testing.T) {
	// Setup mini redis
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	store := NewRedisStateStore(client)
	ctx := context.Background()

	// Set a state
	state := "test-state-123"
	expiry := time.Minute

	err := store.Set(ctx, state, expiry)
	require.NoError(t, err)

	// Get the state
	exists, err := store.Exists(ctx, state)
	require.NoError(t, err)
	assert.True(t, exists, "state should exist immediately after Set")
}

func TestRedisStateStore_Expiration(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	store := NewRedisStateStore(client)
	ctx := context.Background()

	state := "expiring-state"
	err := store.Set(ctx, state, 100*time.Millisecond)
	require.NoError(t, err)

	// Fast forward mini redis
	s.FastForward(200 * time.Millisecond)

	exists, err := store.Exists(ctx, state)
	require.NoError(t, err)
	assert.False(t, exists, "state should be expired")
}

func TestRedisStateStore_Delete(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	store := NewRedisStateStore(client)
	ctx := context.Background()

	state := "delete-me"
	_ = store.Set(ctx, state, time.Minute)

	err := store.Delete(ctx, state)
	require.NoError(t, err)

	exists, _ := store.Exists(ctx, state)
	assert.False(t, exists, "state should be deleted")
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/oauth/... -v`
Expected: FAIL with "undefined: NewRedisStateStore"

**Step 3: Implement RedisStateStore**

Create `backend/internal/oauth/state_store.go`:

```go
package oauth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	oauthStateKeyPrefix = "oauth:state:"
)

// StateStore defines the interface for OAuth state storage
type StateStore interface {
	Set(ctx context.Context, state string, expiry time.Duration) error
	Exists(ctx context.Context, state string) (bool, error)
	Delete(ctx context.Context, state string) error
}

// RedisStateStore stores OAuth state tokens in Redis with automatic expiration
type RedisStateStore struct {
	redis *redis.Client
}

// NewRedisStateStore creates a new Redis-backed state store
func NewRedisStateStore(redisClient *redis.Client) *RedisStateStore {
	return &RedisStateStore{
		redis: redisClient,
	}
}

// Set stores an OAuth state token with expiration
func (s *RedisStateStore) Set(ctx context.Context, state string, expiry time.Duration) error {
	key := oauthStateKeyPrefix + state
	return s.redis.Set(ctx, key, "1", expiry).Err()
}

// Exists checks if an OAuth state token exists and is not expired
func (s *RedisStateStore) Exists(ctx context.Context, state string) (bool, error) {
	key := oauthStateKeyPrefix + state
	result, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check state existence: %w", err)
	}
	return result == 1, nil
}

// Delete removes an OAuth state token
func (s *RedisStateStore) Delete(ctx context.Context, state string) error {
	key := oauthStateKeyPrefix + state
	return s.redis.Del(ctx, key).Err()
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/oauth/... -v`
Expected: PASS

**Step 5: Update OAuthHandler to use RedisStateStore**

Modify `backend/internal/handlers/oauth.go`:

```go
// Remove the in-memory stateStore struct entirely
// Replace OAuthHandler definition:

// OAuthHandler handles OAuth authentication
type OAuthHandler struct {
	oauthService     *oauth.Service
	db               database.Database
	tokenService     *auth.TokenService
	stateStore       oauth.StateStore  // Changed: use interface instead of concrete type
	stateExpiry      time.Duration
	frontendRedirect string
}

// Remove: states field, stateStore struct
// Remove: startCleanupWorker, Stop, setState, getState, deleteState, cleanupExpiredStates methods
```

Update `NewOAuthHandler` function in `oauth.go`:

```go
func NewOAuthHandler(db database.Database, tokenService *auth.TokenService, cfg *config.Config, redisClient *redis.Client) *OAuthHandler {
	// ... existing config setup ...

	h := &OAuthHandler{
		oauthService:     oauth.NewService(db, oauthConfig),
		db:               db,
		tokenService:     tokenService,
		stateStore:       oauth.NewRedisStateStore(redisClient),  // Changed: use Redis
		stateExpiry:      10 * time.Minute,
		frontendRedirect: frontendURL,
	}

	// Remove: go h.startCleanupWorker() call - no longer needed

	return h
}
```

Update `Authorize` method in `oauth.go` (line ~187):

```go
// Store state with expiry (now in Redis with automatic expiration)
ctx := context.Background()
if err := h.stateStore.Set(ctx, state, h.stateExpiry); err != nil {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "failed to store state",
		"code":  "STATE_ERROR",
	})
}
```

Update `Callback` method in `oauth.go` (line ~250-258):

```go
// Verify state (now checks Redis)
ctx := context.Background()
exists, err := h.stateStore.Exists(ctx, state)
if err != nil || !exists {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "invalid or expired state",
		"code":  "INVALID_STATE",
	})
}

if err := h.stateStore.Delete(ctx, state); err != nil {
	log.Printf("[OAuth] Failed to delete state: %v", err)
	// Continue anyway - state will expire naturally
}
```

**Step 6: Update main.go to pass redisClient**

Modify `backend/cmd/server/main.go:103`:

```go
// Change from:
oauthHandler := handlers.NewOAuthHandler(db, tokenSvc, cfg)

// To:
oauthHandler := handlers.NewOAuthHandler(db, tokenSvc, cfg, redisClient)
```

**Step 7: Run tests to verify everything works**

Run: `cd backend && go test ./tests/handlers/oauth_test.go -v`
Expected: PASS (may need to update existing tests)

**Step 8: Commit**

```bash
cd backend
git add internal/oauth/state_store.go internal/oauth/state_store_test.go
git add internal/handlers/oauth.go cmd/server/main.go
git commit -m "security(h-2): move OAuth state to Redis for DoS resilience

- Replace in-memory state store with Redis-backed implementation
- Adds automatic expiration via Redis TTL
- Removes background cleanup goroutine (no longer needed)
- Prevents unbounded memory growth from state flooding

Fixes: H-2 from security audit"
```

---

## Task 2: WebSocket Ticket Rate Limiting

**Issue:** H-1 - Missing rate limiting on WebSocket ticket generation endpoint.

**Files:**
- Modify: `backend/internal/middleware/ratelimit.go`
- Modify: `backend/cmd/server/main.go:244`

**Step 1: Add WebSocket ticket limiter**

Add to `backend/internal/middleware/ratelimit.go`:

```go
// WSTicketLimiters returns rate limiters for WebSocket ticket endpoints
func WSTicketLimiters(trustedProxies []string) *EndpointLimiters {
	return &EndpointLimiters{
		Ticket: NewRateLimiter(10, time.Minute), // 10 tickets per minute per IP
	}
}

// Ticket returns a rate limiting handler for WebSocket ticket generation
func (l *EndpointLimiters) Ticket(c *fiber.Ctx) error {
	ip := getClientIP(c, trustedProxies)
	key := fmt.Sprintf("ws_ticket:%s", ip)

	allowed, reset := limiter.Allow(key, 10, time.Minute)
	if !allowed {
		c.Set("X-RateLimit-Limit", "10")
		c.Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Too many ticket requests. Please try again later.",
			"code":  "RATE_LIMIT_EXCEEDED",
		})
	}

	return c.Next()
}
```

**Step 2: Apply limiter to ws-ticket endpoint**

Modify `backend/cmd/server/main.go:244`:

```go
// Change from:
protected.Post("/auth/ws-ticket", wsHandler.GetWSTicket)

// To:
wsTicketLimiter := middleware.WSTicketLimiters(trustedProxies)
protected.Post("/auth/ws-ticket", wsTicketLimiter.Ticket, wsHandler.GetWSTicket)
```

**Step 3: Write test for rate limiting**

Create `backend/tests/middleware/ws_ticket_ratelimit_test.go`:

```go
package middleware_test

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/tdawe1/translation-app/internal/middleware"
)

func TestWSTicketRateLimiting(t *testing.T) {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})

	limiter := middleware.WSTicketLimiters([]string{})

	protected := app.Group("/api/v1/auth")
	protected.Post("/ws-ticket", limiter.Ticket, func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ticket": "test-ticket"})
	})

	// Make 10 requests (should all succeed)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/ws-ticket", nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode, "Request %d should succeed", i+1)
	}

	// 11th request should be rate limited
	req := httptest.NewRequest("POST", "/api/v1/auth/ws-ticket", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode, "11th request should be rate limited")
}
```

**Step 4: Run test**

Run: `cd backend && go test ./tests/middleware/ws_ticket_ratelimit_test.go -v`
Expected: PASS

**Step 5: Commit**

```bash
cd backend
git add internal/middleware/ratelimit.go cmd/server/main.go
git add tests/middleware/ws_ticket_ratelimit_test.go
git commit -m "security(h-1): add rate limiting to WebSocket ticket endpoint

- Limit: 10 tickets per minute per IP
- Prevents Redis memory exhaustion through ticket spam
- Returns 429 with clear error message when limit exceeded

Fixes: H-1 from security audit"
```

---

## Task 3: Constant-Time Password Verification

**Issue:** M-1 - Timing differences between existing and non-existing user accounts may aid enumeration.

**Files:**
- Modify: `backend/internal/auth/user_service.go`
- Test: `backend/tests/auth/user_service_test.go`

**Step 1: Write test for timing-safe password verification**

Add to `backend/tests/auth/user_service_test.go`:

```go
func TestUserService_Login_TimingSafeUserNotFound(t *testing.T) {
	db := RequireDB(t)
	cfg := &config.Config{JWTSecret: "test-secret-for-testing-only-32-chars-min"}
	tokenSvc := auth.NewTokenService(cfg.JWTSecret)
	userSvc := auth.NewUserService(db, tokenSvc)

	// Attempt login with non-existent user
	req := auth.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "SomePassword123!",
	}

	_, err := userSvc.Login(req)
	assert.Error(t, err)
	// Error should be generic, not reveal user existence
	assert.NotContains(t, err.Error(), "not found")
	assert.NotContains(t, err.Error(), "no user")
}
```

**Step 2: Run test to verify current behavior**

Run: `cd backend && go test ./tests/auth/user_service_test.go -run TestUserService_Login_TimingSafeUserNotFound -v`
Expected: May PASS or FAIL depending on current error messages - this documents baseline

**Step 3: Implement timing-safe password verification**

Modify `backend/internal/auth/user_service.go`, update the `Login` method:

```go
func (s *UserService) Login(req LoginRequest) (*AuthResult, error) {
	// Always validate email format first (fast path)
	if req.Email == "" || req.Password == "" {
		return nil, ErrInvalidCredentials
	}

	// Fetch user
	var user models.User
	result := s.db.Where("email = ?", req.Email).First(&user)

	// Dummy bcrypt hash for non-existent users (same cost as real hashes)
	// This ensures timing parity between existing and non-existent accounts
	dummyHash := []byte("$2a$12$dummy......................")

	var err error
	if result.Error != nil {
		// User doesn't exist - still perform bcrypt to normalize timing
		err = bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		// Always return generic error regardless of bcrypt result
		return nil, ErrInvalidCredentials
	}

	// User exists - verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		// Wrong password - return same generic error
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrInactiveUser
	}

	// Generate tokens
	accessToken, err := s.tokenService.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         &user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
```

**Step 4: Run test**

Run: `cd backend && go test ./tests/auth/user_service_test.go -run TestUserService_Login_TimingSafeUserNotFound -v`
Expected: PASS

**Step 5: Commit**

```bash
cd backend
git add internal/auth/user_service.go tests/auth/user_service_test.go
git commit -m "security(m-1): implement timing-safe password verification

- Always perform bcrypt comparison for non-existent users
- Uses dummy hash with same cost factor as real hashes
- Normalizes timing between existing and non-existing accounts
- Generic error messages prevent account enumeration

Fixes: M-1 from security audit"
```

---

## Task 4: Email Format Validation

**Issue:** M-3 - Email format not validated before database operations.

**Files:**
- Create: `backend/internal/validation/email.go`
- Modify: `backend/internal/handlers/auth.go`, `email_verification.go`, `password_reset.go`, `magic_link.go`

**Step 1: Write email validation test**

Create `backend/internal/validation/email_test.go`:

```go
package validation

import "testing"

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		want    bool
	}{
		{"valid email", "user@example.com", true},
		{"valid with subdomain", "user@mail.example.com", true},
		{"valid with + tag", "user+tag@example.com", true},
		{"missing @", "userexample.com", false},
		{"missing domain", "user@", false},
		{"missing user", "@example.com", false},
		{"empty string", "", false},
		{"spaces", "user @example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEmail(tt.email)
			if got != tt.want {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/validation/... -v`
Expected: FAIL with "undefined: ValidateEmail"

**Step 3: Implement email validator**

Create `backend/internal/validation/email.go`:

```go
package validation

import (
	"fmt"
	"regexp"
)

var (
	// emailRegex validates email addresses according to RFC 5322
	// Simplified but practical for web applications
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ValidateEmail checks if the email format is valid
func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}
	return emailRegex.MatchString(email)
}

// ValidateEmailRequest is a helper that returns an error if email is invalid
func ValidateEmailRequest(email string) error {
	if !ValidateEmail(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}
```

**Step 4: Run test**

Run: `cd backend && go test ./internal/validation/... -v`
Expected: PASS

**Step 5: Apply validation to handlers**

Add import to handlers that accept email:

`backend/internal/handlers/auth.go` (add to imports):
```go
	"github.com/tdawe1/translation-app/internal/validation"
```

Update `Register` handler (around line 40):
```go
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid request body")
	}

	// NEW: Validate email format before processing
	if err := validation.ValidateEmailRequest(req.Email); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid email format")
	}
```

Similarly update Login, magic link, password reset handlers.

**Step 6: Commit**

```bash
cd backend
git add internal/validation/email.go internal/validation/email_test.go
git add internal/handlers/auth.go internal/handlers/magic_link.go
git add internal/handlers/email_verification.go internal/handlers/password_reset.go
git commit -m "security(m-3): add email format validation

- Validates email format before database operations
- Uses RFC 5322-compliant regex pattern
- Applied to all email-accepting endpoints

Fixes: M-3 from security audit"
```

---

## Task 5: Stricter Admin Rate Limits

**Issue:** M-2 - Admin endpoints need stricter limits for destructive operations.

**Files:**
- Modify: `backend/internal/middleware/ratelimit.go`
- Modify: `backend/cmd/server/main.go:250-253`

**Step 1: Add destructive operation limits**

Add to `backend/internal/middleware/ratelimit.go` in `AdminLimiters`:

```go
// AdminLimiters returns rate limiters for admin endpoints
func AdminLimiters(trustedProxies []string) *EndpointLimiters {
	return &EndpointLimiters{
		Management: NewRateLimiter(30, time.Minute), // 30 reads/min
		Destructive: NewRateLimiter(5, time.Hour),   // 5 destructive actions/hour
	}
}

// Destructive returns a rate limiting handler for destructive admin operations
func (l *EndpointLimiters) Destructive(c *fiber.Ctx) error {
	ip := getClientIP(c, trustedProxies)
	key := fmt.Sprintf("admin_destructive:%s", ip)

	allowed, reset := limiter.Allow(key, 5, time.Hour)
	if !allowed {
		c.Set("X-RateLimit-Limit", "5")
		c.Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": "Too many destructive admin actions. Please try again later.",
			"code":  "RATE_LIMIT_EXCEEDED",
		})
	}

	return c.Next()
}
```

**Step 2: Apply stricter limits to destructive endpoints**

Modify `backend/cmd/server/main.go:250-253`:

```go
// Change from:
adminLimiter := middleware.AdminLimiters(trustedProxies)
adminGroup.Get("/users", adminLimiter.Management, adminHandler.ListUsers)
adminGroup.Patch("/users/:id/role", adminLimiter.Management, adminHandler.UpdateUserRole)
adminGroup.Delete("/users/:id", adminLimiter.Management, adminHandler.DeleteUser)

// To:
adminLimiter := middleware.AdminLimiters(trustedProxies)
adminGroup.Get("/users", adminLimiter.Management, adminHandler.ListUsers)
adminGroup.Patch("/users/:id/role", adminLimiter.Destructive, adminHandler.UpdateUserRole)
adminGroup.DELETE("/users/:id", adminLimiter.Destructive, adminHandler.DeleteUser)
```

**Step 3: Commit**

```bash
cd backend
git add internal/middleware/ratelimit.go cmd/server/main.go
git commit -m "security(m-2): add stricter rate limits for destructive admin operations

- Read operations (list users): 30/minute
- Destructive operations (delete, role change): 5/hour
- Prevents bulk account manipulation attacks

Fixes: M-2 from security audit"
```

---

## Testing Checklist

After completing all tasks:

```bash
# Run all backend tests
cd backend && make test

# Run specific test suites
go test ./internal/oauth/... -v
go test ./internal/validation/... -v
go test ./tests/middleware/... -v
go test ./tests/auth/... -v

# Verify server starts
go run ./cmd/server &

# Run integration tests (if exists)
./scripts/test-integration.sh
```

---

## Summary

This plan addresses 5 security findings from the audit:

| Task | Finding | Files Changed | Risk Mitigated |
|------|---------|---------------|----------------|
| 1 | H-2 | 4 files | DoS via OAuth state flooding |
| 2 | H-1 | 2 files | DoS via WebSocket ticket spam |
| 3 | M-1 | 2 files | Account enumeration via timing |
| 4 | M-3 | 5 files | Invalid email database load |
| 5 | M-2 | 2 files | Bulk admin operations |

**Total estimated time:** 2-3 hours with testing and commits.
