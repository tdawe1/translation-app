"""Authentication exceptions for GengoWatcher SaaS.

All exceptions follow the standard error format:
{"error": str, "code": str, "details": dict}
"""


class AuthException(Exception):
    """Base authentication exception."""

    def __init__(
        self,
        message: str,
        code: str,
        details: Optional[dict] = None,
        status_code: int = 400,
    ):
        self.message = message
        self.code = code
        self.details = details or {}
        self.status_code = status_code
        super().__init__(message)

    def to_dict(self) -> dict:
        """Convert exception to standard error format."""
        result = {"error": self.message, "code": self.code}
        if self.details:
            result["details"] = self.details
        return result


class UserNotFoundException(AuthException):
    """Raised when a user is not found."""

    def __init__(self, identifier: str):
        super().__init__(
            message=f"User not found",
            code="USER_NOT_FOUND",
            details={"identifier": identifier},
            status_code=404,
        )


class UserExistsException(AuthException):
    """Raised when attempting to register an existing user."""

    def __init__(self, email: str):
        super().__init__(
            message="A user with this email already exists",
            code="USER_EXISTS",
            details={"email": email},
            status_code=409,
        )


class InvalidCredentialsException(AuthException):
    """Raised when email/password combination is invalid."""

    def __init__(self):
        super().__init__(
            message="Invalid email or password",
            code="INVALID_CREDENTIALS",
            status_code=401,
        )


class InvalidTokenException(AuthException):
    """Raised when a token is invalid or expired."""

    def __init__(self, token_type: str = "token"):
        super().__init__(
            message=f"Invalid or expired {token_type}",
            code="INVALID_TOKEN",
            status_code=401,
        )


class TokenExpiredException(AuthException):
    """Raised when a token has expired."""

    def __init__(self):
        super().__init__(
            message="Token has expired",
            code="TOKEN_EXPIRED",
            status_code=401,
        )


class InactiveUserException(AuthException):
    """Raised when attempting to authenticate an inactive user."""

    def __init__(self):
        super().__init__(
            message="User account is inactive",
            code="INACTIVE_USER",
            status_code=403,
        )


class EmailNotVerifiedException(AuthException):
    """Raised when attempting to authenticate without email verification."""

    def __init__(self):
        super().__init__(
            message="Email address not verified",
            code="EMAIL_NOT_VERIFIED",
            status_code=403,
        )


class OAuthException(AuthException):
    """Raised when OAuth flow fails."""

    def __init__(self, provider: str, reason: str = "Authentication failed"):
        super().__init__(
            message=f"{provider} authentication failed: {reason}",
            code="OAUTH_ERROR",
            details={"provider": provider},
            status_code=400,
        )


class RateLimitException(AuthException):
    """Raised when rate limit is exceeded."""

    def __init__(self, retry_after: Optional[int] = None):
        super().__init__(
            message="Too many requests. Please try again later.",
            code="RATE_LIMIT_EXCEEDED",
            details={"retry_after": retry_after} if retry_after else {},
            status_code=429,
        )
