# HTTP Client Created Per Request

**Priority**: P0 (Critical) | **Status**: Pending | **Assigned**: Unassigned

## Summary

The email service creates a new HTTP client for every email sent, bypassing connection pooling and increasing latency.

## Location

- File: `backend/internal/email/service.go`
- Lines: 79-84

## Problem

```go
func (s *Service) SendEmail(to, subject, htmlContent string) error {
    client := &http.Client{Timeout: 10 * time.Second}  // NEW CLIENT EVERY TIME
    // ...
}
```

## Impact

- No connection reuse
- Increased latency (~50-100ms per email)
- More file descriptors consumed
- Higher load on target server

## Solution

Use a singleton client with connection pooling:

```go
type Service struct {
    apiKey    string
    fromEmail string
    fromName  string
    httpClient *http.Client  // Reusable client
}

func NewService(cfg *config.Config) *Service {
    return &Service{
        apiKey:    cfg.ResendAPIKey,
        fromEmail: cfg.FromEmail,
        fromName:  cfg.FromName,
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
            },
        },
    }
}

func (s *Service) SendEmail(to, subject, htmlContent string) error {
    // Use s.httpClient instead of creating new one
}
```

## Acceptance

- [ ] HTTP client is a Service field
- [ ] Connection pool configured
- [ ] SendEmail uses shared client
- [ ] Benchmark shows latency improvement

## Related

- #017 (Performance optimization)
