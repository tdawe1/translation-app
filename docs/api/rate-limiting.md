# Rate Limiting

GengoWatcher SaaS employs rate limiting to protect our API from abuse, ensure system stability, and guarantee fair resource allocation for all users.

## How it Works

We use the **Leaky Bucket** algorithm for rate limiting. Limits are applied based on your **IP Address** for unauthenticated requests and your **User ID** for authenticated requests.

---

## Standard Limits

| Category | Limit | Window | Applies To |
|----------|-------|--------|------------|
| **Public API** | 60 | Minute | All non-auth endpoints |
| **Authentication** | 5 | Minute | Login, Register |
| **User API (Free)** | 100 | Minute | All `/api/v1` endpoints |
| **User API (Pro)** | 500 | Minute | All `/api/v1` endpoints |
| **Webhooks** | 1000 | Minute | Incoming billing events |

---

## Rate Limit Headers

Every API response includes headers indicating your current rate limit status:

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | The maximum number of requests allowed in the window. |
| `X-RateLimit-Remaining` | The number of requests remaining in the current window. |
| `X-RateLimit-Reset` | The Unix timestamp when the limit will reset. |

---

## Handling Rate Limits

When a limit is exceeded, the API returns a `429 Too Many Requests` status code.

**Error Response:**
```json
{
  "error": "Rate limit exceeded. Please try again later.",
  "code": "RATE_LIMITED",
  "details": {
    "retry_after_seconds": 45
  }
}
```

### Best Practices
- **Exponential Backoff**: If you receive a 429, wait before retrying. Increase the wait time if you continue to hit the limit.
- **Check Headers**: Monitor the `X-RateLimit-Remaining` header to proactively slow down requests before hitting the limit.
- **Cache Results**: Cache frequently accessed data (like your own profile or configuration) to reduce unnecessary API calls.

## Next Steps
- [Error Codes](../api/error-codes.md)
- [API Overview](../api/overview.md)
