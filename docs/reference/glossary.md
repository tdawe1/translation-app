# Glossary

Terms and definitions used within the GengoWatcher SaaS ecosystem.

| Term | Definition |
|------|------------|
| **Auto-Accept** | A Pro/Enterprise feature that automatically claims translation jobs matching high-priority criteria. |
| **Bento Style** | A design language that organizes UI elements into clear, distinct rectangular cards or blocks. |
| **Found Job** | A translation job discovered by a watcher that matched a user's language and reward filters. |
| **Gengo** | The primary translation platform monitored by GengoWatcher. |
| **JWT** | JSON Web Token; used for short-term authentication in API requests. |
| **LemonSqueezy** | Our third-party payment processor and merchant of record. |
| **Magic Link** | A secure, one-time login link sent via email to provide passwordless authentication. |
| **Multi-Tenancy** | An architecture where multiple independent users share the same infrastructure while their data is strictly isolated. |
| **Provider** | An external authentication source (e.g., Google, GitHub). |
| **Pub/Sub** | Publish/Subscribe; a messaging pattern used in Redis to broadcast job alerts to WebSocket handlers. |
| **Resend** | Our third-party transactional email service provider. |
| **RSS Monitor** | A background process that periodically polls RSS feeds for new jobs. |
| **Watcher** | A background process (RSS or WebSocket) that monitors platforms for new jobs on behalf of a user. |
| **WebSocket** | A persistent bi-directional communication protocol used for real-time dashboard updates. |

## Next Steps
- [Architecture Overview](../core-concepts/architecture-overview.md)
- [Watcher System](../core-concepts/watcher-system.md)
