# Authentication Documentation

## Overview

GengoWatcher supports multiple authentication methods:
1. **Email/Password** - Traditional username and password
2. **Magic Link** - Passwordless email-based authentication
3. **OAuth 2.0** - Google and GitHub social login

## Token System

### Access Token (JWT)

Short-lived token for API authentication.

| Property | Value |
|----------|-------|
| Type | JWT (RS256) |
| Expiry | 15 minutes |
| Payload | `user_id`, `email`, `exp`, `iat` |
| Storage | Memory only (not persisted) |

### Refresh Token

Long-lived token for session management.

| Property | Value |
|----------|-------|
| Type | Random string (32 bytes) |
| Expiry | 7 days |
| Storage | httpOnly cookie + database |
| Rotation | New token on each refresh |

### Token Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Token Lifecycle                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. User logs in                                                            │
│     ┌─────────┐      POST /auth/login       ┌──────────┐                   │
│     │ Browser │ ──────────────────────────▶ │ Backend  │                   │
│     └─────────┘                            └────┬─────┘                   │
│                                                  │                         │
│                                           Generate JWT + Refresh Token      │
│                                                  │                         │
│                                    ┌──────────────┴──────────────┐          │
│                                    │                               │          │
│                            Store refresh token              Set httpOnly    │
│                            in database                       cookie          │
│                                    │                               │          │
│                                    │               ┌──────────────▼───────┐  │
│                                    │               │ Browser stores cookie │  │
│                                    │               └──────────────────────┘  │
│                                    │                                         │
│  2. Access token expires                                                          │
│     ┌─────────┐      Request (no valid token)   ┌──────────┐                 │
│     │ Browser │ ───────────────────────────────▶ │ Backend  │                 │
│     └─────────┘                                 └────┬─────┘                 │
│                                                      │                       │
│                                               Token expired                   │
│                                                      │                       │
│                                               Return 401 Unauthorized         │
│                                                      │                       │
│                                    ┌─────────────────▼─────────────────┐     │
│                                    │ Browser automatically sends      │     │
│                                    │ refresh token cookie            │     │
│                                    └─────────────────┬───────────────┘     │
│                                                          │                 │
│                                    ┌────────────────────▼─────────────────┐ │
│                                    │ Validate refresh token against DB  │ │
│                                    │ Generate new access token         │ │
│                                    └────────────────────┬────────────────┘ │
│                                                          │                 │
│                                    ┌────────────────────▼────────────────┐  │
│                                    │ Return new access token          │  │
│                                    └───────────────────────────────────┘  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Authentication Methods

### 1. Email/Password Authentication

#### Registration

```
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securePassword123!"
}
```

**Password Requirements:**
- Minimum 8 characters
- No maximum length
- No specific complexity requirements (but recommended)

**Password Hashing:**
- Algorithm: bcrypt
- Cost: 12 (OWASP recommended minimum for 2024+)
- Example:
  ```go
  hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
  ```

#### Login

```
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securePassword123!"
}
```

**Response (200 OK):**

```json
{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "email_verified": false,
    "is_active": true
  }
}
```

**Error Responses:**

| Code | HTTP Status | Message |
|------|-------------|---------|
| INVALID_CREDENTIALS | 401 | Invalid email or password |

---

### 2. Magic Link Authentication

#### Request Magic Link

```
POST /api/v1/auth/magic-link
Content-Type: application/json

{
  "email": "user@example.com"
}
```

**Security Note:** Always returns 200 OK even if email doesn't exist (prevents user enumeration).

**Response (200 OK):**

```json
{
  "message": "If an account exists, a magic link has been sent"
}
```

**Token Details:**
- Length: 32 bytes (base64 encoded = 43 characters)
- Expiry: 15 minutes
- Storage: Database with automatic cleanup

#### Verify Magic Link

```
GET /api/v1/auth/verify?token=<token>
```

**On Success:** Redirects to dashboard with session cookie.

**On Failure:** Returns error page with message.

---

### 3. OAuth 2.0 Authentication

#### Supported Providers

| Provider | Client ID Env Var | Client Secret Env Var |
|----------|-------------------|----------------------|
| Google | GOOGLE_OAUTH_CLIENT_ID | GOOGLE_OAUTH_CLIENT_SECRET |
| GitHub | GITHUB_OAUTH_CLIENT_ID | GITHUB_OAUTH_CLIENT_SECRET |

#### OAuth Flow

**Step 1: Get Authorization URL**

```
GET /api/v1/oauth/authorize?provider=google
```

**Response (200 OK):**

```json
{
  "auth_url": "https://accounts.google.com/o/oauth2/v2/auth?..."
}
```

**Redirect the user to this URL.**

**Step 2: Handle Callback**

The OAuth provider redirects to:

```
GET /api/v1/oauth/google/callback?code=AUTHORIZATION_CODE&state=CSRF_STATE
```

**Security Validation:**
1. Verify state format (32-64 alphanumeric characters)
2. Verify state matches session (CSRF protection)
3. Verify state hasn't expired (10 minutes)
4. Delete used state

**Step 3: Token Exchange**

Backend exchanges authorization code for access token with OAuth provider.

**Step 4: User Provisioning**

```
If user exists with same email:
  → Link OAuth account to existing user

If OAuth account exists:
  → Fetch associated user

If neither exists:
  → Create new user with:
    - Random password (32 chars)
    - Email verified = true (from OAuth provider)
  → Create OAuth account link
```

**Step 5: Session Creation**

Sets `refresh_token` httpOnly cookie and returns user info.

---

## Cookie Security

### Cookie Configuration

| Attribute | Value | Purpose |
|-----------|-------|---------|
| Name | `refresh_token` | Consistent naming |
| HTTPOnly | `true` | Prevents JavaScript access |
| Secure | `true` (prod) | HTTPS only |
| SameSite | `Strict` | CSRF protection |
| MaxAge | 7 days | Session duration |
| Domain | Empty | Host-only |

### Security Headers

All responses include:

```
X-Frame-Options: SAMEORIGIN
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Strict-Transport-Security: max-age=63072000 (production)
```

---

## Password Security

### Hashing Configuration

| Setting | Value |
|---------|-------|
| Algorithm | bcrypt |
| Cost Factor | 12 |
| Salt Length | 128 bits (built-in) |

### Password Verification

```go
import "golang.org/x/crypto/bcrypt"

func VerifyPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

### Password Generation (for OAuth users)

```go
import "github.com/tdawe1/translation-app/internal/password"

func GenerateSecurePassword() (string, error) {
    return password.GenerateRandomPassword(32)
}
```

---

## Session Management

### Session Storage

| Token Type | Storage Location | TTL |
|------------|-----------------|-----|
| Access Token | Memory (client) | 15 minutes |
| Refresh Token | Database + httpOnly cookie | 7 days |

### Session Invalidation

Sessions are invalidated when:
1. User logs out
2. Password is changed
3. User account is deactivated
4. Refresh token expires
5. Admin revokes session

### Concurrent Sessions

Users can have multiple active sessions (different devices/browsers). Each gets a unique refresh token.

---

## Security Measures

### CSRF Protection

OAuth state parameter includes session binding:

```
state = sessionID:randomState
```

On callback, verifies `sessionID` matches cookie.

### Rate Limiting

| Endpoint | Limit | Window |
|----------|-------|--------|
| POST /auth/register | 3 | per minute |
| POST /auth/login | 10 | per minute |
| POST /auth/magic-link/send | 3 | per hour |
| POST /auth/password-reset/send | 3 | per hour |

### Timing Attack Prevention

Token comparison uses constant-time algorithm:

```go
import "crypto/subtle"

func SecureCompare(a, b string) bool {
    return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
```

### User Enumeration Prevention

All authentication errors return generic messages:

| Attempted Action | Returned Message |
|-----------------|------------------|
| Login with invalid email | "Invalid credentials" |
| Login with wrong password | "Invalid credentials" |
| Request magic link (any email) | "If an account exists, a magic link has been sent" |
| Register with existing email | "Invalid credentials" (to avoid revealing existence) |

---

## API Key Authentication

For programmatic access (future feature).

### Creating API Keys

```
POST /api/v1/me/api-keys
Content-Type: application/json

{
  "name": "My Integration",
  "expires_at": "2026-12-31T23:59:59Z"  // Optional
}
```

**Response:**

```json
{
  "id": "uuid",
  "name": "My Integration",
  "key_prefix": "gk_live_abc123...",  // First 12 chars visible once
  "expires_at": "2026-12-31T23:59:59Z"
}
```

### Using API Keys

Include in Authorization header:

```
Authorization: Bearer gk_live_abc123...
```

---

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| "Invalid credentials" | Check email/password; reset password if needed |
| Magic link expired | Request new link (15 min expiry) |
| OAuth callback error | Check OAuth app configuration |
| Session expired | Refresh token or re-login |
| Too many requests | Wait for rate limit reset |

### Debug Steps

1. Check browser cookies for `refresh_token`
2. Verify JWT isn't expired (check jwt.io)
3. Confirm user account is active
4. Check rate limit headers:
   ```
   X-RateLimit-Limit: 10
   X-RateLimit-Remaining: 0
   X-RateLimit-Reset: 1234567890
   ```

---

**Last Updated**: January 2026
**Version**: 1.0.0
