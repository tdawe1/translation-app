# Watcher Config Created in Auth Flow

**Priority**: P1 (High) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The OAuth service creates watcher config and state synchronously during user registration, violating separation of concerns and slowing auth flow.

## Location

- File: `backend/internal/oauth/service.go`
- Lines: 212-226

## Problem

```go
// Create watcher config
config := models.WatcherConfig{UserID: user.ID}
if err := tx.Create(&config).Error; err != nil {
    tx.Rollback()
    return nil, fmt.Errorf("failed to create watcher config: %w", err)
}

// Create watcher state
watcherState := models.WatcherState{
    UserID:        user.ID,
    WatcherStatus: "stopped",
}
if err := tx.Create(&watcherState).Error; err != nil {
    tx.Rollback()
    return nil, fmt.Errorf("failed to create watcher state: %w", err)
}
```

## Issues

1. Auth flow knows about watcher domain models
2. Slower OAuth callback (unnecessary DB writes)
3. If watcher creation fails, auth fails (bad UX)
4. Tight coupling between modules

## Solution

Option A: Lazy initialization (recommended)

Create watcher resources on first access:

```go
// In watcher handler GetConfig
func (h *WatcherHandler) GetConfig(c *fiber.Ctx) error {
    userID := c.Locals("user_id").(string)

    var config models.WatcherConfig
    err := h.db.Where("user_id = ?", userID).First(&config).Error

    if err == gorm.ErrRecordNotFound {
        // Create default config on first access
        config = models.WatcherConfig{UserID: userID}
        h.db.Create(&config)
    }
    // ...
}
```

Option B: Async background creation

```go
// In auth flow, just return success
// Queue background job to create watcher resources
go func() {
    // Create with retry logic
    for i := 0; i < 3; i++ {
        err := createWatcherResources(userID)
        if err == nil {
            break
        }
        time.Sleep(time.Second)
    }
}()
```

Option C: Domain event

```go
// Publish UserCreated event
eventBus.Publish("user.created", UserCreatedEvent{UserID: user.ID})

// Watcher service subscribes and creates resources
watcherService.OnUserCreated(func(userID uuid.UUID) {
    // Create watcher config/state
})
```

## Acceptance

- [ ] Auth flow doesn't create watcher resources
- [ ] Watcher resources created on first access or async
- [ ] OAuth callback is faster
- [ ] Failed watcher creation doesn't break auth

## Related

- Architecture improvement (domain events)
