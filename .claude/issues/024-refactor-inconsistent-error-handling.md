# Refactor: Inconsistent Error Handling Patterns

**Priority**: P2 (Medium - Tech Debt) | **Status**: Pending | **Assigned**: Unassigned

## Summary

Three different error response patterns exist across handlers, making the codebase harder to maintain.

## Location

- `backend/internal/handlers/auth.go`
- `backend/internal/handlers/oauth.go`
- `backend/internal/handlers/email.go`

## Problem

Pattern 1 (auth.go):
```go
return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
    "error": "Invalid credentials",
    "code": "INVALID_CREDENTIALS",
})
```

Pattern 2 (oauth.go):
```go
return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
    "error": "Invalid state",
    "code":  "INVALID_STATE",
})
```

Pattern 3 (email.go):
```go
return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
    "error": "Invalid or expired verification code",
    "code":  "INVALID_CODE",
    "details": fiber.Map{"field": "code"},  // Extra field!
})
```

## Solution

Create shared error response helper:

```go
// In internal/handlers/response.go (already exists!)
package handlers

import "github.com/gofiber/fiber/v2"

type ErrorCode string

const (
    ErrUnauthorized     ErrorCode = "UNAUTHORIZED"
    ErrInvalidInput     ErrorCode = "INVALID_INPUT"
    ErrInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
    ErrInvalidToken     ErrorCode = "INVALID_TOKEN"
    ErrEmailFailed      ErrorCode = "EMAIL_SEND_FAILED"
    // ... all error codes
)

func ErrorResponse(c *fiber.Ctx, status int, code ErrorCode, message string) error {
    return c.Status(status).JSON(fiber.Map{
        "error": message,
        "code":  string(code),
    })
}

func ValidationError(c *fiber.Ctx, field, message string) error {
    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
        "error": message,
        "code":  "VALIDATION_ERROR",
        "details": fiber.Map{"field": field},
    })
}
```

Use everywhere:

```go
return handlers.ErrorResponse(c, 401, handlers.ErrInvalidCredentials, "Invalid credentials")
```

## Acceptance

- [ ] Shared error response functions created
- [ ] All handlers use shared functions
- [ ] Error codes are constants
- [ ] Tests verify error format consistency

## Related

- #012, #013 (Code duplication)
