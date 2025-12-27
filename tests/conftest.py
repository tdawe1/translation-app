"""Pytest configuration and fixtures for GengoWatcher SaaS tests."""

import asyncio
import os
from pathlib import Path
from typing import AsyncGenerator, Generator

import pytest
import redis.asyncio as redis
from sqlalchemy.ext.asyncio import AsyncSession, async_sessionmaker, create_async_engine
from sqlalchemy.pool import StaticPool

# Add src to path
src_path = Path(__file__).parent.parent / "src"
import sys

sys.path.insert(0, str(src_path))

from src.gengowatcher.database.models import Base

TEST_DATABASE_URL = os.getenv(
    "TEST_DATABASE_URL", "sqlite+aiosqlite:///./test.db"
)


@pytest.fixture(scope="session")
def event_loop():
    """Create an instance of the event loop for the test session."""
    loop = asyncio.new_event_loop()
    yield loop
    loop.close()


@pytest.fixture(scope="session")
async def engine():
    """Create a test database engine."""
    engine = create_async_engine(
        TEST_DATABASE_URL,
        connect_args={"check_same_thread": False} if "sqlite" in TEST_DATABASE_URL else {},
        poolclass=StaticPool if "sqlite" in TEST_DATABASE_URL else None,
        echo=False,
    )
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield engine
    await engine.dispose()


@pytest.fixture
async def db_session(engine) -> AsyncGenerator[AsyncSession, None]:
    """Create a test database session."""
    async_session = async_sessionmaker(
        engine, class_=AsyncSession, expire_on_commit=False
    )
    async with async_session() as session:
        yield session
    await session.rollback()


@pytest.fixture
async def redis_client() -> AsyncGenerator[redis.Redis, None]:
    """Create a test Redis client."""
    client = redis.from_url(
        os.getenv("TEST_REDIS_URL", "redis://localhost:6379/1"),
        decode_responses=True,
    )
    yield client
    await client.flushdb()
    await client.close()


@pytest.fixture
def mock_user():
    """Create a mock user object."""
    return {
        "id": "00000000-0000-0000-0000-000000000001",
        "email": "test@example.com",
        "email_verified": True,
        "is_active": True,
    }


@pytest.fixture
def mock_access_token(mock_user):
    """Create a mock access token."""
    return {
        "sub": mock_user["id"],
        "exp": 9999999999,
        "type": "access",
    }
