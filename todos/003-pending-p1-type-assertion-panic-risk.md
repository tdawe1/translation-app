---
status: resolved
priority: p1
issue_id: "003"
tags:
  - reliability
  - error-handling
  - type-safety
  - code-review
dependencies: []
---

# P1: Unsafe Type Assertions Cause Panic Risk

## Problem Statement

Multiple locations use type assertions without checking the `ok` return value. If the error is not the expected type, the application will panic instead of returning a proper error response.

**Files**:
- `backend/internal/handlers/auth.go:54,55,80,81,107`
- `backend/internal/handlers/watcher.go` (similar patterns)

```go
status := h.statusCodeForError(apiErr.(*apperrors.APIError).Code)
return RespondWithAPIError(c, status, apiErr.(*apperrors.APIError))
```

## Findings

### Impact
- **Severity**: CRITICAL - Causes service crash on unexpected error types
- **Confidence**: 90/100 - Pattern confirmed across multiple handlers
- **Locations**: At least 5 occurrences in auth handler

### What Happens
1. An unexpected error type occurs (not `*apperrors.APIError`)
2. Type assertion `apiErr.(*apperrors.APIError)` returns `nil, false`
3. Code tries to access `.Code` on `nil` → **PANIC**
4. Entire handler goroutine crashes
5. User receives 500 error instead of proper error message

### Evidence
- `internal/handlers/auth.go:54` - No ok check before accessing `.Code`
- `internal/handlers/auth.go:80` - Duplicate pattern
- `internal/handlers/auth.go:107` - Duplicate pattern
- Assumes perfect type flow - no defensive programming

## Proposed Solutions

### Option 1: Safe Type Assertion (Recommended)

```go
func (h *AuthHandler) Register(c *fiber.Ctx) error {
    // ... parse request ...

    result, err := h.userService.Register(req)
    if err != nil {
        // Safe type assertion
        if apiErr, ok := err.(*apperrors.APIError); ok {
            status := h.statusCodeForError(apiErr.Code)
            return RespondWithAPIError(c, status, apiErr)
        }
        // Fallback for unexpected error types
        return RespondWithError(c, fiber.StatusInternalServerError,
            apperrors.ErrInternal, "An unexpected error occurred")
    }
    // ...
}
```

**Pros**:
- Prevents panic on unexpected error types
- Graceful degradation for unknown errors
- Clear error handling path
- Maintains existing error response structure

**Cons**:
- More verbose code
- Need to repeat pattern in each handler

**Effort**: Small
**Risk**: Low

### Option 2: Centralize Error Handling

Create a helper function that handles all errors safely:

```go
// In handlers/response.go
func HandleError(c *fiber.Ctx, err error) error {
    if err == nil {
        return nil
    }

    // Check if it's our API error
    if apiErr, ok := err.(*apperrors.APIError); ok {
        status := statusCodeForError(apiErr.Code)
        return RespondWithAPIError(c, status, apiErr)
    }

    // Check for other known error types
    if validationErr, ok := err.(validation.Errors); ok {
        return RespondWithError(c, 400, "VALIDATION_ERROR", validationErr.Error())
    }

    // Generic fallback
    log.Printf("Unexpected error: %T: %v", err, err)
    return RespondWithError(c, 500, apperrors.ErrInternal, "An unexpected error occurred")
}
```

**Pros**:
- Single place to update error handling
- Consistent error responses across all handlers
- Can add logging/metrics in one place
- Less code duplication

**Cons**:
- Requires refactoring all error handling
- Need to ensure all handlers use the helper

**Effort**: Medium
**Risk**: Low

### Option 3: Error Interface Method

Add a method to errors that returns HTTP status:

```go
type HTTPError interface {
    HTTPStatus() int
    Code() string
    Message() string
}

func (e *APIError) HTTPStatus() int {
    return statusCodeForError(e.Code)
}

func (e *APIError) Code() string {
    return string(e.Code)
}

func (e *APIError) Message() string {
    return e.Message
}

// In handlers:
if httpErr, ok := err.(HTTPError); ok {
    return c.Status(httpErr.HTTPStatus()).JSON(fiber.Map{
        "error": httpErr.Message(),
        "code": httpErr.Code(),
    })
}
```

**Pros**:
- Clean interface-based approach
- Extensible to new error types
- Type-safe

**Cons**:
- Requires modifying error types
- More architectural change

**Effort**: Medium
**Risk**: Low

## Recommended Action

**Implement Option 2** - Centralize error handling in a helper function.

## Technical Details

### Affected Files
- `backend/internal/handlers/auth.go:54,80,107`
- `backend/internal/handlers/watcher.go` (likely similar issues)
- `backend/internal/handlers/response.go` (add helper function)
- `backend/internal/handlers/lemonsqueezy.go` (may have similar issues)

### Components
- All HTTP handlers
- Error response formatting

### Database Changes
None

## Acceptance Criteria

- [ ] No unsafe type assertions without `ok` check
- [ ] Centralized `HandleError()` function created
- [ ] All handlers use the error helper
- [ ] Tests verify panic doesn't occur on unexpected errors
- [ ] Unexpected errors return 500 with generic message
- [ ] Structured logging for unexpected errors
- [ ] All handlers reviewed for similar patterns

## Work Log

### 2025-12-29
- **Finding**: Code review identified unsafe type assertions
- **Analysis**: Confirmed 5+ instances without ok check
- **Decision**: Selected centralized error handling approach
- **Status**: Pending implementation

## Resources

- [Go Type Assertions](https://go.dev/tour/methods/16)
- [Error Handling in Go](https://go.dev/doc/effective_go#recover)
- [Don't Panic](https://github.com/golang/go/wiki/CodeReviewComments#don't-panic)
