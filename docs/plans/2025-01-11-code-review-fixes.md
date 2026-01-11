# Code Review Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix three issues identified by AI code review: cookie domain mismatch in logout, WebSocket race condition, and untyped JWT claims.

**Architecture:**
- Cookie domain fix: Introduce `SessionConfig` struct to ensure Set/Clear operations use matching cookie attributes (Domain, Secure, SameSite)
- WebSocket fix: Add explicit `isConnectedRef` to track connection state synchronously in React hook
- JWT claims fix: Add typed helper functions (`GetUserRole`, `IsAdmin`) to replace scattered runtime type assertions

**Tech Stack:** Go 1.25, Fiber v2, React 19, TypeScript, Zustand

---

## Task 1: Add Cookie Configuration to Config Struct

**Files:**
- Modify: `backend/internal/config/config.go`

**Step 1: Add cookie fields to Config struct**

Locate the `Config` struct (around line 13) and add the cookie configuration fields:

```go
// Cookies
CookieSecure  bool   // Set Secure flag on cookies
CookieDomain  string // Domain for cookies (empty for localhost, ".example.com" for prod)
CookieSameSite string // SameSite policy: "Lax", "Strict", or "None"
```

**Step 2: Add environment variable loading**

Locate the `Load()` function and add the cookie environment loading (around line 89):

```go
CookieSecure:              getEnv("ENV", "development") == "production",
CookieDomain:              getEnv("COOKIE_DOMAIN", ""),    // Empty = current host only
CookieSameSite:            getEnv("COOKIE_SAMESITE", "Lax"),
```

**Step 3: Verify the code compiles**

Run: `go build ./cmd/server`
Expected: No errors

**Step 4: Commit**

```bash
git add backend/internal/config/config.go
git commit -m "feat(config): add CookieDomain and CookieSameSite configuration

This allows configuring cookie domain and SameSite policy via environment
variables, which is necessary for proper cookie clearing in production."
```

---

## Task 2: Create SessionConfig Pattern for Cookie Management

**Files:**
- Modify: `backend/internal/handlers/response.go`

**Step 1: Remove hardcoded CookieDomain constant**

Remove the old `const CookieDomain = ""` (around line 14) - this is no longer needed.

**Step 2: Add SessionConfig struct**

Add after the constants section (around line 19):

```go
// SessionConfig holds cookie configuration for session management.
// This ensures that SetSessionCookie and ClearSessionCookie use matching
// cookie attributes (domain, secure, sameSite), which is critical for
// proper cookie clearing in production environments.
type SessionConfig struct {
    Domain   string        // Cookie domain (empty for localhost, ".example.com" for prod)
    Secure   bool          // Whether to set the Secure flag (HTTPS only)
    SameSite string        // SameSite policy: "Lax", "Strict", or "None"
    Expires  time.Duration // Cookie expiration duration (for Set only, not Clear)
}
```

**Step 3: Add SessionConfig helper functions**

Add after the SessionConfig struct:

```go
// DefaultSessionConfig returns a SessionConfig with development-friendly defaults.
func DefaultSessionConfig() SessionConfig {
    return SessionConfig{
        Domain:   "",            // Current host only (localhost)
        Secure:   false,         // HTTP in development
        SameSite: "Lax",
        Expires:  DefaultCookieExpiration,
    }
}

// SessionConfigFromEnv creates a SessionConfig from environment-based values.
func SessionConfigFromEnv(domain string, secure bool, sameSite string) SessionConfig {
    return SessionConfig{
        Domain:   domain,
        Secure:   secure,
        SameSite: sameSite,
        Expires:  DefaultCookieExpiration,
    }
}
```

**Step 4: Update SetSessionCookie to use SessionConfig**

Replace the existing `SetSessionCookie` function:

```go
// SetSessionCookie sets the httpOnly session cookie with the given configuration.
func SetSessionCookie(c *fiber.Ctx, token string, config SessionConfig) {
    c.Cookie(&fiber.Cookie{
        Name:     CookieName,
        Value:    token,
        Domain:   config.Domain,
        HTTPOnly: true,
        Secure:   config.Secure,
        SameSite: config.SameSite,
        Expires:  time.Now().Add(config.Expires),
    })
}
```

**Step 5: Update ClearSessionCookie to use SessionConfig**

Replace the existing `ClearSessionCookie` function:

```go
// ClearSessionCookie clears the session cookie.
// IMPORTANT: Must use the same Domain/Secure/SameSite values as when the cookie was set.
func ClearSessionCookie(c *fiber.Ctx, config SessionConfig) {
    c.Cookie(&fiber.Cookie{
        Name:     CookieName,
        Value:    "",
        Domain:   config.Domain,
        HTTPOnly: true,
        Secure:   config.Secure,
        SameSite: config.SameSite,
        Expires:  time.Now().Add(-1 * time.Hour), // Set to past to ensure deletion
    })
}
```

**Step 6: Add convenience functions**

Add helper functions for backward compatibility:

```go
// SetSessionCookieWithDefaults is a convenience function that uses default session config.
func SetSessionCookieWithDefaults(c *fiber.Ctx, token string, secure bool) {
    config := DefaultSessionConfig()
    config.Secure = secure
    SetSessionCookie(c, token, config)
}

// ClearSessionCookieWithDefaults is a convenience function that uses default session config.
func ClearSessionCookieWithDefaults(c *fiber.Ctx, secure bool) {
    config := DefaultSessionConfig()
    config.Secure = secure
    ClearSessionCookie(c, config)
}
```

**Step 7: Verify the code compiles**

Run: `go build ./cmd/server`
Expected: No errors

**Step 8: Commit**

```bash
git add backend/internal/handlers/response.go
git commit -m "refactor(handlers): introduce SessionConfig pattern for cookie management

This fixes a production bug where cookies couldn't be cleared due to domain
mismatch between Set and Clear operations. The SessionConfig ensures
matching attributes."
```

---

## Task 3: Update AuthHandler to Use New Cookie Pattern

**Files:**
- Modify: `backend/internal/handlers/auth.go`

**Step 1: Update Register handler**

Line ~99: Replace `SetSessionCookie(c, result.AccessToken, h.secureCookie)` with:
```go
SetSessionCookieWithDefaults(c, result.AccessToken, h.secureCookie)
```

**Step 2: Update Login handler**

Line ~129: Replace `SetSessionCookie(c, result.AccessToken, h.secureCookie)` with:
```go
SetSessionCookieWithDefaults(c, result.AccessToken, h.secureCookie)
```

**Step 3: Update Logout handler**

Line ~187: Replace `ClearSessionCookie(c, h.secureCookie)` with:
```go
ClearSessionCookieWithDefaults(c, h.secureCookie)
```

**Step 4: Update VerifyMagicLink handler**

Line ~294: Replace `SetSessionCookie(c, accessToken, h.secureCookie)` with:
```go
SetSessionCookieWithDefaults(c, accessToken, h.secureCookie)
```

**Step 5: Verify the code compiles**

Run: `go build ./cmd/server`
Expected: No errors

**Step 6: Commit**

```bash
git add backend/internal/handlers/auth.go
git commit -m "fix(auth): use SessionConfig pattern for all cookie operations

Ensures Set and Clear operations use matching cookie attributes, fixing
logout in production environments with custom cookie domains."
```

---

## Task 4: Fix WebSocket Race Condition

**Files:**
- Modify: `frontend/hooks/use-watcher-websocket.ts`

**Step 1: Add isConnectedRef to track connection state**

After line 96 (reconnectAttemptsRef), add:
```typescript
// Explicit connection state ref to avoid race condition with readyState checks
const isConnectedRef = useRef(false);
```

**Step 2: Set isConnectedRef in onopen handler**

Line ~187: Add inside the `ws.onopen` callback:
```typescript
isConnectedRef.current = true; // Set explicit state
```

**Step 3: Clear isConnectedRef in onclose handler**

Line ~257: Add inside the `ws.onclose` callback:
```typescript
isConnectedRef.current = false; // Clear explicit state
```

**Step 4: Update connected derived value**

Line ~296: Replace the entire line:
```typescript
// Old: const connected = wsRef.current?.readyState === WebSocket.OPEN;
// New:
const connected = isConnectedRef.current;
```

Add comment above:
```typescript
// Use explicit state ref instead of deriving from readyState to avoid race conditions
// The ref is set synchronously in onopen/onclose handlers for consistent state
const connected = isConnectedRef.current;
```

**Step 5: Verify TypeScript compiles**

Run: `npx tsc --noEmit hooks/use-watcher-websocket.ts`
Expected: No errors in this file

**Step 6: Commit**

```bash
git add frontend/hooks/use-watcher-websocket.ts
git commit -m "fix(hooks): avoid WebSocket connection state race condition

Uses explicit isConnectedRef set synchronously in onopen/onclose handlers
instead of deriving from wsRef.current?.readyState which can change between
check and return."
```

---

## Task 5: Add Role Field to TokenClaims Struct

**Files:**
- Modify: `backend/internal/auth/token.go`

**Step 1: Update TokenClaims struct**

Line ~18: Add the `Role` field:
```go
// TokenClaims represents the JWT claims structure
type TokenClaims struct {
    UserID string `json:"sub"`
    Role   string `json:"role"`
    Type   string `json:"type"`
    jwt.RegisteredClaims
}
```

**Step 2: Update ValidateToken to extract role**

Line ~72: Add role extraction:
```go
userID, _ := claims["sub"].(string)
role, _ := claims["role"].(string)
tokenType, _ := claims["type"].(string)

return &TokenClaims{
    UserID: userID,
    Role:   role,
    Type:   tokenType,
}, nil
```

**Step 3: Verify the code compiles**

Run: `go build ./cmd/server`
Expected: No errors

**Step 4: Commit**

```bash
git add backend/internal/auth/token.go
git commit -m "refactor(auth): add Role field to TokenClaims struct

Matches the actual claims generated in GenerateAccessToken which includes
the role field."
```

---

## Task 6: Add Typed JWT Helper Functions

**Files:**
- Modify: `backend/internal/middleware/context.go`

**Step 1: Add GetUserRole helper function**

Add after the `GetUserID` function (after line 31):
```go
// GetUserRole extracts the user role from JWT claims in the Fiber context.
// Returns the role string and true if found, empty string and false otherwise.
func GetUserRole(c *fiber.Ctx) (string, bool) {
    claims := c.Locals("user")
    if claims == nil {
        return "", false
    }

    var role string
    var ok bool

    if mapClaims, typeOK := claims.(map[string]interface{}); typeOK {
        role, ok = mapClaims["role"].(string)
    } else if jwtClaims, typeOK := claims.(jwt.MapClaims); typeOK {
        role, ok = jwtClaims["role"].(string)
    }

    if ok {
        return role, true
    }
    return "", false
}
```

**Step 2: Add IsAdmin helper function**

Add after `GetUserRole`:
```go
// IsAdmin checks if the current user has admin role.
// Returns false if not authenticated or role is not admin.
func IsAdmin(c *fiber.Ctx) bool {
    role, ok := GetUserRole(c)
    return ok && role == "admin"
}
```

**Step 3: Verify the code compiles**

Run: `go build ./cmd/server`
Expected: No errors

**Step 4: Commit**

```bash
git add backend/internal/middleware/context.go
git commit -m "feat(middleware): add typed GetUserRole and IsAdmin helpers

Provides typed alternatives to scattered runtime type assertions for JWT
claims extraction."
```

---

## Task 7: Update RateLimiter to Use Typed Helpers

**Files:**
- Modify: `backend/internal/middleware/ratelimit.go`

**Step 1: Update RoleBasedLimiter KeyGenerator**

Line ~199-207: Replace the type assertions:
```go
KeyGenerator: func(c *fiber.Ctx) string {
    // For authenticated users, use user ID (typed helper)
    if userID, ok := GetUserID(c); ok {
        return "user:" + userID
    }
    // Fallback to IP for unauthenticated
    return "ip:" + getClientIP(c, config.TrustedProxies)
},
```

**Step 2: Update RoleBasedLimiter admin check**

Line ~215-227: Replace the type assertions:
```go
return func(c *fiber.Ctx) error {
    // Check if user is admin using typed helper
    if IsAdmin(c) {
        // Admin: check if unlimited
        if config.AdminMax == 0 {
            // Unlimited - just log and continue
            if userID, ok := GetUserID(c); ok {
                log.Printf("[RateLimit] Admin bypass for user=%s", userID)
            }
            return c.Next()
        }
        // Apply admin limit (not implemented in this version, would need separate limiter)
    }

    // Apply user limiter (IP-based for unauthenticated, user-based for authenticated)
    return userLimiter(c)
}
```

**Step 3: Update AdminLimiters KeyGenerator**

Line ~243-248: Replace the type assertions:
```go
KeyGenerator: func(c *fiber.Ctx) string {
    // Use typed helper to get user ID
    if userID, ok := GetUserID(c); ok {
        return "admin:" + userID
    }
    return "admin-ip:" + getClientIP(c, trustedProxies)
},
```

**Step 4: Verify the code compiles**

Run: `go build ./cmd/server`
Expected: No errors

**Step 5: Run static analysis**

Run: `go vet ./internal/...`
Expected: No warnings

**Step 6: Commit**

```bash
git add backend/internal/middleware/ratelimit.go
git commit -m "refactor(middleware): use typed helpers in rate limiters

Replaces scattered runtime type assertions with GetUserID and IsAdmin
helper functions for cleaner, type-safe code."
```

---

## Task 8: Update .env.example with New Variables

**Files:**
- Modify: `.env.example`

**Step 1: Add cookie configuration variables**

Add to the Environment Variables section:
```bash
# Cookies
COOKIE_SECURE=false    # Set to 'true' in production (HTTPS required)
COOKIE_DOMAIN=         # Cookie domain (empty for localhost, ".example.com" for prod)
COOKIE_SAMESITE=Lax    # SameSite policy: "Lax", "Strict", or "None"
```

**Step 2: Commit**

```bash
git add .env.example
git commit -m "docs: add COOKIE_* environment variables to example

Documents the new cookie configuration options for production deployments."
```

---

## Testing Checklist

After implementation, verify:

- [ ] Backend compiles: `go build ./cmd/server`
- [ ] Backend static analysis passes: `go vet ./internal/...`
- [ ] Config tests pass: `go test ./internal/config/...`
- [ ] Frontend TypeScript compiles: `npx tsc --noEmit`
- [ ] Manual test: Login → Logout → Verify cookie is cleared (check browser dev tools)
- [ ] Manual test: WebSocket `connected` state updates correctly in React DevTools
- [ ] Manual test: Rate limiter allows admin bypass when role=admin in JWT

---

## Notes

- Integration tests require PostgreSQL running with `gengowatcher_test` database
- Frontend `@types/pg` error is pre-existing and unrelated to these changes
- All changes follow existing code patterns and maintain backward compatibility
