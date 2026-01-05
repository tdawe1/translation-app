# Token Generation Duplicated 4x

**Priority**: P1 (High) | **Status**: Pending | **Assigned**: Unassigned

## Summary

Secure token generation logic is duplicated in 4 separate locations (email verification, magic link, password reset, OAuth state).

## Location

1. `backend/internal/handlers/email.go:84-90` - Email verification
2. `backend/internal/handlers/email.go:211-216` - Magic link
3. `backend/internal/handlers/email.go:376-381` - Password reset
4. `backend/internal/oauth/service.go:87-93` - OAuth state

## Problem

Each location has its own `generateSecureToken()` function with slight variations:

```go
// email.go:84
token, err := generateSecureToken()
if err != nil {
    return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
        "error": "Failed to generate token",
        "code":  "TOKEN_GENERATION_FAILED",
    })
}
```

## Impact

- 4x maintenance burden
- Risk of inconsistency (different token lengths, character sets)
- Security bugs could be fixed in one place but not others

## Solution

Create shared token generator:

```go
// In internal/crypto/token.go
package crypto

import (
    "crypto/rand"
    "encoding/base64"
    "errors"
    "fmt"
)

var (
    ErrTokenGeneration = errors.New("failed to generate token")
)

// GenerateSecureToken creates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
    b := make([]byte, length)
    if _, err := rand.Read(b); err != nil {
        return "", fmt.Errorf("%w: %v", ErrTokenGeneration, err)
    }
    return base64.URLEncoding.EncodeToString(b)[:length], nil
}

// Convenience functions for common token lengths
func GenerateEmailToken() (string, error) {
    return GenerateSecureToken(32)
}

func GenerateOAuthState() (string, error) {
    return GenerateSecureToken(32)
}
```

Replace all duplicates:

```go
import "github.com/tdawe1/translation-app/internal/crypto"

token, err := crypto.GenerateEmailToken()
if err != nil {
    // handle error
}
```

## Acceptance

- [ ] Single crypto/token package created
- [ ] All 4 locations import and use shared function
- [ ] Tests verify token properties (length, uniqueness, character set)
- [ ] Old generateSecureToken() functions removed

## Related

- #012 (Provider Validation Duplication)
