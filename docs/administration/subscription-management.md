# Subscription Management

GengoWatcher SaaS integrates with **LemonSqueezy** to handle billing and subscriptions. This guide covers how to manage these subscriptions from an administrative perspective.

## 1. Subscription Lifecycle

Subscriptions transition through several states:

- **`on_trial`**: User is in the initial trial period (if applicable).
- **`active`**: Payments are up to date, features enabled.
- **`past_due`**: A payment failed; we provide a 3-day grace period before downgrading.
- **`unpaid`**: Multiple payment attempts failed. Features disabled.
- **`cancelled`**: User has stopped their subscription; features remain active until the end of the current period.
- **`expired`**: Subscription period ended. User is reverted to the Free tier.

---

## 2. Handling Tier Upgrades

When a user upgrades (e.g., Free to Pro):
1. **Webhook Received**: LemonSqueezy sends a `subscription_created` event.
2. **Database Update**: The `user_tier` is updated in our database.
3. **Session Refresh**: The user's next API request will reflect the new permissions.
4. **Watcher Limits**: The system immediately allows the user to start up to 5 watchers.

---

## 3. Grace Periods & Failures

If a subscription enters `past_due`:
1. **Notification**: Send an automated email to the user: "Payment Failed - Action Required".
2. **Wait**: Allow 72 hours for the user to update their card.
3. **Downgrade**: If still unpaid, stop all watchers except the first one and set the tier back to `free`.

---

## 4. Manual Overrides

Administrators can manually set a user's subscription tier for support or promotional purposes.

**⚠️ Warning**: Manual overrides may be overwritten by the next automated webhook from LemonSqueezy unless the billing link is also updated.

---

## 5. Refunds and Cancellations

Refunds must be processed via the LemonSqueezy dashboard.
- **Full Refund**: Reverts the user to Free tier immediately.
- **Cancellation**: Features remain until `expires_at`.

## Next Steps
- [User Management](../administration/user-management.md)
- [Billing Configuration](../configuration/billing-configuration.md)
