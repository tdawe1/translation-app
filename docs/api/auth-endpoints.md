# Authentication Endpoints

Endpoints for managing user accounts and sessions.

## Register User
Create a new user account with email and password.

- **URL**: `/auth/register`
- **Method**: `POST`
- **Auth Required**: No

### Request Body
```json
{
  "email": "user@example.com",
  "password": "securePassword123!"
}
```

### Success Response
- **Code**: `201 Created`
- **Body**:
```json
{
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "email_verified": false
  },
  "access_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

---

## Login
Authenticate with email and password to start a session.

- **URL**: `/auth/login`
- **Method**: `POST`
- **Auth Required**: No

### Request Body
```json
{
  "email": "user@example.com",
  "password": "securePassword123!"
}
```

### Success Response
- **Code**: `200 OK`
- **Body**:
```json
{
  "data": {
    "id": "uuid",
    "email": "user@example.com"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIs..."
}
```
**Note**: Sets an `httpOnly` cookie named `refresh_token`.

---

## Logout
Terminate the current session and invalidate tokens.

- **URL**: `/auth/logout`
- **Method**: `POST`
- **Auth Required**: Yes

### Success Response
- **Code**: `204 No Content`

---

## Request Magic Link
Send a passwordless login link to an email address.

- **URL**: `/auth/magic-link`
- **Method**: `POST`
- **Auth Required**: No

### Request Body
```json
{
  "email": "user@example.com"
}
```

### Success Response
- **Code**: `200 OK`
- **Body**:
```json
{
  "message": "If an account exists, a magic link has been sent"
}
```
**Security Note**: Returns the same response regardless of whether the email exists to prevent account enumeration.

---

## Get Current User
Retrieve the profile of the currently authenticated user.

- **URL**: `/auth/me`
- **Method**: `GET`
- **Auth Required**: Yes

### Success Response
- **Code**: `200 OK`
- **Body**:
```json
{
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "email_verified": true,
    "tier": "pro",
    "created_at": "2026-01-01T12:00:00Z"
  }
}
```

---

## Update User Password
Change the password for the current user.

- **URL**: `/auth/me/password`
- **Method**: `PUT`
- **Auth Required**: Yes

### Request Body
```json
{
  "current_password": "oldPassword123!",
  "new_password": "newSecurePassword456!"
}
```

### Success Response
- **Code**: `204 No Content`

## Next Steps
- [OAuth Endpoints](../api/oauth-endpoints.md)
- [Watcher Endpoints](../api/watcher-endpoints.md)
