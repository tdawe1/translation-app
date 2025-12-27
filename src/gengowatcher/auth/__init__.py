"""Authentication module for GengoWatcher SaaS."""

from src.gengowatcher.auth.exceptions import (
    AuthException,
    EmailNotVerifiedException,
    InactiveUserException,
    InvalidCredentialsException,
    InvalidTokenException,
    OAuthException,
    RateLimitException,
    TokenExpiredException,
    UserExistsException,
    UserNotFoundException,
)
from src.gengowatcher.auth.routes import router
from src.gengowatcher.auth.security import (
    create_access_token,
    create_refresh_token,
    get_password_hash,
    verify_password,
)
from src.gengowatcher.auth.service import AuthService

__all__ = [
    # Service
    "AuthService",
    # Security
    "create_access_token",
    "create_refresh_token",
    "get_password_hash",
    "verify_password",
    # Exceptions
    "AuthException",
    "UserNotFoundException",
    "UserExistsException",
    "InvalidCredentialsException",
    "InvalidTokenException",
    "TokenExpiredException",
    "InactiveUserException",
    "EmailNotVerifiedException",
    "OAuthException",
    "RateLimitException",
    # Routes
    "router",
]
