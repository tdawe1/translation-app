# Billing Configuration (LemonSqueezy)

GengoWatcher SaaS integrates with **LemonSqueezy** for subscriptions, product management, and checkout.

## 1. Prerequisites
1. Create a **LemonSqueezy** account.
2. Set up a **Store** and create your **Products** (Free, Pro, Enterprise).
3. Create **Variants** for Monthly and Yearly plans.

---

## 2. Environment Variables

| Variable | Description |
|----------|-------------|
| `LEMONSQUEEZY_WEBHOOK_SECRET` | Used to verify that incoming webhooks are authentic. |
| `LEMONSQUEEZY_API_KEY` | Your LemonSqueezy API key (for programmatic management). |

---

## 3. Webhook Setup

You must configure LemonSqueezy to send webhooks to your production API.

1. In LemonSqueezy, go to **Settings > Webhooks**.
2. Click **Add Webhook**.
3. **Payload URL**: `https://api.gengowatcher.com/api/v1/billing/webhook`.
4. **Secret**: Enter a long, random string and copy it to `LEMONSQUEEZY_WEBHOOK_SECRET` in your `.env`.
5. **Events**: Select at least:
   - `subscription_created`
   - `subscription_updated`
   - `subscription_cancelled`
   - `subscription_expired`

---

## 4. Product ID Mapping

In the backend, we map LemonSqueezy Product/Variant IDs to our internal subscription tiers.

```go
var TierMap = map[string]string{
    "variant_id_123": "pro",
    "variant_id_456": "enterprise",
}
```

Ensure these IDs match the ones provided in your LemonSqueezy dashboard.

---

## 5. Testing (Sandbox Mode)

1. Use LemonSqueezy's **Test Mode**.
2. Use their provided test card numbers to simulate successful and failed payments.
3. Use a tool like `ngrok` to receive webhooks on your local development machine.

---

## 6. Security

- **Signature Verification**: Every webhook request is verified using the SHA-256 HMAC of the request body and your secret.
- **Idempotency**: We store `event_id`s to ensure that duplicate webhooks don't cause multiple subscription updates.

## Next Steps
- [Subscription Tiers](../core-concepts/subscription-tiers.md)
- [Webhook Handling Guide](../guides/webhook-handling.md)
