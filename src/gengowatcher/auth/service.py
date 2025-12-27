"""Authentication service for GengoWatcher SaaS.

Handles user registration, login, token refresh, and user management.
"""

from datetime import datetime, timedelta
from typing import Optional
from uuid import UUID

from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from src.gengowatcher.auth.exceptions import (
    InactiveUserException,
    InvalidCredentialsException,
    InvalidTokenException,
    UserExistsException,
    UserNotFoundException,
)
from src.gengowatcher.auth.security import (
    create_access_token,
    create_refresh_token,
    get_password_hash,
    verify_password,
)
from src.gengowatcher.database.models import (
    RefreshToken,
    User,
    UserWatcherConfig,
    UserWatcherState,
)


class AuthService:
    """Service for authentication operations."""

    def __init__(self, db: AsyncSession):
        self.db = db

    async def register_user(
        self,
        email: str,
        password: str,
        email_verified: bool = False,
    ) -> User:
        """Register a new user.

        Args:
            email: User email address
            password: Plain text password (will be hashed)
            email_verified: Whether email is pre-verified

        Returns:
            Created User object

        Raises:
            UserExistsException: If email already registered
        """
        # Check if user exists
        result = await self.db.execute(select(User).where(User.email == email))
        if result.scalar_one_or_none():
            raise UserExistsException(email)

        # Create user
        user = User(
            email=email,
            password_hash=get_password_hash(password),
            email_verified=email_verified,
            is_active=True,
        )
        self.db.add(user)

        # Flush to get user ID for dependent objects
        await self.db.flush()

        # Create default watcher config
        config = UserWatcherConfig(user=user)
        self.db.add(config)

        # Create default watcher state
        state = UserWatcherState(user=user)
        self.db.add(state)

        await self.db.commit()
        await self.db.refresh(user)
        return user

    async def authenticate_user(
        self, email: str, password: str
    ) -> tuple[User, str, str]:
        """Authenticate a user with email/password.

        Args:
            email: User email address
            password: Plain text password

        Returns:
            Tuple of (User, access_token, refresh_token)

        Raises:
            InvalidCredentialsException: If email/password invalid
            InactiveUserException: If user account is inactive
        """
        result = await self.db.execute(select(User).where(User.email == email))
        user = result.scalar_one_or_none()

        if not user or not verify_password(password, user.password_hash or ""):
            raise InvalidCredentialsException()

        if not user.is_active:
            raise InactiveUserException()

        # Create tokens
        access_token = create_access_token({"sub": str(user.id)})
        refresh_token = create_refresh_token()

        # Store refresh token in database
        rt = RefreshToken(
            user_id=user.id,
            token=refresh_token,
            expires_at=datetime.utcnow() + timedelta(days=7),
        )
        self.db.add(rt)
        await self.db.commit()

        return user, access_token, refresh_token

    async def refresh_tokens(
        self, refresh_token: str
    ) -> Optional[tuple[str, str, User]]:
        """Refresh access token using refresh token.

        Args:
            refresh_token: The refresh token from cookie/storage

        Returns:
            Tuple of (new_access_token, new_refresh_token, user) or None

        Raises:
            InvalidTokenException: If refresh token is invalid
        """
        result = await self.db.execute(
            select(RefreshToken, User)
            .join(User, RefreshToken.user_id == User.id)
            .where(RefreshToken.token == refresh_token)
            .where(RefreshToken.revoked_at == None)
        )
        row = result.first()

        if not row:
            raise InvalidTokenException("refresh token")

        rt, user = row

        # Check if expired
        if rt.expires_at < datetime.utcnow():
            # Revoke the expired token
            rt.revoked_at = datetime.utcnow()
            await self.db.commit()
            raise InvalidTokenException("refresh token (expired)")

        # Revoke old token (rotation)
        rt.revoked_at = datetime.utcnow()

        # Create new tokens
        new_access = create_access_token({"sub": str(user.id)})
        new_refresh = create_refresh_token()

        # Store new refresh token
        new_rt = RefreshToken(
            user_id=user.id,
            token=new_refresh,
            expires_at=datetime.utcnow() + timedelta(days=7),
        )
        self.db.add(new_rt)

        await self.db.commit()
        return new_access, new_refresh, user

    async def revoke_refresh_token(self, refresh_token: str) -> bool:
        """Revoke a refresh token (logout).

        Args:
            refresh_token: The refresh token to revoke

        Returns:
            True if token was found and revoked
        """
        result = await self.db.execute(
            select(RefreshToken).where(
                RefreshToken.token == refresh_token,
                RefreshToken.revoked_at == None,
            )
        )
        rt = result.scalar_one_or_none()

        if rt:
            rt.revoked_at = datetime.utcnow()
            await self.db.commit()
            return True
        return False

    async def revoke_all_user_tokens(self, user_id: UUID) -> int:
        """Revoke all refresh tokens for a user.

        Args:
            user_id: The user ID

        Returns:
            Number of tokens revoked
        """
        result = await self.db.execute(
            select(RefreshToken).where(
                RefreshToken.user_id == user_id,
                RefreshToken.revoked_at == None,
            )
        )
        tokens = result.scalars().all()

        count = 0
        for rt in tokens:
            rt.revoked_at = datetime.utcnow()
            count += 1

        await self.db.commit()
        return count

    async def get_user_by_id(self, user_id: UUID) -> Optional[User]:
        """Get user by ID.

        Args:
            user_id: The user UUID

        Returns:
            User object or None
        """
        result = await self.db.execute(select(User).where(User.id == user_id))
        return result.scalar_one_or_none()

    async def get_user_by_email(self, email: str) -> Optional[User]:
        """Get user by email.

        Args:
            email: The user email

        Returns:
            User object or None
        """
        result = await self.db.execute(select(User).where(User.email == email))
        return result.scalar_one_or_none()

    async def verify_email(self, user_id: UUID) -> User:
        """Mark user email as verified.

        Args:
            user_id: The user ID

        Returns:
            Updated User object

        Raises:
            UserNotFoundException: If user not found
        """
        user = await self.get_user_by_id(user_id)
        if not user:
            raise UserNotFoundException(str(user_id))

        user.email_verified = True
        await self.db.commit()
        await self.db.refresh(user)
        return user

    async def change_password(
        self, user_id: UUID, old_password: str, new_password: str
    ) -> bool:
        """Change user password.

        Args:
            user_id: The user ID
            old_password: Current password for verification
            new_password: New password to set

        Returns:
            True if password changed

        Raises:
            InvalidCredentialsException: If old password is incorrect
        """
        user = await self.get_user_by_id(user_id)
        if not user:
            return False

        if not verify_password(old_password, user.password_hash or ""):
            raise InvalidCredentialsException()

        user.password_hash = get_password_hash(new_password)
        await self.db.commit()

        # Revoke all refresh tokens for security
        await self.revoke_all_user_tokens(user_id)
        return True
