# Duplicate Route Registration

**Priority**: P1 (High) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The GitHub OAuth callback route is registered twice in main.go, causing confusion and potential routing issues.

## Location

- File: `backend/cmd/server/main.go`
- Lines: 137-139

## Problem

```go
oauthGroup.Get("/github/callback", oauthHandler.Callback)
oauthGroup.Get("/github/callback", oauthHandler.Callback)  // DUPLICATE
```

Fiber allows duplicate routes but the last one wins (or behavior is undefined).

## Solution

Remove the duplicate:

```go
oauthGroup.Get("/github/callback", oauthHandler.Callback)
// Delete the duplicate line
```

## Acceptance

- [ ] No duplicate routes in main.go
- [ ] `app routes` command shows no duplicates
- [ ] All OAuth tests still pass

## Related

- N/A
