# Database Schema Documentation

## Overview

The GengoWatcher SaaS uses PostgreSQL 17 as its primary database. GORM 2.0 is used as the ORM layer.

## Database Connection

### Connection String Format

```
postgresql://username:password@host:port/database?sslmode=require
```

### Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| DB_HOST | PostgreSQL host | localhost |
| DB_PORT | PostgreSQL port | 5432 |
| DB_USER | Database user | gengo |
| DB_PASSWORD | Database password | devpass |
| DB_NAME | Database name | gengowatcher |
| DB_SSLMODE | SSL mode | require (prod) / disable (dev) |

## Schema Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                users                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│ id                    │ UUID              │ PRIMARY KEY                     │
│ email                 │ VARCHAR(255)      │ UNIQUE, NOT NULL                │
│ email_verified        │ BOOLEAN           │ DEFAULT FALSE                   │
│ password_hash         │ VARCHAR(255)      │                                 │
│ is_active             │ BOOLEAN           │ DEFAULT TRUE                    │
│ created_at            │ TIMESTAMP         │ DEFAULT NOW()                   │
│ updated_at            │ TIMESTAMP         │                                 │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
        ┌───────────▼───┐ ┌───────────▼───┐ ┌───────▼───────┐
        │ oauth_accounts│ │magic_link_   │ │email_verify_ │
        │               │ │tokens        │ │tokens        │
        ├───────────────┤ ├───────────────┤ ├─────────────┤
        │ user_id (FK)  │ │ email         │ │ email       │
        │ provider      │ │ token         │ │ token       │
        │ provider_user │ │ expires_at    │ │ expires_at  │
        │ created_at    │ │ used_at       │ │ used_at     │
        └───────────────┘ └───────────────┘ └─────────────┘
                    │
        ┌───────────▼───┐
        │ password_reset│
        │tokens         │
        ├───────────────┤
        │ email         │
        │ token         │
        │ expires_at    │
        │ used_at       │
        └───────────────┘
                    │
        ┌───────────▼───┐
        │refresh_tokens │
        ├───────────────┤
        │ user_id (FK)  │
        │ token_hash    │
        │ expires_at    │
        │ created_at    │
        │ revoked_at    │
        └───────────────┘
                    │
        ┌───────────▼───┐
        │ api_keys      │
        ├───────────────┤
        │ user_id (FK)  │
        │ name          │
        │ key_hash      │
        │ last_used_at  │
        │ expires_at    │
        └───────────────┘
                    │
        ┌───────────▼───┐
        │watcher_configs│
        ├───────────────┤
        │ user_id (FK)  │ ◀── One-to-One
        │ rss_feed_url  │
        │ min_reward    │
        │ max_reward    │
        │ ...           │
        └───────────────┘
                    │
        ┌───────────▼───┐
        │ watcher_states│
        ├───────────────┤
        │ user_id (FK)  │ ◀── One-to-One
        │ status        │
        │ jobs_found    │
        │ last_poll     │
        └───────────────┘
```

## Table Definitions

### users

Primary user table storing authentication and profile information.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique user identifier |
| email | VARCHAR(255) | UNIQUE, NOT NULL | User's email address |
| email_verified | BOOLEAN | DEFAULT FALSE | Whether email is verified |
| password_hash | VARCHAR(255) | NULL if OAuth-only | Bcrypt hashed password |
| is_active | BOOLEAN | DEFAULT TRUE | Account active status |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMP | | Last update timestamp |

**Indexes:**
- `idx_users_email` (UNIQUE) - Fast user lookup by email
- `idx_users_email_verified` - Filter verified users

---

### oauth_accounts

Links users to OAuth providers (Google, GitHub).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| user_id | UUID | REFERENCES users(id) | Owner user |
| provider | VARCHAR(50) | NOT NULL | OAuth provider ('google', 'github') |
| provider_user_id | VARCHAR(255) | NOT NULL | User ID from provider |
| created_at | TIMESTAMP | DEFAULT NOW() | Link creation time |

**Indexes:**
- `idx_oauth_provider_user` (UNIQUE) - `(provider, provider_user_id)` - Fast OAuth lookup
- `idx_oauth_user_id` - Find all accounts for user

---

### magic_link_tokens

One-time tokens for passwordless authentication via email.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| email | VARCHAR(255) | NOT NULL | Target email address |
| token | VARCHAR(255) | UNIQUE, NOT NULL | One-time use token |
| expires_at | TIMESTAMP | NOT NULL | Token expiration (15 min) |
| used_at | TIMESTAMP | NULL | When token was used |

**Indexes:**
- `idx_magic_link_token` (UNIQUE) - Token validation
- `idx_magic_link_email` - Find tokens by email
- `idx_magic_link_expires` - Cleanup expired tokens

---

### email_verification_tokens

Email verification tokens for new accounts.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| email | VARCHAR(255) | NOT NULL | Target email address |
| token | VARCHAR(255) | UNIQUE, NOT NULL | Verification token |
| expires_at | TIMESTAMP | NOT NULL | Token expiration (24 hours) |
| used_at | TIMESTAMP | NULL | When token was used |

**Indexes:**
- `idx_email_verify_token` (UNIQUE) - Token validation
- `idx_email_verify_email` - Find tokens by email
- `idx_email_verify_expires` - Cleanup expired tokens

---

### password_reset_tokens

Password reset request tokens.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| email | VARCHAR(255) | NOT NULL | Target email address |
| token | VARCHAR(255) | UNIQUE, NOT NULL | Reset token |
| expires_at | TIMESTAMP | NOT NULL | Token expiration (1 hour) |
| used_at | TIMESTAMP | NULL | When token was used |

**Indexes:**
- `idx_password_reset_token` (UNIQUE) - Token validation
- `idx_password_reset_email` - Find tokens by email
- `idx_password_reset_expires` - Cleanup expired tokens

---

### refresh_tokens

JWT refresh tokens for session management.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| user_id | UUID | REFERENCES users(id) | Owner user |
| token_hash | VARCHAR(255) | NOT NULL | Hashed token |
| expires_at | TIMESTAMP | NOT NULL | Token expiration (7 days) |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation time |
| revoked_at | TIMESTAMP | NULL | When token was revoked |

**Indexes:**
- `idx_refresh_token_user_id` - Find all tokens for user
- `idx_refresh_token_expires` - Cleanup expired tokens

---

### api_keys

User API keys for programmatic access.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| user_id | UUID | REFERENCES users(id) | Owner user |
| name | VARCHAR(100) | NOT NULL | Human-readable name |
| key_hash | VARCHAR(255) | UNIQUE, NOT NULL | Hashed key prefix |
| last_used_at | TIMESTAMP | NULL | Last usage time |
| expires_at | TIMESTAMP | NULL | Optional expiration |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation time |

**Indexes:**
- `idx_api_key_user_id` - Find all keys for user
- `idx_api_key_key_hash` (UNIQUE) - Key validation

---

### watcher_configs

User configuration for job monitoring.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| user_id | UUID | REFERENCES users(id), UNIQUE | Owner user |
| rss_feed_url | VARCHAR(2048) | NULL | RSS feed URL |
| websocket_enabled | BOOLEAN | DEFAULT TRUE | Enable WebSocket |
| gengo_user_id | VARCHAR(100) | NULL | Gengo user ID |
| min_reward | DECIMAL(10,2) | DEFAULT 0.00 | Minimum reward filter |
| max_reward | DECIMAL(10,2) | NULL | Maximum reward filter |
| included_language_pairs | JSONB | NULL | `["en-ja", "en-es"]` |
| enable_desktop_notifs | BOOLEAN | DEFAULT FALSE | Desktop notifications |
| enable_sound_notifs | BOOLEAN | DEFAULT FALSE | Sound notifications |
| enable_email_notifs | BOOLEAN | DEFAULT TRUE | Email notifications |
| notification_email | VARCHAR(255) | NULL | Custom notification email |
| auto_accept_enabled | BOOLEAN | DEFAULT FALSE | Auto-accept jobs |
| auto_accept_min_reward | DECIMAL(10,2) | NULL | Auto-accept minimum |
| auto_accept_max_reward | DECIMAL(10,2) | NULL | Auto-accept maximum |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation time |
| updated_at | TIMESTAMP | | Last update time |

**Indexes:**
- `idx_watcher_config_user_id` (UNIQUE) - Find config by user

---

### watcher_states

Current state of user's job watcher.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| user_id | UUID | REFERENCES users(id), UNIQUE | Owner user |
| watcher_status | VARCHAR(20) | DEFAULT 'stopped' | `stopped`, `running`, `error` |
| last_poll | TIMESTAMP | NULL | Last RSS poll time |
| last_error | TEXT | NULL | Last error message |
| jobs_found | INTEGER | DEFAULT 0 | Total jobs found |
| jobs_accepted | INTEGER | DEFAULT 0 | Jobs auto-accepted |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation time |
| updated_at | TIMESTAMP | | Last update time |

**Indexes:**
- `idx_watcher_state_user_id` (UNIQUE) - Find state by user

---

### subscriptions (Future)

Subscription and billing information.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| user_id | UUID | REFERENCES users(id) | Owner user |
| plan_id | UUID | REFERENCES subscription_plans(id) | Subscription plan |
| status | VARCHAR(20) | NOT NULL | `active`, `cancelled`, `past_due` |
| starts_at | TIMESTAMP | NOT NULL | Subscription start |
| expires_at | TIMESTAMP | NOT NULL | Subscription expiry |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation time |

---

### subscription_plans (Future)

Available subscription tiers.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| name | VARCHAR(50) | NOT NULL | `free`, `pro`, `enterprise` |
| price_monthly | DECIMAL(10,2) | NOT NULL | Monthly price |
| price_yearly | DECIMAL(10,2) | NOT NULL | Yearly price |
| max_watchers | INTEGER | NOT NULL | Maximum watchers |
| max_jobs_per_day | INTEGER | NOT NULL | Daily job limit |
| auto_accept | BOOLEAN | DEFAULT FALSE | Auto-accept enabled |
| created_at | TIMESTAMP | DEFAULT NOW() | Creation time |

---

### billing_events (Future)

Stripe/LemonSqueezy webhook events.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| user_id | UUID | REFERENCES users(id) | Owner user |
| subscription_id | UUID | REFERENCES subscriptions(id) | Related subscription |
| event_type | VARCHAR(50) | NOT NULL | Event type |
| event_data | JSONB | NOT NULL | Raw webhook data |
| processed_at | TIMESTAMP | NULL | Processing timestamp |
| created_at | TIMESTAMP | DEFAULT NOW() | Event time |

---

### audit_logs (Future)

User action audit trail.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique identifier |
| user_id | UUID | REFERENCES users(id) | Actor |
| action | VARCHAR(100) | NOT NULL | Action performed |
| resource_type | VARCHAR(50) | NOT NULL | Affected resource type |
| resource_id | UUID | NULL | Affected resource ID |
| ip_address | INET | NULL | Client IP |
| user_agent | VARCHAR(500) | NULL | Client user agent |
| metadata | JSONB | NULL | Additional context |
| created_at | TIMESTAMP | DEFAULT NOW() | Event time |

---

## Database Migrations

### Migration Files

| File | Description |
|------|-------------|
| `001_initial_schema.py` | Initial table creation |
| `002_add_auth_indexes.py` | Performance indexes for auth |

### Running Migrations

```bash
# Apply all pending migrations
alembic upgrade head

# Rollback one migration
alembic downgrade -1

# Show current migration
alembic current

# Show migration history
alembic history
```

---

## User Isolation

All queries are filtered by `user_id` to ensure data isolation:

```go
// Get user's watcher config
var config models.WatcherConfig
err := db.Where("user_id = ?", userID).First(&config).Error
```

## Performance Considerations

### Indexes

The following indexes ensure optimal query performance:

| Table | Index | Columns | Type |
|-------|-------|---------|------|
| users | idx_users_email | email | UNIQUE |
| users | idx_users_email_verified | email_verified | |
| oauth_accounts | idx_oauth_provider_user | provider, provider_user_id | UNIQUE |
| oauth_accounts | idx_oauth_user_id | user_id | |
| magic_link_tokens | idx_magic_link_token | token | UNIQUE |
| magic_link_tokens | idx_magic_link_expires | expires_at | |
| email_verification_tokens | idx_email_verify_token | token | UNIQUE |
| refresh_tokens | idx_refresh_token_user_id | user_id | |
| watcher_configs | idx_watcher_config_user_id | user_id | UNIQUE |
| watcher_states | idx_watcher_state_user_id | user_id | UNIQUE |

---

**Last Updated**: January 2026
**Version**: 1.0.0
