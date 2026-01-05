# State Store Memory Leak

**Priority**: P0 (Critical) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The OAuth state store never removes expired entries, causing unbounded memory growth over time.

## Location

- File: `backend/internal/handlers/oauth.go`
- Lines: 30-31, 100-119

## Problem

```go
func (h *OAuthHandler) Authorize(c *fiber.Ctx) error {
    state, _ := oauth.GenerateState()
    h.stateStore[state] = time.Now().Add(h.stateExpiry)  // NEVER CLEANED UP
    // ...
}
```

Every OAuth authorization adds an entry. Expired entries accumulate forever.

## Solution

Option A: Background cleanup goroutine

```go
func (h *OAuthHandler) startCleanupWorker() {
    ticker := time.NewTicker(time.Minute)
    go func() {
        for range ticker.C {
            h.cleanupExpiredStates()
        }
    }()
}

func (h *OAuthHandler) cleanupExpiredStates() {
    now := time.Now()
    h.stateStore.Lock()
    defer h.stateStore.Unlock()
    for key, expiry := range h.stateStore.m {
        if now.After(expiry) {
            delete(h.stateStore.m, key)
        }
    }
}
```

Option B: Use Redis with TTL (recommended for production)

```go
redisClient.Set(ctx, "oauth:"+state, "1", 10*time.Minute)
```

## Acceptance

- [ ] Expired states are automatically removed
- [ ] Memory usage stays bounded under load
- [ ] Cleanup runs on configurable interval
- [ ] Tests verify memory doesn't grow unbounded

## Related

- #001 (OAuth State Race Condition)
- #007 (Redis-backed State Store)
