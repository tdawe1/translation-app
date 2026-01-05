# Silent Email Failures

**Priority**: P1 (High) | **Status**: Pending | **Assigned**: Unassigned

## Summary

When email sending fails, the error is only printed to stdout and the request succeeds. Users think an email was sent when it wasn't.

## Location

- File: `backend/internal/handlers/email.go`
- Lines: 109-112, 236-239, 401-404

## Problem

```go
if err := h.emailService.SendVerificationEmail(req.Email, token); err != nil {
    fmt.Printf("Failed to send verification email: %v\n", err)
    // Don't fail the request if email fails in dev
}
return c.Status(fiber.StatusOK).JSON(fiber.Map{
    "message": "Verification email sent",
})
```

## Impact

- Users wait for email that never comes
- No way to retry (request "succeeded")
- Poor UX and support burden
- Dev comment suggests this is intentional but it's wrong

## Solution

Option A: Always fail the request (recommended):

```go
if err := h.emailService.SendVerificationEmail(req.Email, token); err != nil {
    log.Printf("Failed to send verification email: %v", err)
    return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
        "error": "Failed to send email",
        "code":  "EMAIL_SEND_FAILED",
    })
}
```

Option B: Retry with exponential backoff:

```go
const maxRetries = 3

for i := 0; i < maxRetries; i++ {
    err := h.emailService.SendVerificationEmail(req.Email, token)
    if err == nil {
        break
    }
    if i == maxRetries-1 {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to send email after retries",
            "code":  "EMAIL_SEND_FAILED",
        })
    }
    time.Sleep(time.Duration(1<<i) * time.Second)
}
```

## Acceptance

- [ ] Email failures return error to client
- [ ] Error logged with proper context
- [ ] Client can display retry option
- [ ] Optional: Retry logic implemented

## Related

- #009 (Rate Limiting - helps prevent quota exhaustion)
