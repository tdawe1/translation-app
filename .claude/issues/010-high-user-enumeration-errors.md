# User Enumeration via Error Messages

**Priority**: P1 (High) | **Status**: Pending | **Assigned**: Unassigned

## Summary

Error messages reveal whether an email/username exists, enabling attackers to enumerate valid user accounts.

## Location

- File: `backend/internal/handlers/auth.go`
- File: `backend/internal/handlers/email.go`

## Problem

```go
// Different error messages reveal if user exists
if userExists {
    return errors.New("email already registered")  // REVEALS EXISTENCE
}
if !userExists {
    return errors.New("user not found")  // REVEALS NON-EXISTENCE
}
```

## Impact

- Attackers can harvest valid email addresses
- Privacy violation (GDPR Article 25)
- Targeted phishing attacks

## Solution

Use consistent error messages:

```go
// ALWAYS return the same message
return errors.New("if the email exists, a verification link has been sent")

// For login:
return errors.New("invalid credentials")  // Don't specify which field is wrong
```

Log the actual error internally for debugging:

```go
log.Printf("Login attempt for non-existent user: %s", email)
return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
    "error": "invalid credentials",
    "code":  "INVALID_CREDENTIALS",
})
```

## Acceptance

- [ ] All auth errors return generic messages
- [ ] Actual errors logged separately
- [ ] Tests verify message consistency
- [ ] Documentation updated

## Related

- #009 (Rate Limiting)
