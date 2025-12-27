"""
SQLAlchemy models for GengoWatcher SaaS.

All models include user isolation via user_id foreign keys.
TimestampMixin provides created_at/updated_at for all models.
"""

import uuid
from datetime import datetime
from typing import Optional

from sqlalchemy import (
    Boolean,
    Column,
    DateTime,
    DECIMAL,
    Float,
    ForeignKey,
    Integer,
    JSON,
    String,
    Text,
)
from sqlalchemy.dialects.postgresql import UUID, TIMESTAMPTZ
from sqlalchemy.orm import relationship

from src.gengowatcher.database.base import Base


class TimestampMixin:
    """Mixin for created_at and updated_at timestamps."""

    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(
        DateTime, default=datetime.utcnow, onupdate=datetime.utcnow, nullable=False
    )


class User(Base, TimestampMixin):
    """User account with multi-method authentication support."""

    __tablename__ = "users"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    email = Column(String(255), unique=True, nullable=False, index=True)
    email_verified = Column(Boolean, default=False, nullable=False)
    password_hash = Column(String(255), nullable=True)  # Argon2id, nullable for OAuth-only users
    is_active = Column(Boolean, default=True, nullable=False)

    # Relationships
    oauth_accounts = relationship(
        "OAuthAccount", back_populates="user", cascade="all, delete-orphan"
    )
    api_keys = relationship(
        "APIKey", back_populates="user", cascade="all, delete-orphan"
    )
    refresh_tokens = relationship(
        "RefreshToken", back_populates="user", cascade="all, delete-orphan"
    )
    watcher_config = relationship(
        "UserWatcherConfig",
        back_populates="user",
        uselist=False,
        cascade="all, delete-orphan",
    )
    watcher_state = relationship(
        "UserWatcherState",
        back_populates="user",
        uselist=False,
        cascade="all, delete-orphan",
    )
    subscription = relationship(
        "Subscription",
        back_populates="user",
        uselist=False,
        cascade="all, delete-orphan",
    )


class OAuthAccount(Base, TimestampMixin):
    """OAuth provider account linking (Google, GitHub, etc.)."""

    __tablename__ = "oauth_accounts"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(
        UUID(as_uuid=True),
        ForeignKey("users.id", ondelete="CASCADE"),
        nullable=False,
        index=True,
    )
    provider = Column(String(50), nullable=False)  # 'google', 'github'
    provider_user_id = Column(String(255), nullable=False)
    access_token = Column(Text)
    refresh_token = Column(Text)
    expires_at = Column(DateTime)

    user = relationship("User", back_populates="oauth_accounts")


class APIKey(Base, TimestampMixin):
    """Programmatic API key for external integrations."""

    __tablename__ = "api_keys"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(
        UUID(as_uuid=True),
        ForeignKey("users.id", ondelete="CASCADE"),
        nullable=False,
        index=True,
    )
    key_hash = Column(String(64), unique=True, nullable=False, index=True)
    key_prefix = Column(String(10), nullable=False)  # "gengo_sk_abc"
    name = Column(String(100), nullable=False)
    scopes = Column(JSON, default=list)  # ["jobs:read", "jobs:write"]
    is_active = Column(Boolean, default=True, nullable=False)
    last_used = Column(DateTime)

    user = relationship("User", back_populates="api_keys")


class RefreshToken(Base):
    """Refresh token for long-lived sessions."""

    __tablename__ = "refresh_tokens"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(
        UUID(as_uuid=True),
        ForeignKey("users.id", ondelete="CASCADE"),
        nullable=False,
        index=True,
    )
    token = Column(String(255), unique=True, nullable=False, index=True)
    expires_at = Column(DateTime, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    revoked_at = Column(DateTime, nullable=True)

    user = relationship("User", back_populates="refresh_tokens")


class UserWatcherConfig(Base, TimestampMixin):
    """Per-user watcher configuration settings."""

    __tablename__ = "user_watcher_configs"

    user_id = Column(
        UUID(as_uuid=True),
        ForeignKey("users.id", ondelete="CASCADE"),
        primary_key=True,
    )

    # RSS/WebSocket sources
    rss_feed_url = Column(
        Text, nullable=False, default="https://gengo.com/jobs/rss"
    )
    websocket_enabled = Column(Boolean, default=True, nullable=False)
    gengo_user_id = Column(String(50))
    gengo_session_token = Column(Text)  # Encrypted at rest

    # Job filtering
    min_reward = Column(Float, default=0.0, nullable=False)
    max_reward = Column(Float, default=999999.0, nullable=False)
    included_language_pairs = Column(JSON, default=list)

    # Notifications
    enable_desktop_notifications = Column(Boolean, default=True, nullable=False)
    enable_sound_notifications = Column(Boolean, default=True, nullable=False)
    enable_email_notifications = Column(Boolean, default=False, nullable=False)
    notification_email = Column(String(255))

    # Auto-accept
    auto_accept_enabled = Column(Boolean, default=False, nullable=False)
    auto_accept_min_reward = Column(Float)
    auto_accept_max_reward = Column(Float)

    user = relationship("User", back_populates="watcher_config")


class UserWatcherState(Base):
    """Per-user watcher runtime state and statistics."""

    __tablename__ = "user_watcher_states"

    user_id = Column(
        UUID(as_uuid=True),
        ForeignKey("users.id", ondelete="CASCADE"),
        primary_key=True,
    )

    # Deduplication
    last_seen_job_ids = Column(JSON, default=list)
    last_seen_rss_link = Column(Text)

    # Statistics
    total_jobs_found = Column(Integer, default=0, nullable=False)
    total_jobs_accepted = Column(Integer, default=0, nullable=False)
    total_earnings = Column(Float, default=0.0, nullable=False)

    # Status
    watcher_status = Column(String(20), default="stopped", nullable=False)
    last_activity = Column(DateTime, default=datetime.utcnow, nullable=False)
    recent_job_history = Column(JSON, default=list)
    updated_at = Column(
        DateTime, default=datetime.utcnow, onupdate=datetime.utcnow, nullable=False
    )

    user = relationship("User", back_populates="watcher_state")


class SubscriptionPlan(Base, TimestampMixin):
    """Subscription plan definitions (Free, Pro, Enterprise)."""

    __tablename__ = "subscription_plans"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    name = Column(String(50), unique=True, nullable=False)  # 'free', 'pro', 'enterprise'
    price_cents = Column(Integer, default=0, nullable=False)
    interval = Column(String(20), default="month", nullable=False)  # 'month', 'year'
    features = Column(JSON, default={})
    stripe_price_id = Column(String(100))
    is_active = Column(Boolean, default=True, nullable=False)


class Subscription(Base, TimestampMixin):
    """User subscription with Stripe integration."""

    __tablename__ = "subscriptions"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(
        UUID(as_uuid=True),
        ForeignKey("users.id", ondelete="CASCADE"),
        nullable=False,
        index=True,
    )
    plan_id = Column(
        UUID(as_uuid=True), ForeignKey("subscription_plans.id"), nullable=True
    )
    stripe_customer_id = Column(String(100))
    stripe_subscription_id = Column(String(100), unique=True)
    stripe_subscription_status = Column(String(50))  # 'active', 'past_due', 'canceled'
    current_period_start = Column(TIMESTAMPTZ)
    current_period_end = Column(TIMESTAMPTZ)
    cancel_at_period_end = Column(Boolean, default=False, nullable=False)

    user = relationship("User", back_populates="subscription")


class BillingEvent(Base):
    """Stripe webhook event log for idempotency."""

    __tablename__ = "billing_events"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(
        UUID(as_uuid=True), ForeignKey("users.id", ondelete="SET NULL"), nullable=True
    )
    stripe_event_id = Column(String(100), unique=True, nullable=False)
    event_type = Column(String(50), nullable=False)
    event_data = Column(JSON)
    processed_at = Column(TIMESTAMPTZ, default=datetime.utcnow, nullable=False)


class AuditLog(Base):
    """Security and compliance audit trail."""

    __tablename__ = "audit_log"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(
        UUID(as_uuid=True), ForeignKey("users.id", ondelete="SET NULL"), nullable=True
    )
    event_type = Column(String(50), nullable=False)  # 'login', 'api_key_created', etc.
    event_data = Column(JSON)
    ip_address = Column(String(45))
    user_agent = Column(Text)
    created_at = Column(TIMESTAMPTZ, default=datetime.utcnow, nullable=False)
