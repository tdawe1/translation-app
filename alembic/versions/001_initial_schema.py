"""Initial schema

Revision ID: 001
Revises:
Create Date: 2024-01-01 00:00:00.000000

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

# revision identifiers, used by Alembic.
revision = "001"
down_revision = None
branch_labels = None
depends_on = None


def upgrade() -> None:
    # Create users table
    op.create_table(
        "users",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            primary_key=True,
            server_default=sa.text("gen_random_uuid()"),
        ),
        sa.Column("email", sa.String(255), nullable=False),
        sa.Column("email_verified", sa.Boolean(), nullable=False, server_default="false"),
        sa.Column("password_hash", sa.String(255), nullable=True),
        sa.Column("is_active", sa.Boolean(), nullable=False, server_default="true"),
        sa.Column("created_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
        sa.Column("updated_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
    )
    op.create_index("ix_users_email", "users", ["email"], unique=True)

    # Create oauth_accounts table
    op.create_table(
        "oauth_accounts",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            primary_key=True,
            server_default=sa.text("gen_random_uuid()"),
        ),
        sa.Column(
            "user_id",
            postgresql.UUID(as_uuid=True),
            sa.ForeignKey("users.id", ondelete="CASCADE"),
            nullable=False,
        ),
        sa.Column("provider", sa.String(50), nullable=False),
        sa.Column("provider_user_id", sa.String(255), nullable=False),
        sa.Column("access_token", sa.Text(), nullable=True),
        sa.Column("refresh_token", sa.Text(), nullable=True),
        sa.Column("expires_at", sa.DateTime(), nullable=True),
        sa.Column("created_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
        sa.Column("updated_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
    )
    op.create_index("ix_oauth_accounts_user_id", "oauth_accounts", ["user_id"])

    # Create api_keys table
    op.create_table(
        "api_keys",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            primary_key=True,
            server_default=sa.text("gen_random_uuid()"),
        ),
        sa.Column(
            "user_id",
            postgresql.UUID(as_uuid=True),
            sa.ForeignKey("users.id", ondelete="CASCADE"),
            nullable=False,
        ),
        sa.Column("key_hash", sa.String(64), nullable=False),
        sa.Column("key_prefix", sa.String(10), nullable=False),
        sa.Column("name", sa.String(100), nullable=False),
        sa.Column("scopes", sa.JSON(), nullable=True, server_default="[]"),
        sa.Column("is_active", sa.Boolean(), nullable=False, server_default="true"),
        sa.Column("last_used", sa.DateTime(), nullable=True),
        sa.Column("created_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
        sa.Column("updated_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
    )
    op.create_index("ix_api_keys_key_hash", "api_keys", ["key_hash"], unique=True)
    op.create_index("ix_api_keys_user_id", "api_keys", ["user_id"])

    # Create refresh_tokens table
    op.create_table(
        "refresh_tokens",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            primary_key=True,
            server_default=sa.text("gen_random_uuid()"),
        ),
        sa.Column(
            "user_id",
            postgresql.UUID(as_uuid=True),
            sa.ForeignKey("users.id", ondelete="CASCADE"),
            nullable=False,
        ),
        sa.Column("token", sa.String(255), nullable=False),
        sa.Column("expires_at", sa.DateTime(), nullable=False),
        sa.Column("created_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
        sa.Column("revoked_at", sa.DateTime(), nullable=True),
    )
    op.create_index("ix_refresh_tokens_token", "refresh_tokens", ["token"], unique=True)
    op.create_index("ix_refresh_tokens_user_id", "refresh_tokens", ["user_id"])

    # Create user_watcher_configs table
    op.create_table(
        "user_watcher_configs",
        sa.Column(
            "user_id",
            postgresql.UUID(as_uuid=True),
            sa.ForeignKey("users.id", ondelete="CASCADE"),
            primary_key=True,
        ),
        sa.Column("rss_feed_url", sa.Text(), nullable=False, server_default="https://gengo.com/jobs/rss"),
        sa.Column("websocket_enabled", sa.Boolean(), nullable=False, server_default="true"),
        sa.Column("gengo_user_id", sa.String(50), nullable=True),
        sa.Column("gengo_session_token", sa.Text(), nullable=True),
        sa.Column("min_reward", sa.Float(), nullable=False, server_default="0.0"),
        sa.Column("max_reward", sa.Float(), nullable=False, server_default="999999.0"),
        sa.Column("included_language_pairs", sa.JSON(), nullable=True, server_default="[]"),
        sa.Column("enable_desktop_notifications", sa.Boolean(), nullable=False, server_default="true"),
        sa.Column("enable_sound_notifications", sa.Boolean(), nullable=False, server_default="true"),
        sa.Column("enable_email_notifications", sa.Boolean(), nullable=False, server_default="false"),
        sa.Column("notification_email", sa.String(255), nullable=True),
        sa.Column("auto_accept_enabled", sa.Boolean(), nullable=False, server_default="false"),
        sa.Column("auto_accept_min_reward", sa.Float(), nullable=True),
        sa.Column("auto_accept_max_reward", sa.Float(), nullable=True),
        sa.Column("created_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
        sa.Column("updated_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
    )

    # Create user_watcher_states table
    op.create_table(
        "user_watcher_states",
        sa.Column(
            "user_id",
            postgresql.UUID(as_uuid=True),
            sa.ForeignKey("users.id", ondelete="CASCADE"),
            primary_key=True,
        ),
        sa.Column("last_seen_job_ids", sa.JSON(), nullable=True, server_default="[]"),
        sa.Column("last_seen_rss_link", sa.Text(), nullable=True),
        sa.Column("total_jobs_found", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("total_jobs_accepted", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("total_earnings", sa.Float(), nullable=False, server_default="0.0"),
        sa.Column("watcher_status", sa.String(20), nullable=False, server_default="stopped"),
        sa.Column("last_activity", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
        sa.Column("recent_job_history", sa.JSON(), nullable=True, server_default="[]"),
        sa.Column("updated_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
    )

    # Create subscription_plans table
    op.create_table(
        "subscription_plans",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            primary_key=True,
            server_default=sa.text("gen_random_uuid()"),
        ),
        sa.Column("name", sa.String(50), nullable=False, unique=True),
        sa.Column("price_cents", sa.Integer(), nullable=False, server_default="0"),
        sa.Column("interval", sa.String(20), nullable=False, server_default="month"),
        sa.Column("features", sa.JSON(), nullable=True, server_default="{}"),
        sa.Column("stripe_price_id", sa.String(100), nullable=True),
        sa.Column("is_active", sa.Boolean(), nullable=False, server_default="true"),
        sa.Column("created_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
        sa.Column("updated_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
    )

    # Create subscriptions table
    op.create_table(
        "subscriptions",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            primary_key=True,
            server_default=sa.text("gen_random_uuid()"),
        ),
        sa.Column(
            "user_id",
            postgresql.UUID(as_uuid=True),
            sa.ForeignKey("users.id", ondelete="CASCADE"),
            nullable=False,
        ),
        sa.Column(
            "plan_id",
            postgresql.UUID(as_uuid=True),
            sa.ForeignKey("subscription_plans.id"),
            nullable=True,
        ),
        sa.Column("stripe_customer_id", sa.String(100), nullable=True),
        sa.Column("stripe_subscription_id", sa.String(100), unique=True, nullable=True),
        sa.Column("stripe_subscription_status", sa.String(50), nullable=True),
        sa.Column("current_period_start", sa.TIMESTAMP(), nullable=True),
        sa.Column("current_period_end", sa.TIMESTAMP(), nullable=True),
        sa.Column("cancel_at_period_end", sa.Boolean(), nullable=False, server_default="false"),
        sa.Column("created_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
        sa.Column("updated_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")),
    )
    op.create_index("ix_subscriptions_user_id", "subscriptions", ["user_id"])

    # Create billing_events table
    op.create_table(
        "billing_events",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            primary_key=True,
            server_default=sa.text("gen_random_uuid()"),
        ),
        sa.Column(
            "user_id",
            postgresql.UUID(as_uuid=True),
            sa.ForeignKey("users.id", ondelete="SET NULL"),
            nullable=True,
        ),
        sa.Column("stripe_event_id", sa.String(100), unique=True, nullable=False),
        sa.Column("event_type", sa.String(50), nullable=False),
        sa.Column("event_data", sa.JSON(), nullable=True),
        sa.Column("processed_at", sa.TIMESTAMP(), nullable=False, server_default=sa.text("now()")),
    )

    # Create audit_log table
    op.create_table(
        "audit_log",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            primary_key=True,
            server_default=sa.text("gen_random_uuid()"),
        ),
        sa.Column(
            "user_id",
            postgresql.UUID(as_uuid=True),
            sa.ForeignKey("users.id", ondelete="SET NULL"),
            nullable=True,
        ),
        sa.Column("event_type", sa.String(50), nullable=False),
        sa.Column("event_data", sa.JSON(), nullable=True),
        sa.Column("ip_address", sa.String(45), nullable=True),
        sa.Column("user_agent", sa.Text(), nullable=True),
        sa.Column("created_at", sa.TIMESTAMP(), nullable=False, server_default=sa.text("now()")),
    )


def downgrade() -> None:
    op.drop_table("audit_log")
    op.drop_table("billing_events")
    op.drop_table("subscriptions")
    op.drop_table("subscription_plans")
    op.drop_table("user_watcher_states")
    op.drop_table("user_watcher_configs")
    op.drop_table("refresh_tokens")
    op.drop_table("api_keys")
    op.drop_table("oauth_accounts")
    op.drop_table("users")
