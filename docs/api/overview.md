# API Overview

The GengoWatcher SaaS API is a RESTful interface that allows you to manage users, configure job watchers, and receive real-time notifications.

## Base URL

| Environment | URL |
|-------------|-----|
| **Development** | `http://localhost:8000/api/v1` |
| **Production** | `https://api.gengowatcher.com/api/v1` |

---

## Authentication

All authenticated endpoints require a valid session. We use a dual-token system:

1. **Access Token (JWT)**: Include this in the `Authorization` header as a Bearer token.
   ```http
   Authorization: Bearer <your_jwt_token>
   ```
2. **Refresh Token**: Stored in a secure, `httpOnly` cookie named `refresh_token`. This is automatically handled by the browser and used to rotate access tokens.

### WebSocket Authentication
WebSockets require a one-time "ticket" for connection. See the [WebSocket API](../api/websocket-api.md) for details.

---

## Content Type
All requests and responses use JSON. Ensure your requests include the following header:
```http
Content-Type: application/json
```

---

## Response Formats

### Standard Success Response
All data is wrapped in a `data` object.
```json
{
  "data": {
    "id": "uuid",
    "name": "Example"
  }
}
```

### Collection/List Response
Lists include a `pagination` object.
```json
{
  "data": [...],
  "pagination": {
    "total": 100,
    "page": 1,
    "per_page": 20,
    "total_pages": 5
  }
}
```

### Error Response
Errors follow a consistent structure.
```json
{
  "error": "Descriptive error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "email",
    "reason": "already exists"
  }
}
```

---

## Global Status Codes

| Code | Description |
|------|-------------|
| **200 OK** | Request succeeded. |
| **201 Created** | Resource created successfully. |
| **204 No Content** | Action succeeded, no content returned. |
| **400 Bad Request** | Invalid input or malformed JSON. |
| **401 Unauthorized** | Missing or invalid authentication. |
| **403 Forbidden** | Authenticated, but lacking permissions (e.g., tier limits). |
| **404 Not Found** | Resource does not exist. |
| **429 Too Many Requests** | Rate limit exceeded. |
| **500 Internal Error** | Something went wrong on our end. |

---

## Next Steps
- [Authentication Endpoints](../api/auth-endpoints.md)
- [Watcher Endpoints](../api/watcher-endpoints.md)
- [WebSocket API](../api/websocket-api.md)
- [Error Code Reference](../api/error-codes.md)
