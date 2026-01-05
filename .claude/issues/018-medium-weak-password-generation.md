# Weak Password Generation (Misleading Length Parameter)

**Priority**: P2 (Medium) | **Status**: Pending | **Assigned**: Unassigned

## Summary

`GenerateRandomPassword(32)` only generates 16 unique characters despite the length parameter being 32.

## Location

- File: `backend/internal/password/password.go`
- Lines: `GenerateRandomPassword` function

## Problem

```go
func GenerateRandomPassword(length int) (string, error) {
    b := make([]byte, length)
    rand.Read(b)
    return hex.EncodeToString(b)[:length], nil
    //                          ^^^^^^^^^
    //   hex.EncodeToString doubles length, then we truncate
}

// GenerateRandomPassword(32):
// 1. Create 32 random bytes
// 2. hex.EncodeToString makes 64 hex characters
// 3. Truncate to first 32 characters = 16 actual random bytes!
```

## Impact

- OAuth users have weaker passwords than intended
- Misleading API (parameter doesn't do what it says)
- Security: 16 bytes is still 128 bits of entropy (sufficient), but the bug is concerning

## Solution

Fix the function:

```go
func GenerateRandomPassword(length int) (string, error) {
    // Calculate bytes needed for hex encoding
    byteLength := (length + 1) / 2
    b := make([]byte, byteLength)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b)[:length], nil
}
```

Or better, use base64 for more entropy per character:

```go
func GenerateRandomPassword(length int) (string, error) {
    b := make([]byte, length)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b)[:length], nil
}
```

## Acceptance

- [ ] Function generates `length` bytes of entropy
- [ ] Tests verify output length matches input
- [ ] Tests verify entropy is sufficient
- [ ] OAuth passwords regenerated with new function

## Related

- #011 (Bcrypt Cost)
