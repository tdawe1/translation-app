# Provider Validation Duplicated 3x

**Priority**: P1 (High) | **Status**: Pending | **Assigned**: Unassigned

## Summary

OAuth provider validation logic exists in 3 separate locations, creating maintenance burden and risk of inconsistency.

## Location

1. `backend/internal/handlers/oauth.go:77-91` - Authorize handler
2. `backend/internal/handlers/oauth.go:123-129` - Callback handler (partial)
3. `backend/internal/oauth/service.go:100-106` - Service layer

## Problem

```go
// Location 1: oauth.go:77
provider, err := getProviderFromString(c.Params("provider"))
// ...

// Location 2: oauth.go:123
provider, err := getProviderFromString(c.Params("provider"))
// ...

// Location 3: oauth/service.go:100
switch provider {
case ProviderGoogle:
    config = s.GoogleConfig()
case ProviderGitHub:
    config = s.GitHubConfig()
// ...
}
```

## Solution

Extract to shared validation function:

```go
// In internal/oauth/validators.go
package oauth

import "fmt"

func ValidateProvider(provider string) (Provider, error) {
    p := Provider(strings.ToLower(provider))
    switch p {
    case ProviderGoogle, ProviderGitHub:
        return p, nil
    default:
        return "", fmt.Errorf("unsupported provider: %s", provider)
    }
}
```

Use everywhere:

```go
// Handler
provider, err := oauth.ValidateProvider(c.Params("provider"))
if err != nil {
    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
        "error": err.Error(),
        "code":  "INVALID_PROVIDER",
    })
}
```

## Acceptance

- [ ] Single source of truth for provider validation
- [ ] All 3 locations use shared function
- [ ] Tests cover invalid provider input
- [ ] Documentation updated

## Related

- #013 (Token Generation Duplication)
