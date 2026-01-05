# JWT Token Leak in Response Body

**Priority**: P0 (Critical) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The OAuth callback returns the JWT access token in the JSON response body, causing it to be logged in browser history, server logs, and third-party analytics.

## Location

- File: `backend/internal/handlers/oauth.go`
- Lines: 162-166

## Problem

```go
return c.JSON(fiber.Map{
    "access_token": accessToken,  // LEAKED in logs/history
    "user":         user,
})
```

The token is already set in an httpOnly cookie, so including it in the response is:
- Redundant
- Security risk (logs, history, analytics)
- Violates token-handling best practices

## Solution

Return only user data:

```go
return c.JSON(fiber.Map{
    "user": user,
})
```

The frontend should read the token from the cookie (automatically sent with requests).

## Acceptance

- [ ] JWT not returned in any response body
- [ ] Token only accessible via httpOnly cookie
- [ ] Frontend reads from cookie, not response
- [ ] Browser DevTools shows no token in response

## Related

- #006 (Cookie Name Consistency)
