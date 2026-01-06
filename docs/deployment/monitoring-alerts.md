# Monitoring & Alerts in Production

Proactive monitoring is essential to ensure the reliability of job notifications. If GengoWatcher is down, our users miss income opportunities.

## 1. Monitoring Stack

We recommend a standard observability stack:
- **Prometheus**: For metrics collection (CPU, Memory, Request Rates).
- **Grafana**: For visualization dashboards.
- **Loki / ELK**: For centralized log aggregation.
- **UptimeRobot / BetterStack**: For external uptime monitoring.

---

## 2. Critical Metrics to Watch

| Metric | Threshold for Alert | Impact |
|--------|---------------------|--------|
| **HTTP 5xx Error Rate** | > 1% over 5 mins | Users cannot log in or manage watchers. |
| **Watcher Error Rate** | > 5% over 10 mins | Jobs are not being discovered correctly. |
| **DB Connection Usage** | > 80% of capacity | API requests will start failing. |
| **Redis Memory** | > 85% of capacity | Rate limiting and caching will degrade. |
| **API Latency (p99)** | > 500ms | Degraded user experience. |

---

## 3. Alerting Channels

Configure alerts to reach the engineering team via:
- **Slack/Discord**: For "Warning" level alerts.
- **PagerDuty/Opsgenie**: For "Critical" level alerts (e.g., system down).
- **Email**: For daily summaries and non-urgent metrics.

---

## 4. Health Check Strategy

### External Uptime Check
Ping the `/health` endpoint every 60 seconds from multiple global regions.

### Deep Health Check
The `/ready` endpoint should check:
- Is PostgreSQL responsive?
- Is Redis reachable?
- Can we talk to the Resend API?

---

## 5. Log-Based Alerting

Set up alerts for specific error patterns in your logs:
- `ERR_OAUTH_CONFIG`: OAuth is misconfigured.
- `ERR_STRIPE_WEBHOOK`: Failed to process a billing event.
- `PANIC`: The Go backend has encountered a fatal error.

## Next Steps
- [Administration: Monitoring](../administration/monitoring.md)
- [Troubleshooting Guide](../administration/troubleshooting.md)
