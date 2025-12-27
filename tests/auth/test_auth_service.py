"""Tests for auth service."""

import pytest

from src.gengowatcher.auth.exceptions import (
    InvalidCredentialsException,
    UserExistsException,
)
from src.gengowatcher.auth.service import AuthService
from src.gengowatcher.database.models import User


@pytest.mark.asyncio
async def test_register_user(db_session):
    """Test user registration."""
    service = AuthService(db_session)
    user = await service.register_user("test@example.com", "password123")

    assert user.email == "test@example.com"
    assert user.password_hash is not None
    assert user.password_hash != "password123"
    assert user.is_active is True
    assert user.email_verified is False
    assert user.id is not None


@pytest.mark.asyncio
async def test_register_duplicate_email(db_session):
    """Test registering with duplicate email raises exception."""
    service = AuthService(db_session)
    await service.register_user("test@example.com", "password123")

    with pytest.raises(UserExistsException):
        await service.register_user("test@example.com", "different_password")


@pytest.mark.asyncio
async def test_authenticate_user(db_session):
    """Test user authentication."""
    service = AuthService(db_session)
    await service.register_user("test@example.com", "password123")

    user, access, refresh = await service.authenticate_user("test@example.com", "password123")

    assert user.email == "test@example.com"
    assert access is not None
    assert refresh is not None


@pytest.mark.asyncio
async def test_authenticate_wrong_password(db_session):
    """Test authentication with wrong password raises exception."""
    service = AuthService(db_session)
    await service.register_user("test@example.com", "password123")

    with pytest.raises(InvalidCredentialsException):
        await service.authenticate_user("test@example.com", "wrong_password")


@pytest.mark.asyncio
async def test_authenticate_nonexistent_user(db_session):
    """Test authentication with non-existent user raises exception."""
    service = AuthService(db_session)

    with pytest.raises(InvalidCredentialsException):
        await service.authenticate_user("nonexistent@example.com", "password")


@pytest.mark.asyncio
async def test_refresh_tokens(db_session):
    """Test token refresh."""
    service = AuthService(db_session)
    await service.register_user("test@example.com", "password123")
    _, _, refresh_token = await service.authenticate_user("test@example.com", "password123")

    # Refresh tokens
    new_access, new_refresh, user = await service.refresh_tokens(refresh_token)

    assert new_access is not None
    assert new_refresh is not None
    assert user.email == "test@example.com"
    # New refresh token should be different (rotation)
    assert new_refresh != refresh_token


@pytest.mark.asyncio
async def test_get_user_by_id(db_session):
    """Test getting user by ID."""
    service = AuthService(db_session)
    user = await service.register_user("test@example.com", "password123")

    found_user = await service.get_user_by_id(user.id)

    assert found_user is not None
    assert found_user.email == "test@example.com"
    assert found_user.id == user.id


@pytest.mark.asyncio
async def test_verify_email(db_session):
    """Test email verification."""
    service = AuthService(db_session)
    user = await service.register_user("test@example.com", "password123")

    assert user.email_verified is False

    verified_user = await service.verify_email(user.id)

    assert verified_user.email_verified is True


@pytest.mark.asyncio
async def test_change_password(db_session):
    """Test password change."""
    service = AuthService(db_session)
    user = await service.register_user("test@example.com", "password123")

    # Change password
    result = await service.change_password(user.id, "password123", "new_password456")
    assert result is True

    # Old password should not work
    with pytest.raises(InvalidCredentialsException):
        await service.authenticate_user("test@example.com", "password123")

    # New password should work
    user, _, _ = await service.authenticate_user("test@example.com", "new_password456")
    assert user.email == "test@example.com"
