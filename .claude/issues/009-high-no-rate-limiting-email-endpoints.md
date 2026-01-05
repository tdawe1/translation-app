# No Rate Limiting on Email Endpoints

**Priority**: P1 (High) | **Status**: Pending | **Assigned**: Unassigned

## Summary

Email endpoints (magic link, password reset, verification) lack rate limiting, allowing attackers to flood user inboxes or abuse the email service.

## Location

- File: `backend/cmd/server/main.go`
- Lines: 127-132

## Problem

```go
authGroup.Post("/verify-email/send", emailHandler.SendVerificationEmail)
authGroup.Post("/verify-email", emailHandler.VerifyEmail)
authGroup.Post("/magic-link/send", emailHandler.SendMagicLink)
authGroup.Post("/magic-link/verify", emailHandler.VerifyMagicLink)
authGroup.Post("/password-reset/send", emailHandler.SendPasswordReset)
authGroup.Post("/password-reset", emailHandler.ResetPassword)
```

No rate limiter middleware applied.

## Impact

- Email spam attack
- Resend API quota exhaustion
- Increased costs
- Poor UX for legitimate users

## Solution

Add rate limiting middleware (already exists for auth endpoints):

```go
import "github.com/tdawe1/translation-app/internal/middleware"

emailLimiter := middleware.EmailLimiters()  // Create if doesn't exist
authGroup.Post("/verify-email/send", emailLimiter.SendVerification, emailHandler.SendVerificationEmail)
authGroup.Post("/magic-link/send", emailLimiter.SendMagicLink, emailHandler.SendMagicLink)
authGroup.Post("/password-reset/send", emailLimiter.SendPasswordReset, emailHandler.SendPasswordReset)
```

Create email limiters in `internal/middleware/rate_limiter.go`:

```go
// 3 emails per hour per email
func EmailLimiters() *RateLimiterConfig {
    return &RateLimiterConfig{
        SendVerification: NewEmailLimiter(3, time.Hour),
        SendMagicLink:    NewEmailLimiter(3, time.Hour),
        SendPasswordReset: NewEmailLimiter(3, time.Hour),
    }
}
```

## Acceptance

- [ ] All email endpoints have rate limiting
- [ ] Limit: 3 requests per hour per email
- [ ] Tests verify rate limit enforced
- [ ] Rate limit headers returned (X-RateLimit-*)

## Related

- #010 (User Enumeration)
