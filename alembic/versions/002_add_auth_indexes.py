"""Add authentication indexes for performance

Revision ID: 002_add_auth_indexes
Revises: 001_initial_schema
Create Date: 2026-01-05

"""

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = "002_add_auth_indexes"
down_revision = "003_add_auth_token_tables"
branch_labels = None
depends_on = None


def upgrade():
    """Add indexes for authentication-related tables to improve query performance."""

    # Users table indexes
    op.create_index("idx_users_email", "users", ["email"], unique=True)
    op.create_index("idx_users_email_verified", "users", ["email_verified"])

    # OAuth accounts indexes
    op.create_index(
        "idx_oauth_provider_user",
        "oauth_accounts",
        ["provider", "provider_user_id"],
        unique=True,
    )
    op.create_index("idx_oauth_user_id", "oauth_accounts", ["user_id"])

    # Magic link tokens indexes
    op.create_index("idx_magic_link_token", "magic_link_tokens", ["token"], unique=True)
    op.create_index("idx_magic_link_email", "magic_link_tokens", ["email"])
    op.create_index("idx_magic_link_expires", "magic_link_tokens", ["expires_at"])

    # Email verification tokens indexes
    op.create_index(
        "idx_email_verify_token", "email_verification_tokens", ["token"], unique=True
    )
    op.create_index("idx_email_verify_email", "email_verification_tokens", ["email"])
    op.create_index(
        "idx_email_verify_expires", "email_verification_tokens", ["expires_at"]
    )

    # Password reset tokens indexes
    op.create_index(
        "idx_password_reset_token", "password_reset_tokens", ["token"], unique=True
    )
    op.create_index("idx_password_reset_email", "password_reset_tokens", ["email"])
    op.create_index(
        "idx_password_reset_expires", "password_reset_tokens", ["expires_at"]
    )

    # Refresh tokens indexes
    op.create_index("idx_refresh_token_user_id", "refresh_tokens", ["user_id"])
    op.create_index("idx_refresh_token_expires", "refresh_tokens", ["expires_at"])

    # API keys indexes
    op.create_index("idx_api_key_user_id", "api_keys", ["user_id"])
    op.create_index("idx_api_key_key_hash", "api_keys", ["key_hash"], unique=True)


def downgrade():
    """Remove authentication indexes."""

    # API keys indexes
    op.drop_index("idx_api_key_key_hash", table_name="api_keys")
    op.drop_index("idx_api_key_user_id", table_name="api_keys")

    # Refresh tokens indexes
    op.drop_index("idx_refresh_token_expires", table_name="refresh_tokens")
    op.drop_index("idx_refresh_token_user_id", table_name="refresh_tokens")

    # Password reset tokens indexes
    op.drop_index("idx_password_reset_expires", table_name="password_reset_tokens")
    op.drop_index("idx_password_reset_email", table_name="password_reset_tokens")
    op.drop_index("idx_password_reset_token", table_name="password_reset_tokens")

    # Email verification tokens indexes
    op.drop_index("idx_email_verify_expires", table_name="email_verification_tokens")
    op.drop_index("idx_email_verify_email", table_name="email_verification_tokens")
    op.drop_index("idx_email_verify_token", table_name="email_verification_tokens")

    # Magic link tokens indexes
    op.drop_index("idx_magic_link_expires", table_name="magic_link_tokens")
    op.drop_index("idx_magic_link_email", table_name="magic_link_tokens")
    op.drop_index("idx_magic_link_token", table_name="magic_link_tokens")

    # OAuth accounts indexes
    op.drop_index("idx_oauth_user_id", table_name="oauth_accounts")
    op.drop_index("idx_oauth_provider_user", table_name="oauth_accounts")

    # Users table indexes
    op.drop_index("idx_users_email_verified", table_name="users")
    op.drop_index("idx_users_email", table_name="users")
