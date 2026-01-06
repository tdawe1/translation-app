# Subscription Tiers & Pricing

GengoWatcher SaaS offers flexible pricing plans designed to grow with your freelance translation career. We use **LemonSqueezy** for secure payment processing and subscription management.

## Plan Comparison

| Feature | Free | Pro | Enterprise |
|---------|------|-----|------------|
| **Price** | $0/mo | $29/mo | $99/mo |
| **Watcher Instances** | 1 | 5 | Unlimited |
| **Polling Interval** | 2 minutes | 30 seconds | 10 seconds |
| **WebSocket Support** | No | Yes | Yes |
| **Auto-Accept** | No | Yes | Yes |
| **Advanced Filtering** | Basic | Complete | Complete |
| **Email Support** | Community | Priority | 24/7 Dedicated |

---

## Plan Details

### 1. Free Tier ($0)
Perfect for getting started or occasional monitoring.
- **Basic Monitoring**: Monitor one RSS feed.
- **Email Alerts**: Basic email notifications for matches.
- **Dashboard Access**: View job history and current status.

### 2. Pro Tier ($29/mo)
Designed for full-time freelance translators who need every advantage.
- **High Frequency**: Watchers poll 4x more often than the Free tier.
- **Instant WebSockets**: Get jobs pushed to your browser the millisecond they are found.
- **Auto-Accept**: Configure rules to automatically claim the best jobs before anyone else sees them.
- **Keyword Filtering**: Filter by specific domain terms (e.g., "Medical", "Legal", "Gaming").

### 3. Enterprise Tier ($99/mo)
For agencies or power users managing multiple accounts and high volumes.
- **Maximum Speed**: 10-second polling for ultra-competitive markets.
- **API Access**: Use our REST API to build your own custom tools and integrations.
- **Dedicated Infrastructure**: Higher rate limits and priority processing.

---

## Subscription Management

### Upgrading/Downgrading
You can change your plan at any time from the **Billing** section of your settings. Upgrades take effect immediately (pro-rated), while downgrades take effect at the end of your current billing cycle.

### Billing Cycle
We offer both **Monthly** and **Annual** billing. Annual plans typically include a 2-month discount (pay for 10 months, get 12).

### Payment Security
GengoWatcher **never stores your credit card information**. All payments are handled by LemonSqueezy, a PCI-compliant payment processor. We only store a reference to your subscription status and ID.

---

## Limits & Quotas

To ensure system stability, all tiers are subject to fair use limits:
- **API Requests**: 100 per minute (Pro), 500 per minute (Enterprise).
- **Daily Job Limit**: 1,000 matches per day (Pro), 5,000 (Enterprise).

## Next Steps
- [Architecture Overview](../core-concepts/architecture-overview.md)
- [Multi-Tenancy](../core-concepts/multi-tenancy.md)
- [Billing Configuration](../configuration/billing-configuration.md)
