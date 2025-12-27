"""Tests for auth security module."""

import pytest

from src.gengowatcher.auth.security import (
    create_access_token,
    create_refresh_token,
    decode_token,
    get_password_hash,
    verify_password,
)


def test_password_hashing():
    """Test password hashing and verification."""
    password = "test_password_123"
    hash_result = get_password_hash(password)

    # Hash should be different from password
    assert hash_result != password
    assert len(hash_result) > 50

    # Verification should work
    assert verify_password(password, hash_result) is True

    # Wrong password should fail
    assert verify_password("wrong_password", hash_result) is False


def test_access_token_creation():
    """Test JWT access token creation and decoding."""
    user_id = "00000000-0000-0000-0000-000000000001"
    token = create_access_token({"sub": user_id})

    # Token should be a non-empty string
    assert isinstance(token, str)
    assert len(token) > 0

    # Decode and verify payload
    payload = decode_token(token)
    assert payload is not None
    assert payload["sub"] == user_id
    assert payload["type"] == "access"


def test_access_token_expiry():
    """Test that access tokens include expiry."""
    from datetime import timedelta

    user_id = "00000000-0000-0000-0000-000000000001"
    token = create_access_token({"sub": user_id}, expires_delta=timedelta(minutes=30))

    payload = decode_token(token)
    assert payload is not None
    assert "exp" in payload


def test_invalid_token():
    """Test decoding invalid tokens."""
    assert decode_token("invalid_token") is None
    assert decode_token("eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.invalid") is None


def test_refresh_token_generation():
    """Test refresh token generation."""
    token1 = create_refresh_token()
    token2 = create_refresh_token()

    # Tokens should be random and different
    assert isinstance(token1, str)
    assert isinstance(token2, str)
    assert len(token1) > 30
    assert token1 != token2


def test_api_key_generation():
    """Test API key generation."""
    raw_key, key_hash = create_api_key()

    # Raw key should have prefix
    assert raw_key.startswith("gengo_sk_")
    assert len(raw_key) > 40

    # Hash should be different from raw key
    assert key_hash != raw_key
    assert len(key_hash) == 64  # SHA256 hex length


def test_api_key_verification():
    """Test API key verification."""
    raw_key, key_hash = create_api_key()

    # Correct key should verify
    assert verify_api_key(raw_key, key_hash) is True

    # Wrong key should not verify
    assert verify_api_key("wrong_key", key_hash) is False
