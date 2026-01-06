# Introduction to GengoWatcher SaaS

GengoWatcher is a professional, multi-tenant job monitoring platform designed to help freelance translators never miss a high-value job opportunity. By combining real-time RSS polling and high-frequency WebSocket monitoring, GengoWatcher provides instantaneous alerts when new jobs matching your specific criteria appear on the platform.

## What is GengoWatcher?

GengoWatcher transforms the way translators interact with job platforms. Instead of manual refreshing and constant monitoring, GengoWatcher acts as your digital scout, working 24/7 to identify and notify you of jobs that fit your profile.

### Value Proposition

- **Speed to Market**: Get notified within seconds of a job being posted. In the translation world, being first often means getting the job.
- **Precision Filtering**: Set complex criteria including reward ranges, language pairs, and keywords to only see the jobs you want.
- **Automation**: Use auto-accept features (available in Pro/Enterprise tiers) to automatically claim jobs that meet your highest-priority criteria.
- **Ubiquity**: Receive notifications across desktop, mobile (via email), and web dashboard simultaneously.

## Key Features

### Multi-Tenant Watcher Instances
Each user on the platform receives their own isolated watcher instance. This ensures that your specific filters and monitoring settings are handled independently and securely, with no cross-user data leakage.

### Real-Time Realism
Leveraging a combination of:
- **RSS Monitoring**: Reliable polling of job feeds.
- **WebSocket Integration**: Low-latency, real-time job discovery.
- **Redis Pub/Sub**: Internal message bus that pushes updates to your connected devices instantly.

### Flexible Authentication
Securely access your account using:
- Traditional Email/Password
- Passwordless Magic Links
- Social Login (Google, GitHub)
- Programmatic API Keys (for developers)

### Subscription-Based Scaling
From individual freelancers to translation agencies, GengoWatcher scales with you:
- **Free**: Essential monitoring for 1 watcher.
- **Pro**: Advanced filtering, higher frequency, and auto-accept.
- **Enterprise**: Unlimited watchers and priority job processing.

## How it Works

1. **Connect**: Link your account and configure your monitoring preferences.
2. **Monitor**: GengoWatcher's background system polls job sources in real-time.
3. **Filter**: Jobs are analyzed against your rewards, languages, and settings.
4. **Notify**: When a match is found, you are alerted via WebSocket, Email, or Browser notifications.
5. **Accept**: Claim the job directly from the notification or let the auto-accept system handle it for you.

## Next Steps

- [Quick Start Guide](../getting-started/quick-start.md) - Get up and running in 5 minutes.
- [Architecture Overview](../core-concepts/architecture-overview.md) - Learn how the system is built.
- [Subscription Tiers](../core-concepts/subscription-tiers.md) - Choose the right plan for your needs.
