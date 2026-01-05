# Refactor: OAuth Handler Doing Too Much

**Priority**: P2 (Medium - Tech Debt) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The OAuth handler mixes concerns: HTTP handling, state management, token generation, and user authentication logic.

## Location

- File: `backend/internal/handlers/oauth.go`
- Entire file (~215 lines)

## Problem

The `OAuthHandler` struct and methods handle:
1. HTTP routing (correct)
2. State storage (should be separate)
3. State validation (should be in service)
4. User lookup (already in service, but handler duplicates)

This violates Single Responsibility Principle.

## Solution

Extract to proper layers:

**New: `internal/oauth/statestore.go`**
```go
package oauth

import "context"

type StateStore interface {
    Generate(ctx context.Context) (string, error)
    Validate(ctx context.Context, state string) (bool, error)
    Delete(ctx context.Context, state string) error
}

type RedisStateStore struct {
    client *redis.Client
}

func (s *RedisStateStore) Generate(ctx context.Context) (string, error) {
    state := generateRandomString()
    if err := s.client.Set(ctx, "oauth:state:"+state, "1", 10*time.Minute).Err(); err != nil {
        return "", err
    }
    return state, nil
}

func (s *RedisStateStore) Validate(ctx context.Context, state string) (bool, error) {
    exists, err := s.client.Exists(ctx, "oauth:state:"+state).Result()
    return exists > 0, err
}
```

**Simplify handler:**
```go
type OAuthHandler struct {
    oauthService *oauth.Service
    stateStore   oauth.StateStore  // Interface, not concrete
    tokenService *auth.TokenService
}

func (h *OAuthHandler) Authorize(c *fiber.Ctx) error {
    provider, _ := oauth.ValidateProvider(c.Params("provider"))

    // Generate state via store
    state, err := h.stateStore.Generate(c.Context())
    if err != nil {
        return ErrorResponse(c, 500, "STATE_ERROR", "Failed to generate state")
    }

    // Get auth URL from service
    authURL, err := h.oauthService.GetAuthURL(c.Context(), provider, state)
    if err != nil {
        return ErrorResponse(c, 500, "OAUTH_ERROR", "Failed to get auth URL")
    }

    return c.Redirect(authURL)
}
```

## Acceptance

- [ ] StateStore interface created
- [ ] Redis implementation created
- [ ] Handler only handles HTTP concerns
- [ ] Service layer handles business logic
- [ ] Tests mock StateStore interface

## Related

- #001, #003, #016 (OAuth state issues all solved by this refactor)
