# HTTP Status Code Reference

GengoWatcher SaaS uses standard HTTP status codes to communicate the result of API requests.

## Success (2xx)

| Code | Status | Description |
|------|--------|-------------|
| **200** | `OK` | The request was successful. |
| **201** | `Created` | A new resource (e.g., User, API Key) was created. |
| **204** | `No Content` | Action succeeded, but there is no data to return. |

---

## Client Errors (4xx)

| Code | Status | Description |
|------|--------|-------------|
| **400** | `Bad Request` | Invalid input, missing required fields, or malformed JSON. |
| **401** | `Unauthorized` | Authentication is required or your token is invalid. |
| **403** | `Forbidden` | You are authenticated but do not have permission for this action (e.g., tier limits). |
| **404** | `Not Found` | The requested resource does not exist. |
| **409** | `Conflict` | The request conflicts with current state (e.g., email already registered). |
| **422** | `Unprocessable Entity` | Valid JSON but fails business logic validation. |
| **429** | `Too Many Requests` | You have exceeded your rate limit. |

---

## Server Errors (5xx)

| Code | Status | Description |
|------|--------|-------------|
| **500** | `Internal Server Error` | An unexpected error occurred on our end. |
| **503** | `Service Unavailable` | The database, cache, or a required third-party service is down. |

---

## Error Response Body
When a `4xx` or `5xx` error occurs, the API returns a JSON object with more details:

```json
{
  "error": "Descriptive message",
  "code": "MACHINE_READABLE_CODE",
  "details": {
    "field": "reason"
  }
}
```

## Next Steps
- [Error Code Reference](../api/error-codes.md)
- [API Overview](../api/overview.md)
