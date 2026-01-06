# Database Schema Reference

This reference provides detailed information about every table in the GengoWatcher SaaS database.

## User Management Tables

### `users`
The core table for user accounts.
- `id` (UUID, PK): Unique user identifier.
- `email` (String, Unique): User's primary email address.
- `password_hash` (String): Argon2id hashed password.
- `email_verified` (Boolean): Activation status.
- `tier` (String): `free`, `pro`, or `enterprise`.
- `is_active` (Boolean): Administrative account status.

### `oauth_accounts`
Links users to Google/GitHub.
- `id` (UUID, PK)
- `user_id` (UUID, FK): Owner user.
- `provider` (String): `google`, `github`.
- `provider_user_id` (String): ID from the social provider.

---

## Watcher & Job Tables

### `watcher_configs`
Stores user-specific monitoring settings.
- `user_id` (UUID, FK, Unique): One configuration per user.
- `rss_feed_url` (String)
- `min_reward` (Decimal)
- `max_reward` (Decimal)
- `included_language_pairs` (JSONB): Array of language strings.
- `enable_email_notifs` (Boolean)
- `auto_accept_enabled` (Boolean)

### `watcher_states`
Stores the runtime status of a user's watcher.
- `user_id` (UUID, FK, Unique)
- `status` (String): `running`, `stopped`, `error`.
- `last_poll_at` (Timestamp)
- `jobs_found_total` (Integer)

### `found_jobs`
A log of all jobs matching a user's criteria.
- `id` (String, PK): The platform's job ID.
- `user_id` (UUID, FK)
- `title` (String)
- `reward` (Decimal)
- `status` (String): `new`, `accepted`, `ignored`.
- `found_at` (Timestamp)

---

## System Tables

### `audit_logs`
A permanent trail of system actions.
- `id` (UUID, PK)
- `user_id` (UUID, FK, Nullable)
- `action` (String)
- `metadata` (JSONB): Contextual information.
- `ip_address` (String)
- `created_at` (Timestamp)

### `refresh_tokens`
Used for JWT session rotation.
- `id` (UUID, PK)
- `user_id` (UUID, FK)
- `token_hash` (String, Unique)
- `expires_at` (Timestamp)
- `revoked_at` (Timestamp, Nullable)

## Next Steps
- [Relationships & ERD](../database/relationships.md)
- [Database Performance](../database/performance-optimization.md)
