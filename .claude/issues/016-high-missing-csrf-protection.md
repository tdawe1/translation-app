# Missing CSRF Protection on OAuth

**Priority**: P1 (High) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The OAuth flow uses state parameter but doesn't verify it comes from the same user session, enabling CSRF attacks.

## Location

- File: `backend/internal/handlers/oauth.go`
- Lines: 75-108 (authorize), 119-145 (callback)

## Problem

```go
// Authorize: Generate state
state, _ := oauth.GenerateState()
h.stateStore[state] = time.Now().Add(h.stateExpiry)
return c.Redirect(authURL)

// Callback: Verify state exists
storedExpiry, exists := h.stateStore[state]
if !exists || time.Now().After(storedExpiry) {
    return error
}
```

The state is generated and stored but NOT tied to the user session. An attacker could:
1. Get their own valid state
2. Trick victim into using that state
3. Hijack the OAuth callback

## Solution

Bind state to user session:

```go
// When generating state, include session identifier
sessionID := c.Cookies("session_id") // or from JWT claim
if sessionID == "" {
    sessionID = uuid.New().String()
    c.Cookie(&fiber.Cookie{Name: "session_id", Value: sessionID})
}

state := fmt.Sprintf("%s:%s", sessionID, randomString)
h.stateStore[state] = time.Now().Add(h.stateExpiry)

// When verifying, check session matches
parts := strings.Split(state, ":")
if len(parts) != 2 || parts[0] != sessionID {
    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
        "error": "Invalid state",
        "code":  "INVALID_STATE",
    })
}
```

Or store state in Redis with user binding:

```go
redisKey := fmt.Sprintf("oauth:state:%s:%s", userID, state)
redisClient.Set(ctx, redisKey, "1", 10*time.Minute)
```

## Acceptance

- [ ] State bound to user session
- [ ] Cannot reuse state across sessions
- [ ] CSRF attack tests fail
- [ ] Documentation updated

## Related

- #001 (OAuth State Race Condition)
- #003 (State Store Memory Leak - Redis solves both)
