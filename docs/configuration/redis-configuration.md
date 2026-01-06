# Redis Configuration

Redis is a vital component of GengoWatcher SaaS, serving as our primary cache, rate limiter, and real-time message broker.

## 1. Usage Scenarios

GengoWatcher uses Redis for three distinct purposes:

### A. Rate Limiting
Stores IP-based and User-based request counts to prevent API abuse.
- **Key Pattern**: `ratelimit:{id}`
- **Type**: Counter with TTL.

### B. Real-Time Pub/Sub
Broadcasts job discoveries from background watchers to WebSocket handlers.
- **Channel Pattern**: `user:{id}:jobs`
- **Type**: Pub/Sub Message.

### C. Caching & Sessions
Stores temporary data like WebSocket tickets and OAuth states.
- **Key Pattern**: `ws_ticket:{token}`
- **Type**: String with short TTL (e.g., 60 seconds).

---

## 2. Configuration Parameters

Configure Redis via the `REDIS_URL` environment variable.

| Component | Example Value |
|-----------|---------------|
| **Protocol** | `redis://` or `rediss://` (for TLS) |
| **Host/Port** | `localhost:6379` |
| **Database** | `/0` (Default) |

**Production Example:**
`REDIS_URL=rediss://:yourpassword@redis-prod.example.com:6379/0`

---

## 3. Performance Tuning

### Memory Policy
Since Redis is used for both volatile (cache) and non-volatile (pub/sub) data, we recommend the `allkeys-lru` eviction policy.
- **Config**: `maxmemory-policy allkeys-lru`

### Persistence
For maximum performance, you may choose to disable RDB/AOF persistence if your deployment treats Redis as a transient cache. However, if using Redis for session persistence, enable RDB snapshots every 15 minutes.

---

## 4. Security

- **Password Protection**: Always use a strong password in production.
- **TLS**: Use `rediss://` to encrypt data in transit between your API and Redis.
- **Network**: Ensure Redis is only accessible from your Backend API network (Private VPC).

---

## 5. Monitoring

Monitor these Redis metrics:
- **`used_memory`**: Ensure it stays within your allocated limits.
- **`connected_clients`**: Number of active connections from your API instances.
- **`pubsub_channels`**: Number of active users currently using WebSockets.

## Next Steps
- [Real-Time Notifications](../core-concepts/real-time-notifications.md)
- [Environment Variables Reference](../configuration/environment-variables.md)
