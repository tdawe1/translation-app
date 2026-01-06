# Multi-Tenancy & User Isolation

GengoWatcher SaaS is a multi-tenant application where multiple users share the same infrastructure while their data remains strictly isolated.

## The Isolation Model

We use a **Logical Isolation** model at the database level and **Process Isolation** at the application level.

### 1. Database Level Isolation
Every table containing user-specific data includes a `user_id` column.

**Example Schema:**
```sql
CREATE TABLE watcher_configs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) NOT NULL,
    rss_feed_url TEXT,
    min_reward DECIMAL,
    -- ...
);
```

**Query Enforcement:**
Our backend application enforces isolation by always including the `user_id` in the `WHERE` clause of every query. This is handled by a middleware that extracts the `user_id` from the authenticated JWT.

```go
// Enforced isolation in Go
func GetConfig(c *fiber.Ctx) error {
    userID := c.Locals("userID").(uuid.UUID)
    var config models.WatcherConfig
    db.Where("user_id = ?", userID).First(&config)
    return c.JSON(config)
}
```

---

### 2. Real-Time Isolation (Redis & WebSockets)
WebSocket notifications must only reach the intended user. We achieve this using scoped Redis channels.

- **Channel Pattern**: `user:{user_id}:notifications`
- **Subscriber**: When a user connects via WebSocket, the server subscribes to only their specific channel.
- **Publisher**: When a background watcher finds a job for a user, it publishes only to that user's channel.

---

### 3. Background Process Isolation
Each user can have one or more "Watcher" instances running. These are managed by the `WatcherManager`.

- **Manager**: Tracks active routines in a map: `map[userID]Watcher`.
- **Routines**: Each watcher runs in its own lightweight Go routine.
- **Resources**: CPU and memory usage are monitored to ensure no single user's watcher can starve the system (preventing "Noisy Neighbor" issues).

---

## Security Implications

### Data Leakage Prevention
- **API**: Unauthorized access attempts (e.g., trying to access another user's UUID) result in a `404 Not Found` or `401 Unauthorized`.
- **Logs**: We automatically redact sensitive information and never log PII (Personally Identifiable Information) along with raw job data.

### Subscription Quotas
Isolation also allows us to enforce tier-based limits:
- **Free**: 1 concurrent watcher instance.
- **Pro**: Up to 5 concurrent watcher instances.
- **Enterprise**: Custom limits.

## Infrastructure Sharing
While users share the Backend API, PostgreSQL, and Redis clusters, the logical partitioning ensures that user A can never see or interact with the data or processes of user B.

## Next Steps
- [Architecture Overview](../core-concepts/architecture-overview.md)
- [Watcher System](../core-concepts/watcher-system.md)
