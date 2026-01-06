# Troubleshooting Guide

This guide covers common issues encountered by users and administrators and provides steps for resolution.

## User-Facing Issues

### 1. Not Receiving Notifications
- **Check Watcher Status**: Is the watcher currently "Running"?
- **Check Filters**: Are the filters too restrictive? (Try lowering the min reward).
- **Check WebSocket Connection**: Is the "Live" indicator green in the dashboard?
- **Email Issues**: Verify the email isn't in Spam and the address is correct in settings.

### 2. Login Fails
- **OAuth Error**: Ensure you're using the same provider (Google vs. GitHub) you used during signup.
- **Verification Missing**: Have you clicked the link in your verification email?
- **Cookie Blocked**: Ensure your browser allows cookies from `gengowatcher.com`.

---

## Technical/System Issues

### 1. Database Connection Refused
- **Symptom**: `503 Service Unavailable` or backend logs show `dial tcp: connection refused`.
- **Solution**: 
  - Check if PostgreSQL is running: `docker ps`.
  - Verify `DATABASE_URL` environment variable.
  - Check if the database host is reachable from the application container.

### 2. Redis Pub/Sub Failures
- **Symptom**: Watchers find jobs, but dashboard doesn't update in real-time.
- **Solution**:
  - Verify Redis is running.
  - Check backend logs for `redis: connection lost`.
  - Ensure the WebSocket handler is correctly subscribed to the user's channel.

### 3. High CPU Usage
- **Symptom**: Server responsiveness degrades.
- **Solution**:
  - Identify if a specific user's watcher is causing issues (e.g., polling too fast).
  - Check for Go routine leaks using `pprof`.
  - Increase horizontal scaling (add more instances).

---

## Recovery Procedures

### Restarting the System
1. **Stop Traffic**: Point the load balancer to a maintenance page.
2. **Graceful Shutdown**: Send `SIGTERM` to backend processes.
3. **Infrastructure**: Restart Redis and PostgreSQL if necessary.
4. **Start Backend**: Launch backend instances and wait for health checks.
5. **Resume Traffic**: Point the load balancer back to the production instances.

### Database Backups
In case of data corruption:
1. Locate the latest daily snapshot.
2. Spin up a temporary database instance from the snapshot.
3. Verify data integrity.
4. Point the application to the new database instance.

## Next Steps
- [Monitoring & Metrics](../administration/monitoring.md)
- [Security Best Practices](../administration/security-best-practices.md)
