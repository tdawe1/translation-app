"""Authentication security utilities for GengoWatcher SaaS.

Implements JWT token creation/validation and Argon2id password hashing.
"""

import os
import secrets
from datetime import datetime, timedelta
from typing import Optional

from jose import JWTError, jwt
from passlib.context import CryptContext

# Configuration
SECRET_KEY = os.getenv("SECRET_KEY", secrets.token_urlsafe(32))
ALGORITHM = "HS256"
ACCESS_TOKEN_EXPIRE_MINUTES = int(
    os.getenv("ACCESS_TOKEN_EXPIRE_MINUTES", "15")
)
REFRESH_TOKEN_EXPIRE_DAYS = int(os.getenv("REFRESH_TOKEN_EXPIRE_DAYS", "7"))

# Argon2id - memory-hard password hashing
pwd_context = CryptContext(schemes=["argon2"], deprecated="auto")


def verify_password(plain_password: str, hashed_password: str) -> bool:
    """Verify a password against its hash."""
    return pwd_context.verify(plain_password, hashed_password)


def get_password_hash(password: str) -> str:
    """Hash a password using Argon2id."""
    return pwd_context.hash(password)


def create_access_token(data: dict, expires_delta: Optional[timedelta] = None) -> str:
    """Create a JWT access token.

    Args:
        data: Payload data (e.g., {"sub": user_id})
        expires_delta: Optional custom expiration

    Returns:
        Encoded JWT token string
    """
    to_encode = data.copy()
    if expires_delta:
        expire = datetime.utcnow() + expires_delta
    else:
        expire = datetime.utcnow() + timedelta(minutes=ACCESS_TOKEN_EXPIRE_MINUTES)

    to_encode.update(
        {"exp": expire, "type": "access", "iat": datetime.utcnow()}
    )
    return jwt.encode(to_encode, SECRET_KEY, algorithm=ALGORITHM)


def create_refresh_token() -> str:
    """Create a random refresh token.

    Refresh tokens are stored in the database and don't contain
    encoded claims (unlike JWT access tokens).
    """
    return secrets.token_urlsafe(32)


def decode_token(token: str) -> Optional[dict]:
    """Decode and validate a JWT token.

    Args:
        token: JWT token string

    Returns:
        Decoded payload or None if invalid
    """
    try:
        payload = jwt.decode(token, SECRET_KEY, algorithms=[ALGORITHM])
        return payload
    except JWTError:
        return None


def create_api_key() -> tuple[str, str]:
    """Create a new API key.

    Returns:
        Tuple of (raw_key, key_hash)
        - raw_key: The key to give to the user (starts with "gengo_sk_")
        - key_hash: SHA256 hash to store in database
    """
    import hashlib

    raw_key = f"gengo_sk_{secrets.token_urlsafe(32)}"
    key_hash = hashlib.sha256(raw_key.encode()).hexdigest()
    return raw_key, key_hash


def verify_api_key(raw_key: str, key_hash: str) -> bool:
    """Verify an API key against its hash.

    Args:
        raw_key: The API key provided by the user
        key_hash: The stored hash from the database

    Returns:
        True if the key matches the hash
    """
    import hashlib

    computed_hash = hashlib.sha256(raw_key.encode()).hexdigest()
    return secrets.compare_digest(computed_hash, key_hash)
