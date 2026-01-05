# Verbose Error Messages Expose Internal Details

**Priority**: P3 (Low) | **Status**: Pending | **Assigned**: Unassigned

## Summary

Some error messages expose internal implementation details that could aid attackers.

## Location

- Various handlers

## Problem

```go
return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
    "error": "Failed to generate token: crypto/rand: failed to read random bytes",
    "code":  "TOKEN_GENERATION_FAILED",
})
```

## Impact

- Reveals crypto library in use
- Exposes file paths, stack traces in some errors
- Aids attacker reconnaissance

## Solution

Use generic error messages for user-facing responses:

```go
return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
    "error": "Failed to process request",
    "code":  "INTERNAL_ERROR",
})
```

Log detailed errors internally:

```go
log.Printf("Token generation failed: %v", err) // Full details here
```

## Acceptance

- [ ] User errors are generic
- [ ] Detailed errors logged only
- [ ] No stack traces in responses
- [ ] No internal paths exposed

## Related

- #010 (User Enumeration)
