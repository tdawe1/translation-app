# Plaintext OAuth Tokens in Logs

**Priority**: P1 (High) | **Status**: Pending | **Assigned**: Unassigned

## Summary

OAuth access tokens are logged in plaintext, exposing credentials that could be used to impersonate users.

## Location

- File: `backend/internal/handlers/oauth.go`
- Wherever tokens are exchanged or used

## Problem

When the OAuth token is exchanged or user info is fetched, the token may be logged via:
- Debug logging
- Error messages
- Fiber middleware logger

## Solution

Ensure tokens are never logged:

```go
// DON'T do this:
log.Printf("Exchanging token: %s", accessToken)  // BAD

// DO this instead:
log.Printf("Exchanging token for user: %s", providerUserID)  // GOOD
```

Add a custom logger middleware that redacts sensitive headers:

```go
func RedactSensitiveHeaders() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Redact Authorization header
        if auth := c.Get("Authorization"); auth != "" {
            c.Locals("auth_header", "***REDACTED***")
        }
        return c.Next()
    }
}
```

## Acceptance

- [ ] No tokens in any log statements
- [ ] Authorization headers redacted in logs
- [ ] Grepping logs finds no "Bearer " patterns

## Related

- #002 (JWT Token Leak)
