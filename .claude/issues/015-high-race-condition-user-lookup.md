# Race Condition in User Lookup During OAuth

**Priority**: P1 (High) | **Status**: Pending | **Assigned**: Unassigned

## Summary

During OAuth login, there's a race condition where two simultaneous requests from the same OAuth provider can create duplicate users.

## Location

- File: `backend/internal/oauth/service.go`
- Lines: 149-168

## Problem

```go
// Check if user exists
var user models.User
err = s.db.Where("email = ?", userInfo.Email).First(&user).Error

if err == gorm.ErrRecordNotFound {
    // Create new user
    // ... but another request might do the same simultaneously!
}
```

## Scenario

1. User authenticates with Google twice (e.g., two browser tabs)
2. Both requests check for existing user simultaneously
3. Both don't find the user
4. Both create a new user with the same email
5. Second one fails with unique constraint violation (or worse, succeeds with corrupted data)

## Solution

Use database constraints with proper error handling:

```go
// Ensure unique constraint on email
// models/user.go should have:
type User struct {
    // ...
    Email string `gorm:"uniqueIndex"`
}

// In the service, handle the race:
err = s.db.Where("email = ?", userInfo.Email).First(&user).Error
if err == gorm.ErrRecordNotFound {
    // Try to create, handling duplicate error
    user = models.User{Email: userInfo.Email, /* ... */}
    if err := s.db.Create(&user).Error; err != nil {
        if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
            // Race happened - retry the fetch
            err = s.db.Where("email = ?", userInfo.Email).First(&user).Error
            if err != nil {
                return nil, err
            }
        } else {
            return nil, err
        }
    }
}
```

Better: Use `FirstOrCreate`:

```go
result := s.db.Where("email = ?", userInfo.Email).
    Assign(models.User{
        EmailVerified: userInfo.Verified,
        // ... other fields that should be updated
    }).
    FirstOrCreate(&user)
```

## Acceptance

- [ ] Unique constraint exists on users.email
- [ ] Race condition handled gracefully
- [ ] Concurrent OAuth tests pass
- [ ] No duplicate users created under load

## Related

- #004 (Missing DB Indexes - unique index solves this too)
