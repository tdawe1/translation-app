# Cookie Name Inconsistency Breaks Magic Link Auth

**Priority**: P0 (Critical) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The OAuth handler sets "session_token" cookie, but the email handler sets "refresh_token" cookie. The frontend expects "refresh_token", breaking OAuth login.

## Location

- File: `backend/internal/handlers/oauth.go`
- Line: 154
- File: `backend/internal/handlers/email.go`
- Line: 226

## Problem

```go
// oauth.go:154
c.Cookie(&fiber.Cookie{
    Name:     "session_token",  // WRONG NAME
    // ...
})

// email.go:226
c.Cookie(&fiber.Cookie{
    Name:     "refresh_token",  // EXPECTED NAME
    // ...
})
```

The frontend's API client reads "refresh_token" from cookies and uses it for auth.

## Impact

- OAuth login appears to succeed but user isn't logged in
- Confusing UX (no error message)
- Support tickets from confused users

## Solution

Standardize on "refresh_token" everywhere:

```go
// oauth.go:154 - CHANGE to:
c.Cookie(&fiber.Cookie{
    Name:     "refresh_token",
    Value:    refreshToken,
    // ...
})
```

## Acceptance

- [ ] All auth flows use "refresh_token" cookie name
- [ ] Frontend successfully reads cookie after OAuth
- [ ] Manual test: Google login works end-to-end
- [ ] Manual test: GitHub login works end-to-end

## Related

- #002 (JWT Token Leak)
