# OAuth State Race Condition

**Priority**: P0 (Critical) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The OAuth state store in `oauth.go` uses a raw `map[string]time.Time` without mutex protection, causing race conditions and potential panics under concurrent requests.

## Location

- File: `backend/internal/handlers/oauth.go`
- Lines: 30-31, 100-108

## Problem

```go
type OAuthHandler struct {
    stateStore   map[string]time.Time  // NOT THREAD-SAFE
    stateExpiry  time.Duration
}
```

Multiple goroutines can read/write this map simultaneously, causing:
- Runtime panic: "concurrent map writes"
- Data corruption
- Lost state entries

## Solution

Replace with `sync.RWMutex` or `sync.Map`:

```go
type OAuthHandler struct {
    stateStore   struct {
        sync.RWMutex
        m map[string]time.Time
    }
    stateExpiry  time.Duration
}

func (h *OAuthHandler) getState(key string) (time.Time, bool) {
    h.stateStore.RLock()
    defer h.stateStore.RUnlock()
    val, ok := h.stateStore.m[key]
    return val, ok
}

func (h *OAuthHandler) setState(key string, val time.Time) {
    h.stateStore.Lock()
    defer h.stateStore.Unlock()
    h.stateStore.m[key] = val
}
```

**Better alternative**: Use Redis for state storage with automatic TTL.

## Acceptance

- [ ] State store is thread-safe
- [ ] No concurrent map access warnings in `go run -race`
- [ ] Tests pass under concurrent load

## Related

- #002 (State Store Memory Leak)
- #007 (Redis-backed State Store consideration)
