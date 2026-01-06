# Monitoring & Metrics

Ensuring the health and performance of GengoWatcher SaaS is critical for maintaining real-time job alerts. We use a combination of health checks, logging, and metrics.

## 1. Health Checks

### Basic Health
- **Endpoint**: `GET /health`
- **Purpose**: Liveness probe for load balancers and Kubernetes.
- **Success**: `200 OK`

### Readiness Probe
- **Endpoint**: `GET /ready`
- **Purpose**: Verifies that the app can handle traffic by checking DB and Redis connections.

---

## 2. Structured Logging

We use JSON-formatted logs for easy parsing by tools like ELK or Datadog.

**Example Log:**
```json
{
  "level": "info",
  "msg": "Job discovered for user",
  "user_id": "uuid",
  "job_id": "job_123",
  "latency_ms": 150
}
```

---

## 3. Key Performance Indicators (KPIs)

Monitor these metrics to ensure the system is operating within normal parameters.

| Metric | Type | Target | Description |
|--------|------|--------|-------------|
| **HTTP Latency** | Histogram | < 200ms | Response time for API endpoints. |
| **Watcher Success Rate**| Ratio | > 99% | Successful RSS/WS polls vs errors. |
| **Job Discovery Lag** | Gauge | < 5s | Time from job posting to user notification. |
| **Active Connections** | Gauge | N/A | Number of concurrent WebSocket sessions. |
| **Error Rate** | Counter | < 1% | Ratio of 5xx responses. |

---

## 4. Resource Monitoring

### PostgreSQL
- **Connection Pool**: Monitor `active_connections`.
- **Query Performance**: Identify slow queries using `pg_stat_statements`.

### Redis
- **Memory Usage**: Ensure we aren't hitting limits (used for caching and pub/sub).
- **Pub/Sub Throughput**: Monitor the volume of job alert messages.

---

## 5. Alerts

We recommend setting up alerts for the following conditions:
- **High Error Rate**: > 5% error rate for more than 2 minutes.
- **Latency Spike**: Average latency > 1s.
- **Watcher Downtime**: If the number of running watchers drops significantly without user action.
- **Disk Space**: If database or log volumes exceed 80% capacity.

## Next Steps
- [Troubleshooting Guide](../administration/troubleshooting.md)
- [Production Setup](../deployment/production-setup.md)
