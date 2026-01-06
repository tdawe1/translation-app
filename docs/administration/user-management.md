# User Management (Admin)

This guide covers administrative tasks for managing GengoWatcher SaaS users, intended for internal support teams and administrators.

## 1. Viewing User Profiles

Administrators can retrieve user details (excluding passwords) via the admin API or internal dashboard.

**Key Information:**
- **Status**: `active`, `suspended`, `pending_verification`.
- **Tier**: `free`, `pro`, `enterprise`.
- **Created Date**: When the account was registered.
- **Last Login**: Timestamp of last successful authentication.

---

## 2. Suspending/Activating Users

If a user violates terms of service (e.g., using unauthorized automation or failing to pay), their account can be suspended.

### Suspension Effect
- **Login**: Prevented immediately.
- **Watchers**: All active watcher processes are forcibly stopped.
- **Sessions**: All active refresh tokens are revoked.

---

## 3. Email Verification (Manual)

In rare cases where a user cannot receive the verification email, an administrator can manually verify the account.

**Procedure:**
1. Confirm the user's identity via support ticket.
2. Update the `email_verified` flag in the database to `true`.
3. Clear any pending `email_verification_tokens`.

---

## 4. Troubleshooting User Accounts

### Missing Jobs
If a user reports they aren't seeing jobs:
1. Check **Watcher State**: Is it `running` or `error`?
2. Validate **RSS URL**: Use a tool to verify the feed is active.
3. Check **Filters**: Often, filters are set too strictly (e.g., min reward too high).

### Login Issues
If a user can't log in:
1. Verify the account isn't **Suspended**.
2. Check if they are using the correct **Provider** (Google vs. GitHub).
3. Check for **Rate Limiting** on their IP address.

---

## 5. Audit Logging

Every administrative action is logged in the `audit_logs` table.
- **Actor**: The admin ID.
- **Action**: e.g., `user_suspended`.
- **Target**: The user ID.
- **Timestamp**: When it occurred.

## Next Steps
- [Subscription Management](../administration/subscription-management.md)
- [Monitoring & Metrics](../administration/monitoring.md)
