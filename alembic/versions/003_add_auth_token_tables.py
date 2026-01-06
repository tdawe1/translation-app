"""Add authentication token tables

Revision ID: 003_add_auth_token_tables
Revises: 002_add_auth_indexes
Create Date: 2026-01-05

"""

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql


# revision identifiers, used by Alembic.
revision = "003_add_auth_token_tables"
down_revision = "001_initial_schema"
branch_labels = None
depends_on = None


def upgrade() -> None:
    # Create magic_link_tokens table
    op.create_table(
        "magic_link_tokens",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            primary_key=True,
            server_default=sa.text("gen_random_uuid()"),
        ),
        sa.Column("email", sa.String(255), nullable=False),
        sa.Column("token", sa.String(255), nullable=False, unique=True),
        sa.Column("expires_at", sa.DateTime(), nullable=False),
        sa.Column("used_at", sa.DateTime(), nullable=True),
        sa.Column(
            "created_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")
        ),
    )
    op.create_index("idx_magic_link_token", "magic_link_tokens", ["token"], unique=True)
    op.create_index("idx_magic_link_email", "magic_link_tokens", ["email"])
    op.create_index("idx_magic_link_expires", "magic_link_tokens", ["expires_at"])

    # Create email_verification_tokens table
    op.create_table(
        "email_verification_tokens",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            primary_key=True,
            server_default=sa.text("gen_random_uuid()"),
        ),
        sa.Column("email", sa.String(255), nullable=False),
        sa.Column("token", sa.String(255), nullable=False, unique=True),
        sa.Column("expires_at", sa.DateTime(), nullable=False),
        sa.Column("used_at", sa.DateTime(), nullable=True),
        sa.Column(
            "created_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")
        ),
    )
    op.create_index(
        "idx_email_verify_token", "email_verification_tokens", ["token"], unique=True
    )
    op.create_index("idx_email_verify_email", "email_verification_tokens", ["email"])
    op.create_index(
        "idx_email_verify_expires", "email_verification_tokens", ["expires_at"]
    )

    # Create password_reset_tokens table
    op.create_table(
        "password_reset_tokens",
        sa.Column(
            "id",
            postgresql.UUID(as_uuid=True),
            primary_key=True,
            server_default=sa.text("gen_random_uuid()"),
        ),
        sa.Column("email", sa.String(255), nullable=False),
        sa.Column("token", sa.String(255), nullable=False, unique=True),
        sa.Column("expires_at", sa.DateTime(), nullable=False),
        sa.Column("used_at", sa.DateTime(), nullable=True),
        sa.Column(
            "created_at", sa.DateTime(), nullable=False, server_default=sa.text("now()")
        ),
    )
    op.create_index(
        "idx_password_reset_token", "password_reset_tokens", ["token"], unique=True
    )
    op.create_index("idx_password_reset_email", "password_reset_tokens", ["email"])
    op.create_index(
        "idx_password_reset_expires", "password_reset_tokens", ["expires_at"]
    )


def downgrade() -> None:
    op.drop_table("password_reset_tokens")
    op.drop_table("email_verification_tokens")
    op.drop_table("magic_link_tokens")
