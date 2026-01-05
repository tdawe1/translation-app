# Timing Attack on Email Verification

**Priority**: P2 (Medium) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The email verification endpoint may leak information via timing differences between valid and invalid tokens.

## Location

- File: `backend/internal/handlers/email.go`
- Lines: 147-177 (VerifyEmail function)

## Problem

```go
var token models.EmailVerificationToken
if err := h.db.Where("token = ? AND expires_at > ?", req.Token, time.Now()).
    First(&token).Error; err != nil {
    // Early return for invalid token
    return c.Status(fiber.StatusBadRequest).JSON(/* ... */)
}

// More work for valid tokens (user lookup, update, etc)
```

An attacker can measure response times to determine:
- If a token format is valid
- If a token exists (DB lookup timing)
- If a token is expired

## Solution

Use constant-time comparison for tokens:

```go
import "crypto/subtle"

func (h *EmailHandler) VerifyEmail(c *fiber.Ctx) error {
    var req VerifyEmailRequest
    // ... parse request ...

    // Always fetch token (even if format is invalid for timing)
    var token models.EmailVerificationToken
    result := h.db.Where("token = ?", req.Token).First(&token)

    // Use constant-time string comparison
    isValid := result.Error == nil &&
        subtle.ConstantTimeCompare([]byte(token.Token), []byte(req.Token)) == 1 &&
        token.ExpiresAt.After(time.Now())

    if !isValid {
        // Still do a fake verification work to normalize timing
        h.db.Where("id = ?", uuid.Nil).First(&models.User{})
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid or expired token",
            "code":  "INVALID_TOKEN",
        })
    }

    // ... proceed with verification ...
}
```

## Acceptance

- [ ] Constant-time comparison used
- [ ] Timing tests show <5ms variance
- [ ] Error messages are identical
- [ ] Fake work done on failure path

## Related

- #010 (User Enumeration)
