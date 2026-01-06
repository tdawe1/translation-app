# The Watcher System

The Watcher System is the heart of GengoWatcher SaaS. it is a concurrent background monitoring engine that tracks translation platforms for new opportunities.

## How it Works

The system consists of two main monitoring components: the **RSS Monitor** and the **WebSocket Monitor**.

### 1. RSS Monitoring
The RSS Monitor polls the platform's job feeds at regular intervals.
- **Polling Frequency**: Depends on the user's subscription tier (e.g., every 30 seconds for Pro, 2 minutes for Free).
- **Deduplication**: The system stores a hash of recently seen jobs to prevent duplicate notifications.
- **Failover**: If a poll fails, the system retries with exponential backoff.

### 2. WebSocket Monitoring
For platforms that support it, the WebSocket Monitor provides near-instantaneous job discovery.
- **Live Stream**: Connects directly to the platform's public or private job stream.
- **Low Latency**: Discovers jobs within milliseconds of them being posted.

---

## The Matching Engine

Once a job is discovered by either monitor, it passes through the Matching Engine.

### Filtering Criteria
Users can configure several filters to ignore irrelevant jobs:
- **Language Pairs**: Only show jobs for specific source and target languages (e.g., `en -> ja`).
- **Reward Range**: Filter jobs by their payout (e.g., `$5.00` to `$100.00`).
- **Keywords**: (Pro Feature) Match or exclude specific words in the job title or description.

### Match Logic
```text
If (Job.LanguagePair matches User.IncludedPairs) AND
   (Job.Reward >= User.MinReward) AND
   (Job.Reward <= User.MaxReward)
THEN
   ProcessMatch(Job)
```

---

## Notification Pipeline

When a match is found, the system triggers the notification pipeline:

1. **Database Update**: The job is logged in the user's discovery history.
2. **Real-time Push**: A message is sent via Redis to the user's active WebSocket session.
3. **Email Alert**: (If enabled) An email is dispatched via Resend.
4. **Desktop/Browser Notification**: (If enabled) The frontend triggers a native browser notification.

---

## Auto-Accept (Pro/Enterprise)

The Auto-Accept feature allows users to automatically claim jobs that meet high-priority criteria.
- **Strict Filters**: Users can set separate, stricter filters for auto-accept (e.g., only auto-accept jobs over `$20.00`).
- **Safety Limits**: Users can set daily or hourly limits on how many jobs to auto-accept to prevent overloading their workload.

## State Management

The `WatcherManager` tracks the health and status of every active watcher:
- **Running**: Actively monitoring and matching.
- **Stopped**: Paused by the user.
- **Error**: Encountered an issue (e.g., invalid RSS URL or authentication error). Users are notified immediately of any errors.

## Next Steps
- [Real-Time Notifications](../core-concepts/real-time-notifications.md)
- [Subscription Tiers](../core-concepts/subscription-tiers.md)
- [Watcher API Reference](../api/watcher-endpoints.md)
