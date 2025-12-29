---
status: resolved
priority: p1
issue_id: "001"
tags:
  - security
  - jwt
  - authentication
  - code-review
dependencies: []
---

# P1: Default JWT Secret Allows Authentication Bypass

## Problem Statement

The JWT middleware uses a hardcoded default secret when `JWT_SECRET` environment variable is not set. This allows attackers to forge valid JWT tokens and authenticate as any user.

**File**: `backend/internal/middleware/jwt.go:51-53`

```go
if config.Secret == "" {
    config.Secret = "change-this-secret-in-production"
}
```

## Findings

### Security Impact
- **OWASP**: A07:2021 - Identification and Authentication Failures
- **CWE**: CWE-322 (Key Exchange without Entity Authentication)
- **Severity**: CRITICAL - Allows complete authentication bypass

### Exploit Scenario
1. Attacker discovers environment lacks `JWT_SECRET`
2. Attacker forges JWT using known default secret `"change-this-secret-in-production"`
3. Attacker sets arbitrary `sub` claim to authenticate as any user
4. Attacker gains full access to victim's account

### Evidence
- Location: `internal/middleware/jwt.go:50-53`
- Silent failure mode - app starts with weak crypto
- No minimum secret strength validation
- Production deployments may accidentally use this

## Proposed Solutions

### Option 1: Fail Fast on Missing Secret (Recommended)
```go
func NewJWTConfig(options ...func(*JWTConfig)) (*JWTConfig, error) {
    config := &JWTConfig{
        Secret: os.Getenv("JWT_SECRET"),
        // ...
    }

    if config.Secret == "" {
        return nil, errors.New("JWT_SECRET must be set in production")
    }

    // Verify minimum secret strength
    if len(config.Secret) < 32 {
        return nil, errors.New("JWT_SECRET must be at least 32 characters")
    }

    return config, nil
}
```

**Pros**:
- Prevents accidental production deployments with weak secrets
- Clear error message on startup
- Fails fast rather than running with vulnerability

**Cons**:
- Requires explicit configuration in all environments
- Adds complexity to config initialization

**Effort**: Small
**Risk**: Low

### Option 2: Environment-Specific Defaults
```go
func NewJWTConfig() *JWTConfig {
    secret := os.Getenv("JWT_SECRET")

    if secret == "" {
        env := os.Getenv("ENV")
        if env == "production" {
            log.Fatal("JWT_SECRET must be set in production")
        }
        secret = "dev-only-secret-not-for-production-" + os.Getenv("HOSTNAME")
        log.Printf("WARNING: Using dev-only JWT secret")
    }

    return &JWTConfig{Secret: secret}
}
```

**Pros**:
- Allows development without explicit config
- Still protects production
- Hostname makes dev secret unique

**Cons**:
- Still allows weak crypto in development
- Relies on `ENV` variable being set correctly

**Effort**: Small
**Risk**: Low

### Option 3: Generate Secret on Startup
```go
if secret == "" {
    if env == "production" {
        log.Fatal("JWT_SECRET must be set in production")
    }
    secret = generateRandomSecret(32)
    log.Printf("Generated temporary JWT secret: %s", secret)
}
```

**Pros**:
- Never uses hardcoded secret
- Unique secret per server restart

**Cons**:
- Invalidates all tokens on restart (bad for UX)
- Still not production-ready

**Effort**: Small
**Risk**: Medium (token invalidation)

## Recommended Action

**Implement Option 1** - Fail fast on missing JWT secret with minimum length validation.

## Technical Details

### Affected Files
- `backend/internal/middleware/jwt.go:50-53`
- `backend/internal/middleware/jwt.go:20-56` (config initialization)
- `backend/cmd/server/main.go:119` (JWT middleware setup)

### Components
- JWT Middleware
- Authentication flow
- Token validation

### Database Changes
None required

## Acceptance Criteria

- [ ] `JWT_SECRET` is required in production environment
- [ ] Minimum secret length of 32 characters enforced
- [ ] Clear error message when secret is missing
- [ ] Application fails to start when requirements not met
- [ ] Development environment can still use a default (with warning)
- [ ] All tests pass with new validation
- [ ] Documentation updated with environment variable requirements

## Work Log

### 2025-12-29
- **Finding**: Security audit identified hardcoded JWT secret fallback
- **Analysis**: Confirmed critical vulnerability allows authentication bypass
- **Decision**: Selected Option 1 (fail fast approach)
- **Status**: Pending implementation

## Resources

- [OWASP A07:2021](https://owasp.org/Top10/A07_2021-Identification_and_Authentication_Failures/)
- [CWE-322](https://cwe.mitre.org/data/definitions/322.html)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)
