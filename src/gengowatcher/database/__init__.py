"""Database module for GengoWatcher SaaS."""

from src.gengowatcher.database.base import Base
from src.gengowatcher.database.models import (
    APIKey,
    AuditLog,
    BillingEvent,
    OAuthAccount,
    RefreshToken,
    Subscription,
    SubscriptionPlan,
    User,
    UserWatcherConfig,
    UserWatcherState,
)
from src.gengowatcher.database.session import (
    AsyncSessionLocal,
    close_db,
    engine,
    get_db,
    get_db_session,
    init_db,
)

__all__ = [
    # Base
    "Base",
    # Models
    "User",
    "OAuthAccount",
    "APIKey",
    "RefreshToken",
    "UserWatcherConfig",
    "UserWatcherState",
    "SubscriptionPlan",
    "Subscription",
    "BillingEvent",
    "AuditLog",
    # Session
    "engine",
    "AsyncSessionLocal",
    "get_db",
    "get_db_session",
    "init_db",
    "close_db",
]
