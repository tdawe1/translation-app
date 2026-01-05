# Missing Input Validation on OAuth State

**Priority**: P2 (Medium) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The OAuth callback doesn't validate the state parameter format before using it as a map key.

## Location

- File: `backend/internal/handlers/oauth.go`
- Lines: 123-129

## Problem

```go
state := c.Query("state")
if state == "" {
    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
        "error": "State parameter is required",
        "code":  "MISSING_STATE",
    })
}

// No format validation - could be arbitrarily long string
storedExpiry, exists := h.stateStore[state]  // DoS vector?
```

## Impact

- Potential DoS via huge state strings
- No validation that state matches expected format
- Memory exhaustion if attacker sends massive strings

## Solution

Validate state format:

```go
const maxStateLength = 128

func isValidState(state string) bool {
    if len(state) > maxStateLength || len(state) < 16 {
        return false
    }
    // State should be base64-encoded (after GenerateState)
    for _, c := range state {
        if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
            (c >= '0' && c <= '9') || c == '-' || c == '_') {
            return false
        }
    }
    return true
}

// In callback handler
state := c.Query("state")
if !isValidState(state) {
    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
        "error": "Invalid state format",
        "code":  "INVALID_STATE_FORMAT",
    })
}
```

## Acceptance

- [ ] State format validated before use
- [ ] Max length enforced
- [ ] Character set validated
- [ ] Tests with invalid states fail appropriately

## Related

- #001 (OAuth State Race Condition)
- #016 (CSRF Protection)
