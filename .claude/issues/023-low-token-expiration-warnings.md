# Add Token Expiration Warnings

**Priority**: P3 (Low) | **Status**: Pending | **Assigned**: Unassigned

## Summary

Email verification and magic link tokens have fixed expiration but users aren't warned when tokens are nearing expiration.

## Location

- File: `backend/internal/handlers/email.go`
- Token expiration logic

## Problem

Tokens expire after a fixed time (e.g., 15 minutes) but:
- Users aren't told the expiration time
- No warning when token is about to expire
- Poor UX if user takes longer than expected

## Solution

Return expiration time in API response:

```go
return c.Status(fiber.StatusOK).JSON(fiber.Map{
    "message": "Verification email sent",
    "expires_at": token.ExpiresAt,  // Add this
    "expires_in_minutes": int(time.Until(token.ExpiresAt).Minutes()),
})
```

Consider refresh mechanism for expiring tokens:

```go
// If token is within 2 minutes of expiration
if time.Until(token.ExpiresAt) < 2*time.Minute {
    // Auto-generate new token and extend
}
```

## Acceptance

- [ ] Token expiration returned in response
- [ ] Frontend displays expiration warning
- [ ] Optional: Auto-refresh for near-expiry tokens

## Related

- #014 (Silent Email Failures)
