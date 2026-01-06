# Watcher Endpoints

Endpoints for managing your job monitoring instances and their configurations.

## Get Watcher Config
Retrieve the current monitoring settings for the user.

- **URL**: `/watcher/config`
- **Method**: `GET`
- **Auth Required**: Yes

### Success Response
- **Code**: `200 OK`
- **Body**:
```json
{
  "data": {
    "rss_feed_url": "https://gengo.com/jobs/rss",
    "min_reward": 5.0,
    "max_reward": 500.0,
    "included_language_pairs": ["en-ja", "en-es"],
    "enable_email_notifs": true,
    "auto_accept_enabled": false
  }
}
```

---

## Update Watcher Config
Update monitoring settings and filters.

- **URL**: `/watcher/config`
- **Method**: `PUT`
- **Auth Required**: Yes

### Request Body
```json
{
  "min_reward": 10.0,
  "included_language_pairs": ["en-ja", "fr-en"],
  "enable_email_notifs": true
}
```

### Success Response
- **Code**: `200 OK`
- **Body**:
```json
{
  "data": {
    "id": "uuid",
    "updated_at": "2026-01-05T23:00:00Z"
  }
}
```

---

## Get Watcher State
Retrieve the current operational status of the watcher.

- **URL**: `/watcher/state`
- **Method**: `GET`
- **Auth Required**: Yes

### Success Response
- **Code**: `200 OK`
- **Body**:
```json
{
  "data": {
    "status": "running",
    "last_poll_at": "2026-01-05T23:05:00Z",
    "jobs_found_today": 12,
    "active_errors": []
  }
}
```

---

## Start Watcher
Activate the background monitoring process.

- **URL**: `/watcher/start`
- **Method**: `POST`
- **Auth Required**: Yes

### Success Response
- **Code**: `200 OK`
- **Body**:
```json
{
  "message": "Watcher started successfully",
  "status": "running"
}
```

---

## Stop Watcher
Pause the background monitoring process.

- **URL**: `/watcher/stop`
- **Method**: `POST`
- **Auth Required**: Yes

### Success Response
- **Code**: `200 OK`
- **Body**:
```json
{
  "message": "Watcher stopped successfully",
  "status": "stopped"
}
```

---

## Get Found Jobs
Retrieve a paginated list of jobs discovered by the watcher.

- **URL**: `/watcher/jobs`
- **Method**: `GET`
- **Auth Required**: Yes

### Query Parameters
- `page`: Page number (default: 1)
- `per_page`: Jobs per page (default: 20)
- `status`: Filter by `new`, `accepted`, `ignored`

### Success Response
- **Code**: `200 OK`
- **Body**:
```json
{
  "data": [
    {
      "id": "job_123",
      "title": "Medical Translation",
      "reward": 45.0,
      "found_at": "2026-01-05T23:00:00Z"
    }
  ],
  "pagination": {
    "total": 1,
    "page": 1,
    "per_page": 20
  }
}
```

## Next Steps
- [Core Concepts: Watcher System](../core-concepts/watcher-system.md)
- [WebSocket API](../api/websocket-api.md)
