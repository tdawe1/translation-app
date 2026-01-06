# Configuring Auto-Accept Rules

**Available in Pro and Enterprise tiers.**

Auto-Accept is GengoWatcher's most powerful feature, allowing the system to automatically claim jobs on your behalf the moment they are discovered.

## Prerequisites
1. **Subscription**: A valid Pro or Enterprise subscription.
2. **Platform Credentials**: Your translation platform's API keys or session tokens must be configured in **Settings > Platform**.
3. **Email Verification**: Your email must be verified.

---

## Setting Up Auto-Accept

### 1. Enable Global Switch
Navigate to **Watcher Settings** and toggle the **"Enable Auto-Accept"** switch.

### 2. Define Auto-Accept Filters
It is highly recommended to use stricter filters for auto-accept than for general notifications.

| Filter | Recommended Setting | Purpose |
|--------|---------------------|---------|
| **Min Reward** | Higher than standard | Only claim high-value jobs automatically. |
| **Max Word Count** | Moderate | Avoid accidentally claiming a 50,000-word book. |
| **Required Rating** | `Standard` or `Pro` | Ensure you only take jobs you are qualified for. |

### 3. Safety Limits (Quotas)
To prevent overwhelming your schedule, set safety limits:
- **Max Jobs Per Day**: e.g., `5`.
- **Max Pending Jobs**: e.g., `2`. The system won't accept more if you already have unfinished jobs.

---

## How the Process Works

1. **Job Found**: Watcher identifies a new job.
2. **Standard Match**: Job matches your notification filters → **Notification Sent**.
3. **Auto-Accept Match**: Job also matches your stricter auto-accept filters.
4. **Execution**: GengoWatcher sends an `ACCEPT` request to the platform.
5. **Confirmation**: You receive a special notification: **"Job #123 Auto-Accepted!"**

---

## Best Practices & Warnings

### ⚠️ Responsibility
You are responsible for any job the system auto-accepts. Most platforms have penalties for dropping jobs once accepted.

### Start Strict
When first using auto-accept, set your **Minimum Reward** very high and your **Daily Limit** to 1. Once you trust your filters, you can gradually expand them.

### Use Keywords
Combine auto-accept with positive keywords like `Medical` or `Legal` to ensure you only automatically take jobs within your strongest expertise.

---

## Troubleshooting

### Job "Already Taken"
Even with 10-second polling, another translator might manually claim the job before the auto-accept request completes. This will be logged as a "Missed Match" in your dashboard.

### Authentication Errors
If your platform credentials expire, auto-accept will fail. Ensure your platform connection status is "Healthy" in settings.

## Next Steps
- [Advanced Filtering Guide](../guides/advanced-filtering.md)
- [Subscription Tiers](../core-concepts/subscription-tiers.md)
