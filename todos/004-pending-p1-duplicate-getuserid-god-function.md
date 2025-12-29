---
status: resolved
priority: p1
issue_id: "004"
tags:
  - architecture
  - code-quality
  - duplication
  - code-review
dependencies: []
---

# P1: Duplicate GetUserID Function - God Function Anti-Pattern

## Problem Statement

The `GetUserID` function is defined twice with nearly identical logic but different implementations. This creates a maintenance nightmare and potential bugs.

**Files**:
- `backend/internal/middleware/jwt.go:172-189`
- `backend/internal/handlers/auth.go:139-155`

## Findings

### The Duplicate

**Version 1** (`jwt.go`):
```go
var GetUserID = func(c *fiber.Ctx) (string, bool) {
    claims := c.Locals("user")
    if claims == nil {
        return "", false
    }
    userClaims, ok := claims.(jwt.MapClaims)
    if !ok {
        return "", false
    }
    if sub, ok := userClaims["sub"].(string); ok {
        return sub, true
    }
    return "", false
}
```

**Version 2** (`auth.go`):
```go
var GetUserID = func(c *fiber.Ctx) (string, bool) {
    claims := c.Locals("user")
    if claims == nil {
        return "", false
    }
    userClaims, ok := claims.(map[string]interface{})
    if !ok {
        return "", false
    }
    if sub, ok := userClaims["sub"].(string); ok {
        return sub, true
    }
    return "", false
}
```

### The Critical Difference

- jwt.go version casts to `jwt.MapClaims`
- auth.go version casts to `map[string]interface{}`
- Both return `(string, bool)` tuple
- Same function name, different behavior

### Impact
- **Type safety violation** - Same function signature, different behavior
- **Maintenance nightmare** - Changes must be synced manually
- **Bug potential** - Handlers importing the wrong version get different behavior
- **Confidence**: 100/100 - Clear duplicate with evidence

## Proposed Solutions

### Option 1: Single Shared Location (Recommended)

Create a new file `internal/middleware/context.go`:

```go
package middleware

import (
    "github.com/gofiber/fiber/v2"
)

// GetUserID extracts the authenticated user ID from the Fiber context.
// Returns the user ID and true if found, empty string and false otherwise.
func GetUserID(c *fiber.Ctx) (string, bool) {
    claims := c.Locals("user")
    if claims == nil {
        return "", false
    }

    // Handle both jwt.MapClaims and map[string]interface{}
    var sub string
    var ok bool

    if mapClaims, typeOK := claims.(map[string]interface{}); typeOK {
        sub, ok = mapClaims["sub"].(string)
    } else if jwtClaims, typeOK := claims.(jwt.MapClaims); typeOK {
        sub, ok = jwtClaims["sub"].(string)
    }

    if ok {
        return sub, true
    }
    return "", false
}
```

**Pros**:
- Single source of truth
- Handles both claim types gracefully
- Clear documentation
- Can be imported by both middleware and handlers

**Cons**:
- Need to update imports in multiple files
- Need to remove both existing definitions

**Effort**: Small
**Risk**: Low

### Option 2: Context Package

Create a dedicated context package:

```go
// package context
type ContextKey string

const (
    UserKey ContextKey = "user"
    UserIDKey ContextKey = "user_id"
)

// GetUserID extracts user ID from context
func GetUserID(c *fiber.Ctx) (string, bool) {
    if userID := c.Locals(string(UserIDKey)); userID != nil {
        if id, ok := userID.(string); ok {
            return id, true
        }
    }
    return "", false
}

// SetUserID stores user ID in context
func SetUserID(c *fiber.Ctx, userID string) {
    c.Locals(string(UserIDKey), userID)
}
```

Update middleware to store user ID directly:

```go
// In JWT middleware - after validation
c.Locals("user_id", sub)
```

**Pros**:
- Type-safe context keys
- No need to extract from claims on each request
- Faster access (no map lookups)
- Cleaner handler code

**Cons**:
- Requires changing middleware implementation
- Need to update all handlers that use GetUserID

**Effort**: Medium
**Risk**: Low

### Option 3: Remove Function, Inline Logic

Don't provide a helper function at all. Each handler extracts what it needs:

```go
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
    claims := c.Locals("user")
    if claims == nil {
        return RespondWithError(c, 401, ErrUnauthorized, "Not authenticated")
    }

    userClaims, ok := claims.(map[string]interface{})
    if !ok {
        return RespondWithError(c, 401, ErrUnauthorized, "Invalid token")
    }

    userID, ok := userClaims["sub"].(string)
    if !ok {
        return RespondWithError(c, 401, ErrUnauthorized, "Invalid token")
    }

    // ... rest of handler
}
```

**Pros**:
- No shared mutable state
- Each handler owns its logic
- No import confusion

**Cons**:
- Significant code duplication
- More verbose handlers
- Harder to test

**Effort**: Large
**Risk**: Medium (more code to maintain)

## Recommended Action

**Implement Option 1** - Create shared function in `internal/middleware/context.go`.

## Technical Details

### Affected Files
- `backend/internal/middleware/jwt.go:172-189` (delete after move)
- `backend/internal/handlers/auth.go:139-155` (delete after move)
- `backend/internal/middleware/context.go` (new file)
- All files importing `GetUserID` need update

### Components
- JWT middleware
- Auth handlers
- Context management

### Database Changes
None

## Acceptance Criteria

- [ ] New `internal/middleware/context.go` file created
- [ ] Both old `GetUserID` definitions removed
- [ ] All imports updated to use shared version
- [ ] Handles both `jwt.MapClaims` and `map[string]interface{}`
- [ ] Tests verify both claim types work
- [ ] No regressions in authentication flow
- [ ] Documentation updated

## Work Log

### 2025-12-29
- **Finding**: Code pattern analysis identified duplicate GetUserID
- **Analysis**: Confirmed 100% - two different implementations
- **Decision**: Selected shared location approach
- **Status**: Pending implementation

## Resources

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Shotgun Surgery](https://refactoring.guru/smells/shotgun-surgery/)
