# Webhook Handling Guide

GengoWatcher SaaS uses webhooks to receive real-time updates from third-party services like **LemonSqueezy** (for billing) and potentially translation platforms.

## What are Webhooks?
Webhooks are "reverse APIs." Instead of you calling our API, we (or a third party) send an HTTP POST request to your pre-defined URL when an event occurs.

---

## Billing Webhooks (LemonSqueezy)

We receive webhooks from LemonSqueezy to update your subscription status instantly.

### Supported Events
| Event | Action Taken |
|-------|--------------|
| `subscription_created` | Activates Pro/Enterprise features for the user. |
| `subscription_updated` | Updates tier or billing cycle. |
| `subscription_cancelled` | Schedules feature deactivation for the end of the period. |
| `subscription_expired` | Immediately reverts user to the Free tier. |

---

## Security & Verification

To ensure that a webhook actually came from a trusted source, you **must** verify the signature.

### How to Verify (Go Example)
The backend verifies the `X-Signature` header using the `LEMONSQUEEZY_WEBHOOK_SECRET`.

```go
func VerifySignature(payload []byte, signature string, secret string) bool {
    h := hmac.New(sha256.New, []byte(secret))
    h.Write(payload)
    expected := hex.EncodeToString(h.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

---

## Best Practices for Webhook Handlers

1. **Return 200 Fast**: Acknowledge receipt immediately. Perform heavy processing (like sending emails) in a background job.
2. **Idempotency**: Webhooks may be sent more than once. Always check if you've already processed a specific event ID.
3. **Handle Retries**: If your server is down, providers will usually retry with exponential backoff. Ensure your endpoint is highly available.
4. **Log Everything**: Store raw webhook payloads in an `audit_logs` or `billing_events` table for debugging purposes.

---

## Testing Webhooks Locally

Since your local server isn't accessible from the internet, use a tool like **ngrok** to tunnel requests.

1. **Start ngrok**:
   ```bash
   ngrok http 8000
   ```
2. **Update Provider**: Copy the ngrok URL (e.g., `https://xyz.ngrok.io/api/v1/billing/webhook`) into your LemonSqueezy dashboard.
3. **Trigger Event**: Use the provider's test tool to send a mock webhook.

## Next Steps
- [Subscription Tiers](../core-concepts/subscription-tiers.md)
- [Billing Configuration](../configuration/billing-configuration.md)
