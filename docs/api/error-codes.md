# Error Code Reference

GengoWatcher SaaS uses machine-readable error codes to help you handle exceptions gracefully in your integrations.

## Error Response Structure

```json
{
  "error": "Human readable message",
  "code": "SPECIFIC_ERROR_CODE",
  "details": {
    "field": "Optional field name",
    "reason": "Specific validation failure"
  }
}
```

---

## Authentication Errors

| Code | Status | Description |
|------|--------|-------------|
| `NOT_AUTHENTICATED` | 401 | Missing or invalid session/JWT. |
| `INVALID_CREDENTIALS` | 401 | Incorrect email or password. |
| `TOKEN_EXPIRED` | 401 | The provided token has expired. |
| `INSUFFICIENT_PERMISSIONS` | 403 | You don't have access to this resource. |
| `TIER_LIMIT_REACHED` | 403 | Action blocked by subscription limits. |

---

## Validation Errors

| Code | Status | Description |
|------|--------|-------------|
| `BAD_REQUEST` | 400 | Malformed JSON or invalid parameters. |
| `VALIDATION_FAILED` | 400 | Specific fields failed validation. See `details`. |
| `CONFLICT` | 409 | Resource already exists (e.g., email already taken). |

---

## Watcher Errors

| Code | Status | Description |
|------|--------|-------------|
| `WATCHER_ALREADY_RUNNING` | 409 | Attempted to start a watcher that is already active. |
| `INVALID_RSS_URL` | 400 | The provided RSS feed URL is malformed or unreachable. |
| `WATCHER_NOT_FOUND` | 404 | No watcher exists for the current user. |

---

## System Errors

| Code | Status | Description |
|------|--------|-------------|
| `INTERNAL_SERVER_ERROR` | 500 | An unexpected error occurred on our side. |
| `SERVICE_UNAVAILABLE` | 503 | Database or Redis is temporarily down. |
| `RATE_LIMITED` | 429 | Too many requests. Please slow down. |

---

## Handling Errors (Example)

```javascript
try {
  const response = await api.post('/watcher/start');
} catch (error) {
  if (error.code === 'TIER_LIMIT_REACHED') {
    showUpgradeModal();
  } else if (error.code === 'INVALID_RSS_URL') {
    highlightInput('rss_url');
  }
}
```

## Next Steps
- [API Overview](../api/overview.md)
- [Rate Limiting](../api/rate-limiting.md)
