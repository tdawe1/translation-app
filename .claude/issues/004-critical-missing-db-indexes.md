# Missing Database Indexes

**Priority**: P0 (Critical) | **Status**: Pending | **Assigned**: Unassigned

## Summary

Critical database queries lack indexes, causing full table scans on every OAuth login and email verification.

## Location

- File: `backend/internal/models/`
- Tables: `users`, `oauth_accounts`, `magic_link_tokens`, `email_verification_tokens`

## Problem

```go
// Full table scan on every OAuth login
err := s.db.Where("provider = ? AND provider_user_id = ?", string(provider), userInfo.ID).
    First(&oauthAccount).Error

// Full table scan on user lookup
err := s.db.Where("email = ?", userInfo.Email).First(&user).Error
```

## Required Indexes

```sql
-- Users table
CREATE UNIQUE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_email_verified ON users(email_verified);

-- OAuth accounts
CREATE UNIQUE INDEX idx_oauth_provider_user ON oauth_accounts(provider, provider_user_id);
CREATE INDEX idx_oauth_user_id ON oauth_accounts(user_id);

-- Magic link tokens
CREATE UNIQUE INDEX idx_magic_link_token ON magic_link_tokens(token);
CREATE INDEX idx_magic_link_expires ON magic_link_tokens(expires_at);

-- Email verification tokens
CREATE UNIQUE INDEX idx_email_verify_token ON email_verification_tokens(token);
CREATE INDEX idx_email_verify_expires ON email_verification_tokens(expires_at);

-- Password reset tokens
CREATE UNIQUE INDEX idx_password_reset_token ON password_reset_tokens(token);
CREATE INDEX idx_password_reset_expires ON password_reset_tokens(expires_at);
```

## Migration

Create `alembic/versions/XXX_add_auth_indexes.py`:

```python
from alembic import op
import sqlalchemy as sa

def upgrade():
    # Users
    op.create_index('idx_users_email', 'users', ['email'], unique=True)
    # ... rest of indexes

def downgrade():
    op.drop_index('idx_users_email', table_name='users')
    # ... rest of indexes
```

## Acceptance

- [ ] Migration file created
- [ ] All indexes applied to database
- [ ] EXPLAIN ANALYZE shows index usage
- [ ] No full table scans in auth flow

## Related

- #015 (Race condition in user lookup - also solved by unique index)
