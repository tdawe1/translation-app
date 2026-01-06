# API Reference

## Base URL

| Environment | URL |
|-------------|-----|
| Development | `http://localhost:8000` |
| Production | `https://api.yourdomain.com` |

## Authentication

All authenticated endpoints require a JWT access token in an httpOnly cookie. The cookie is set automatically upon login.

### Cookie Settings

| Attribute | Value |
|-----------|-------|
| Name | `refresh_token` |
| HTTPOnly | Yes |
| Secure | Yes (production) |
| SameSite | Strict |
| MaxAge | 7 days |

## Response Format

### Success Response

```json
{
  "data": { /* response payload */ }
}
```

### Error Response

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "email",
    "reason": "invalid format"
  }
}
```

### Pagination Response

```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

---

## Authentication Endpoints

### Register User

Create a new user account with email and password.

**Endpoint:** `POST /api/v1/auth/register`

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "securePassword123!"
}
```

**Responses:**

| Status | Description |
|--------|-------------|
| 201 Created | User created successfully |
| 400 Bad Request | Invalid input |
| 409 Conflict | Email already registered |

**Example Response (201):**

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "email_verified": false,
    "is_active": true,
    "created_at": "2026-01-05T22:00:00Z"
  }
}
```

---

### Login

Authenticate with email and password.

**Endpoint:** `POST /api/v1/auth/login`

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "securePassword123!"
}
```

**Responses:**

| Status | Description |
|--------|-------------|
| 200 OK | Login successful |
| 401 Unauthorized | Invalid credentials |

---

### Request Magic Link

Send a passwordless login link to email.

**Endpoint:** `POST /api/v1/auth/magic-link`

**Request Body:**

```json
{
  "email": "user@example.com"
}
```

**Responses:**

| Status | Description |
|--------|-------------|
| 200 OK | If account exists, magic link sent |
| 429 Too Many Requests | Rate limited |

**Note:** Returns same response regardless of whether email exists (prevents enumeration).

---

### Verify Magic Link

Complete magic link authentication.

**Endpoint:** `GET /api/v1/auth/verify?token=<token>`

**Responses:**

| Status | Description |
|--------|-------------|
| 302 Redirect | Redirects to dashboard on success |
| 400 Bad Request | Invalid or expired token |

---

### Get Current User

Get authenticated user's profile.

**Endpoint:** `GET /api/v1/me`

**Authentication Required:** Yes (JWT in cookie)

**Responses:**

| Status | Description |
|--------|-------------|
| 200 OK | Return user profile |
| 401 Unauthorized | Not authenticated |

**Example Response (200):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "email_verified": true,
  "is_active": true,
  "provider": "google",
  "oauth_accounts": [
    {
      "provider": "google",
      "created_at": "2026-01-01T00:00:00Z"
    }
  ],
  "created_at": "2026-01-01T00:00:00Z"
}
```

---

### Change Password

Change authenticated user's password.

**Endpoint:** `PUT /api/v1/me/password`

**Authentication Required:** Yes

**Request Body:**

```json
{
  "old_password": "oldPassword123!",
  "new_password": "newSecurePassword456!"
}
```

**Responses:**

| Status | Description |
|--------|-------------|
| 200 OK | Password changed |
| 400 Bad Request | Invalid input |
| 401 Unauthorized | Wrong old password |

---

### Logout

Invalidate current session.

**Endpoint:** `POST /api/v1/auth/logout`

**Authentication Required:** Yes

**Responses:**

| Status | Description |
|--------|-------------|
| 204 No Content | Logged out successfully |

---

## Email Verification Endpoints

### Send Verification Email

Resend email verification link.

**Endpoint:** `POST /api/v1/auth/verify-email/send`

**Authentication Required:** Yes

**Responses:**

| Status | Description |
|--------|-------------|
| 200 OK | Verification email sent |
| 429 Too Many Requests | Rate limited (3/hour) |

**Example Response (200):**

```json
{
  "message": "Verification email sent",
  "expires_at": "2026-01-05T22:15:00Z",
  "expires_in_minutes": 15
}
```

---

### Verify Email

Verify email address with token.

**Endpoint:** `POST /api/v1/auth/verify-email`

**Request Body:**

```json
{
  "token": "verification-token-here"
}
```

**Responses:**

| Status | Description |
|--------|-------------|
| 200 OK | Email verified |
| 400 Bad Request | Invalid/expired token |

---

## Password Reset Endpoints

### Send Password Reset Email

Request password reset link.

**Endpoint:** `POST /api/v1/auth/password-reset/send`

**Request Body:**

```json
{
  "email": "user@example.com"
}
```

**Responses:**

| Status | Description |
|--------|-------------|
| 200 OK | If account exists, reset link sent |
| 429 Too Many Requests | Rate limited |

---

### Reset Password

Reset password with token.

**Endpoint:** `POST /api/v1/auth/password-reset`

**Request Body:**

```json
{
  "token": "reset-token-here",
  "password": "newPassword123!"
}
```

**Responses:**

| Status | Description |
|--------|-------------|
| 200 OK | Password reset successful |
| 400 Bad Request | Invalid/expired token |

---

## OAuth Endpoints

### Authorize

Get OAuth authorization URL for a provider.

**Endpoint:** `GET /api/v1/oauth/authorize?provider=<google|github>`

**Responses:**

| Status | Description |
|--------|-------------|
| 200 OK | Returns authorization URL |

**Example Response (200):**

```json
{
  "auth_url": "https://accounts.google.com/o/oauth2/v2/auth?..."
}
```

**Redirect the user to `auth_url` to begin OAuth flow.**

---

### OAuth Callback

Handle OAuth provider callback. This endpoint is called by the OAuth provider after user authentication.

**Endpoints:**
- `GET /api/v1/oauth/google/callback`
- `GET /api/v1/oauth/github/callback`

**Query Parameters:**

| Parameter | Description |
|-----------|-------------|
| `code` | Authorization code from OAuth provider |
| `state` | State parameter for CSRF protection |

**Responses:**

| Status | Description |
|--------|-------------|
| 302 Redirect | Redirects to dashboard with session cookie |
| 400 Bad Request | Invalid state, code, or user |

**On success:** Sets `refresh_token` cookie and redirects to dashboard.

---

### Get Linked Accounts

Get user's connected OAuth accounts.

**Endpoint:** `GET /api/v1/oauth/accounts`

**Authentication Required:** Yes

**Responses:**

| Status | Description |
|--------|-------------|
| 200 OK | Returns linked accounts |

**Example Response (200):**

```json
{
  "linked_accounts": [
    {
      "provider": "google",
      "created_at": "2026-01-01T00:00:00Z"
    },
    {
      "provider": "github",
      "created_at": "2026-01-02T00:00:00Z"
    }
  ]
}
```

---

### Unlink Account

Disconnect an OAuth provider.

**Endpoint:** `DELETE /api/v1/oauth/<google|github>`

**Authentication Required:** Yes

**Responses:**

| Status | Description |
|--------|-------------|
| 204 No Content | Unlinked successfully |
| 400 Bad Request | Cannot unlink last provider |

---

## Watcher Endpoints

### Get Watcher Config

Get user's watcher configuration.

**Endpoint:** `GET /api/v1/watcher/config`

**Authentication Required:** Yes

**Example Response (200):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "rss_feed_url": "https://gengo.com/jobs/rss",
  "websocket_enabled": true,
  "gengo_user_id": "user123",
  "min_reward": 5.0,
  "max_reward": 1000.0,
  "included_language_pairs": ["en-ja", "en-es"],
  "enable_desktop_notifs": true,
  "enable_sound_notifs": true,
  "enable_email_notifs": true,
  "notification_email": "user@example.com",
  "auto_accept_enabled": false,
  "auto_accept_min_reward": 10.0,
  "auto_accept_max_reward": 100.0,
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-05T22:00:00Z"
}
```

---

### Update Watcher Config

Update user's watcher configuration.

**Endpoint:** `PUT /api/v1/watcher/config`

**Authentication Required:** Yes

**Request Body:**

```json
{
  "rss_feed_url": "https://gengo.com/jobs/rss",
  "min_reward": 10.0,
  "max_reward": 500.0,
  "enable_desktop_notifs": true,
  "notification_email": "user@example.com"
}
```

**All fields are optional.** Only provided fields will be updated.

---

### Get Watcher State

Get current watcher status.

**Endpoint:** `GET /api/v1/watcher/state`

**Authentication Required:** Yes

**Example Response (200):**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "watcher_status": "running",
  "last_poll": "2026-01-05T22:05:00Z",
  "jobs_found": 42,
  "jobs_accepted": 5,
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-05T22:00:00Z"
}
```

**Watcher Status Values:**
- `stopped` - Watcher is not running
- `running` - Actively monitoring
- `error` - Encountered an error

---

### Start Watcher

Start the job monitoring watcher.

**Endpoint:** `POST /api/v1/watcher/start`

**Authentication Required:** Yes

**Responses:**

| Status | Description |
|--------|-------------|
| 200 OK | Watcher started |

**Example Response (200):**

```json
{
  "status": "running",
  "message": "Watcher started successfully"
}
```

---

### Stop Watcher

Stop the job monitoring watcher.

**Endpoint:** `POST /api/v1/watcher/stop`

**Authentication Required:** Yes

**Responses:**

| Status | Description |
|--------|-------------|
| 200 OK | Watcher stopped |

**Example Response (200):**

```json
{
  "status": "stopped",
  "message": "Watcher stopped successfully"
}
```

---

## WebSocket Endpoints

### WebSocket Connection

Real-time job notifications via WebSocket.

**Endpoint:** `GET /ws`

**Authentication Required:** Yes (via WebSocket ticket)

### Connection Flow

1. Get WebSocket ticket:
   ```bash
   POST /api/v1/auth/ws-ticket
   # Returns: { "ticket": "xxx", "expires_at": 1234567890 }
   ```

2. Connect to WebSocket with ticket:
   ```javascript
   const ws = new WebSocket(
     'wss://api.yourdomain.com/ws?ticket=xxx'
   );
   ```

3. Handle messages:

```json
{
  "type": "job_found",
  "data": {
    "job_id": "job-123",
    "title": "English to Japanese Translation",
    "reward": 15.50,
    "language_pair": "en-ja",
    "url": "https://gengo.com/jobs/123"
  }
}
```

### WebSocket Message Types

| Type | Description |
|------|-------------|
| `job_found` | New job matching criteria |
| `job_accepted` | Auto-accepted job |
| `watcher_error` | Watcher encountered error |
| `heartbeat` | Keep-alive message |

---

## Rate Limiting

| Endpoint | Limit | Window |
|----------|-------|--------|
| POST /auth/register | 3 | per minute |
| POST /auth/login | 10 | per minute |
| POST /auth/verify-email/send | 3 | per hour |
| POST /auth/magic-link/send | 3 | per hour |
| POST /auth/password-reset/send | 3 | per hour |

**Rate Limited Response (429):**

```json
{
  "error": "Too many requests",
  "code": "RATE_LIMITED"
}
```

---

## Health Endpoints

### Basic Health

**Endpoint:** `GET /health`

**Response (200):**

```json
{
  "status": "healthy",
  "service": "gengowatcher-saas"
}
```

---

## HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 204 | No Content |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict |
| 429 | Too Many Requests |
| 500 | Internal Server Error |

---

**Last Updated**: January 2026
**Version**: 1.0.0
