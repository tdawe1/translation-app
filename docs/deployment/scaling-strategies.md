# Scaling Strategies

As GengoWatcher SaaS grows, the system must scale to handle more users, more watchers, and higher volumes of job discoveries.

## 1. Horizontal Scaling (Compute)

### Stateless Backend
The Go API is stateless. All session data is stored in Redis or PostgreSQL. This allows us to run multiple instances behind a load balancer.

- **Scaling Metric**: CPU usage or Request Count.
- **Tools**: Kubernetes HPA or AWS Auto Scaling Groups.

### Frontend
The Next.js frontend is also stateless and can be scaled horizontally. For static assets, we use a **CDN** (e.g., CloudFront, Cloudflare) to offload traffic from the application servers.

---

## 2. Scaling the Watcher Engine

The Watcher Engine is the most resource-intensive part of the system.

- **Concurrency**: Each user watcher runs in its own Go routine. Go can handle thousands of concurrent routines efficiently.
- **Resource Limits**: In production, we limit the number of watchers per instance. If the limit is reached, new watchers are started on a different instance (coordinated via Redis).
- **Polling Optimization**: As the number of users grows, we use a **Worker Pool** pattern to poll RSS feeds, preventing us from opening too many concurrent outbound connections.

---

## 3. Database Scaling (PostgreSQL)

The database is the most common bottleneck in SaaS applications.

- **Read Replicas**: Direct all `GET` requests to one or more read-only replicas.
- **Connection Pooling**: Use **PgBouncer** to manage thousands of concurrent database connections from multiple API instances.
- **Sharding (Future)**: If the `users` or `jobs` table grows into the billions, we will implement horizontal sharding based on `user_id`.

---

## 4. Cache Scaling (Redis)

Redis handles rate limiting and pub/sub.

- **Cluster Mode**: Distribute keys across multiple Redis nodes.
- **Replication**: Use a primary-replica setup for high availability.

---

## 5. Network Scaling

- **CDN**: Cache static assets (JS, CSS, Images) at the edge.
- **Anycast IP**: Use services like Cloudflare to route users to the nearest data center.
- **WebSocket Load Balancing**: Ensure your load balancer supports "Sticky Sessions" (if necessary) or a robust Redis Pub/Sub backend to handle WebSocket broadcasts across multiple server nodes.

## Next Steps
- [Kubernetes Deployment Guide](../deployment/kubernetes-deployment.md)
- [Architecture Overview](../core-concepts/architecture-overview.md)
