# Bcrypt Cost Too Low

**Priority**: P1 (High) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The bcrypt cost factor is set to 10, which is below current recommendations (12+) for password hashing.

## Location

- File: `backend/internal/password/password.go`
- Where `bcrypt.GenerateFromPassword` is called

## Problem

```go
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
//                                                                               ^^
//                                                                         COST TOO LOW
```

## Impact

- Faster to brute force
- Doesn't account for hardware improvements since 2010
- OWASP recommends minimum 12 for 2024+

## Solution

Increase to 12 (add ~4x computation time):

```go
const BcryptCost = 12  // Up from 10

hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
```

Or make it configurable:

```go
// In config.go
type Config struct {
    // ...
    BcryptCost int  // Default: 12
}

// In password.go
func HashPassword(password string, cost int) (string, error) {
    if cost < 10 || cost > 15 {
        return "", fmt.Errorf("invalid bcrypt cost: %d", cost)
    }
    return bcrypt.GenerateFromPassword([]byte(password), cost)
}
```

## Acceptance

- [ ] Bcrypt cost >= 12
- [ ] Existing passwords rehashed on next login
- [ ] Login timing measured (should be <500ms)
- [ ] Configurable via environment variable

## Related

- #018 (Weak Password Generation)
