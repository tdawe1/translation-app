# SameSite Cookie Setting Should Be Strict

**Priority**: P3 (Low) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The SameSite cookie setting is `Lax` but should be `Strict` for authentication cookies to prevent CSRF attacks.

## Location

- File: `backend/internal/handlers/oauth.go`
- Line: 157
- File: `backend/internal/handlers/email.go`
- Line: 229

## Problem

```go
c.Cookie(&fiber.Cookie{
    // ...
    SameSite: "Lax",  // Should be "Strict" for auth cookies
})
```

## Impact

- Lax allows cookies to be sent with top-level navigations
- Strict only sends cookies for same-site requests
- For auth cookies, Strict is more secure

## Solution

Change to `Strict`:

```go
c.Cookie(&fiber.Cookie{
    Name:     "refresh_token",
    Value:    refreshToken,
    // ...
    SameSite: "Strict",  // Changed from "Lax"
})
```

Note: Consider UX implications - Strict may break some OAuth redirect flows. Test thoroughly.

## Acceptance

- [ ] Auth cookies use SameSite=Strict
- [ ] OAuth flows still work
- [ ] CSRF tests verify protection

## Related

- #016 (CSRF Protection)
