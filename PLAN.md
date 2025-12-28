# GengoWatcher Remote SaaS - Agent Implementation Plan

## Approach: Clean Implementation

This is a **new SaaS project**. Reference the existing GengoWatcher's RSS/WebSocket logic for domain knowledge, but implement fresh with user isolation from the start.

**No legacy code integration required.**

---

## Technical Decisions

### Email Provider
**Choice**: Resend (https://resend.com)
- Simple API
- Free tier (3000 emails/day)
- Good Python SDK

### CAPTCHA Solution
Browser-based agent (Playwright) using user's browser session. No API keys needed.

### Standard Error Format
```python
{
    "error": "Human readable message",
    "code": "USER_NOT_FOUND",
    "details": {"field": "email"},
    "request_id": "uuid"
}
```

---

## Sprint 0: Project Scaffolding

**Goal**: Set up development environment, dependencies, and tooling

### Task SPRINT0-001: Update Requirements

**Files to modify**:
```bash
# requirements.txt - Add these dependencies
sqlalchemy[asyncio]>=2.0.0
aiosqlite>=0.19.0
asyncpg>=0.29.0
alembic>=1.12.0
argon2-cffi>=23.0.0
python-jose[cryptography]>=3.3.0
passlib>=1.7.4
authlib>=1.2.1
resend>=0.8.0
stripe>=7.0.0
redis[hiredis]>=4.6.0
```

**Verification**:
```bash
pip install -r requirements.txt
python -c "import sqlalchemy; import argon2; import jose; print('OK')"
```

---

### Task SPRINT0-002: Create Directory Structure

**Files to create**:
```bash
# Backend directories
mkdir -p src/gengowatcher/database
mkdir -p src/gengowatcher/auth
mkdir -p src/gengowatcher/watcher
mkdir -p src/gengowatcher/billing
mkdir -p src/gengowatcher/email
mkdir -p tests/auth
mkdir -p tests/watcher
mkdir -p tests/database

# Frontend directories (if not exists)
mkdir -p frontend/src/contexts
mkdir -p frontend/src/routes
mkdir -p frontend/src/pages/auth
mkdir -p frontend/src/components/auth
```

**Verification**:
```bash
ls -la src/gengowatcher/
ls -la frontend/src/
```

---

### Task SPRINT0-003: Configure Alembic

**Files to create**:
```bash
cd /home/thomas/GengoWatcher
alembic init alembic
```

**Files to modify**:
```python
# alembic/env.py - Modify target_metadata
from src.gengowatcher.database.models import Base
target_metadata = Base.metadata
```

**Verification**:
```bash
alembic current
```

---

### Task SPRINT0-004: Configure pytest

**Files to create**:
```python
# pytest.ini
[pytest]
testpaths = tests
asyncio_mode = auto
python_files = test_*.py
python_classes = Test*
python_functions = test_*
addopts =
    -v
    --tb=short
    --cov=src/gengowatcher
    --cov-report=html

# tests/conftest.py
import asyncio
import pytest
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from sqlalchemy.orm import sessionmaker
from src.gengowatcher.database.models import Base

TEST_DATABASE_URL = "sqlite+aiosqlite:///./test.db"

@pytest.fixture(scope="session")
def event_loop():
    loop = asyncio.get_event_loop_policy().new_event_loop()
    yield loop
    loop.close()

@pytest.fixture(scope="session")
async def engine():
    engine = create_async_engine(TEST_DATABASE_URL, echo=False)
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    yield engine
    await engine.dispose()

@pytest.fixture
async def db(engine) -> AsyncSession:
    async_session = sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)
    async with async_session() as session:
        yield session
    await session.rollback()

@pytest.fixture
def redis():
    import redis.asyncio as redis
    client = redis.from_url("redis://localhost:6379/1", decode_responses=True)
    yield client
    asyncio.run(client.close())
```

**Verification**:
```bash
pytest tests/ -v  # Should run (even with 0 tests)
```

---

### Task SPRINT0-005: Docker Compose for Local Dev

**Files to create**:
```yaml
# docker-compose.yml
version: "3.8"

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: gengowatcher
      POSTGRES_USER: gengo
      POSTGRES_PASSWORD: devpass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U gengo"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  mailhog:
    image: mailhog/mailhog
    ports:
      - "1025:1025"  # SMTP
      - "8025:8025"  # Web UI

volumes:
  postgres_data:
  redis_data:
```

**Verification**:
```bash
docker-compose up -d
docker-compose ps
```

---

### Task SPRINT0-006: Environment Variables Template

**Files to create**:
```bash
# .env.example
DATABASE_URL=sqlite+aiosqlite:///./gengowatcher.db
SECRET_KEY=
REDIS_URL=redis://localhost:6379/0

# OAuth
OAUTH_GOOGLE_CLIENT_ID=
OAUTH_GOOGLE_CLIENT_SECRET=
OAUTH_GITHUB_CLIENT_ID=
OAUTH_GITHUB_CLIENT_SECRET=

# Email
RESEND_API_KEY=

# Stripe
STRIPE_SECRET_KEY=
STRIPE_WEBHOOK_SECRET=
STRIPE_PRICE_ID_PRO=
STRIPE_PRICE_ID_ENTERPRISE=

# Frontend
FRONTEND_URL=http://localhost:5173
VITE_API_URL=http://localhost:8000
```

**Verification**:
```bash
cp .env.example .env
# Edit .env with actual values
```

---

### Task SPRINT0-007: Pre-commit Hooks

**Files to create**:
```bash
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/psf/black
    rev: 23.12.0
    hooks:
      - id: black
  - repo: https://github.com/pycqa/flake8
    rev: 7.0.0
    hooks:
      - id: flake8
        args: ["--ignore=E203,W503"]
  - repo: https://github.com/pre-commit/mirrors-mypy
    rev: v1.7.1
    hooks:
      - id: mypy
        additional_dependencies: [types-all]
```

**Verification**:
```bash
pre-commit install
pre-commit run --all-files
```

---

### Task SPRINT0-008: Create CLAUDE.md and .claude/ Structure

**Files to create**:
```bash
# Directory structure
mkdir -p .claude

# CLAUDE.md (project context - committed)
cat > CLAUDE.md << 'EOF'
# GengoWatcher SaaS

Multi-tenant job monitoring SaaS with per-user watcher instances.

## Commands

### Development
docker-compose up -d              # Start services
pytest tests/ -v                    # Run tests
pytest tests/ --cov=src/gengowatcher --cov-report=html
alembic revision --autogenerate -m "desc"
alembic upgrade head
mypy src/gengowatcher/
black src/gengowatcher/

## Current Sprint
SPRINT0 (Scaffolding)

## Conventions

### Backend
- Routes: /api/v1/*
- Error format: {"error": str, "code": str, "details": dict}
- Async/await for DB operations
- Tests mirror src/ structure

### Architecture
- User isolation: filter by user_id
- Redis keys: user:{user_id}:*
- WebSocket rooms: user:{user_id}:ws

## Design Language
- IBM Plex Sans (headings), IBM Plex Mono (labels)
- Data Factory: bento cards, precision hover, no shadow lift
- ROYGBIV accents for headings ONLY
EOF

# .claude/reference.md (code patterns - committed)
cat > .claude/reference.md << 'EOF'
# Canonical Code Patterns

## Database Model
See: src/gengowatcher/database/models.py, class User

## API Route
See: src/gengowatcher/auth/routes.py, @router.post("/login")

## Test Pattern
See: tests/test_auth_security.py

Always reference before creating similar.
EOF

# .claude/boundaries.md (sprint limits - committed)
cat > .claude/boundaries.md << 'EOF'
# Sprint 0 Boundaries

### Allowed (Scaffolding)
- requirements.txt
- Directory creation
- docker-compose.yml
- .env.example
- pytest.ini
- .pre-commit-config.yaml
- CLAUDE.md

### Off-Limits (Future Sprints)
- src/gengowatcher/database/*  # Sprint 1
- src/gengowatcher/auth/*      # Sprint 1
- src/gengowatcher/watcher/*   # Sprint 2
- src/gengowatcher/billing/*    # Sprint 6
- frontend/*                    # Sprint 4
EOF

# .claude/state.md (session state - NOT committed)
cat > .claude/state.md << 'EOF'
# Session State

## Current Sprint
SPRINT0-001 through SPRINT0-008

## Done This Session
[ ] Update requirements.txt
[ ] Create directories
[ ] Configure Alembic
[ ] Configure pytest
[ ] Docker Compose
[ ] .env.example
[ ] Pre-commit hooks
[ ] CLAUDE.md + .claude/ structure

## Modified Files
(none yet)
EOF

# .gitignore (add state.md)
echo ".claude/state.md" >> .gitignore
```

**Verification**:
```bash
ls -la .claude/
cat CLAUDE.md
git check-ignore .claude/state.md  # Should return .claude/state.md
```

---

## Sprint 1: Database & Models (TDD Approach)

### Task SPRINT1-001: Database Models
**Agent instructions**: Create SQLAlchemy models with tests first

**Files to create**:
```python
# src/gengowatcher/database/__init__.py
from .models import Base, User, OAuthAccount, APIKey, RefreshToken
from .session import get_db, get_db_session
__all__ = ["Base", "User", "get_db", "get_db_session", ...]

# src/gengowatcher/database/session.py
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession
from sqlalchemy.orm import sessionmaker
import os

DATABASE_URL = os.getenv("DATABASE_URL", "sqlite+aiosqlite:///./gengowatcher.db")
engine = create_async_engine(DATABASE_URL, echo=False)
AsyncSessionLocal = sessionmaker(engine, class_=AsyncSession, expire_on_commit=False)

async def get_db() -> AsyncSession:
    async with AsyncSessionLocal() as session:
        yield session

def get_db_session():
    return AsyncSessionLocal()

# src/gengowatcher/database/models.py
from sqlalchemy import Column, String, Boolean, DateTime, ForeignKey, Text, ARRAY, DECIMAL, Integer, JSON
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.dialects.postgresql import UUID, CITEXT, TIMESTAMPTZ
from sqlalchemy.orm import relationship
from datetime import datetime
import uuid

Base = declarative_base()

class TimestampMixin:
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow, nullable=False)

class User(Base, TimestampMixin):
    __tablename__ = "users"
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    email = Column(String(255), unique=True, nullable=False, index=True)
    email_verified = Column(Boolean, default=False, nullable=False)
    password_hash = Column(String(255), nullable=True)
    is_active = Column(Boolean, default=True, nullable=False)
    # Relationships
    oauth_accounts = relationship("OAuthAccount", back_populates="user", cascade="all, delete-orphan")
    api_keys = relationship("APIKey", back_populates="user", cascade="all, delete-orphan")
    refresh_tokens = relationship("RefreshToken", back_populates="user", cascade="all, delete-orphan")
    watcher_config = relationship("UserWatcherConfig", back_populates="user", uselist=False, cascade="all, delete-orphan")
    watcher_state = relationship("UserWatcherState", back_populates="user", uselist=False, cascade="all, delete-orphan")
    subscription = relationship("Subscription", back_populates="user", uselist=False, cascade="all, delete-orphan")

class OAuthAccount(Base, TimestampMixin):
    __tablename__ = "oauth_accounts"
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(UUID(as_uuid=True), ForeignKey("users.id", ondelete="CASCADE"), nullable=False)
    provider = Column(String(50), nullable=False)  # 'google', 'github'
    provider_user_id = Column(String(255), nullable=False)
    access_token = Column(Text)
    refresh_token = Column(Text)
    expires_at = Column(DateTime)
    user = relationship("User", back_populates="oauth_accounts")

class APIKey(Base, TimestampMixin):
    __tablename__ = "api_keys"
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(UUID(as_uuid=True), ForeignKey("users.id", ondelete="CASCADE"), nullable=False)
    key_hash = Column(String(64), unique=True, nullable=False, index=True)
    key_prefix = Column(String(10), nullable=False)
    name = Column(String(100), nullable=False)
    scopes = Column(JSON, default=list)
    is_active = Column(Boolean, default=True, nullable=False)
    last_used = Column(DateTime)
    user = relationship("User", back_populates="api_keys")

class RefreshToken(Base):
    __tablename__ = "refresh_tokens"
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(UUID(as_uuid=True), ForeignKey("users.id", ondelete="CASCADE"), nullable=False)
    token = Column(String(255), unique=True, nullable=False, index=True)
    expires_at = Column(DateTime, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    revoked_at = Column(DateTime)
    user = relationship("User", back_populates="refresh_tokens")

class UserWatcherConfig(Base, TimestampMixin):
    __tablename__ = "user_watcher_configs"
    user_id = Column(UUID(as_uuid=True), ForeignKey("users.id", ondelete="CASCADE"), primary_key=True)
    # RSS/WebSocket
    rss_feed_url = Column(Text, nullable=False, default="https://gengo.com/jobs/rss")
    websocket_enabled = Column(Boolean, default=True)
    gengo_user_id = Column(String(50))
    gengo_session_token = Column(Text)  # Encrypted
    # Filtering
    min_reward = Column(DECIMAL(10,2), default=0.0)
    max_reward = Column(DECIMAL(10,2), default=999999.0)
    included_language_pairs = Column(JSON, default=list)
    # Notifications
    enable_desktop_notifications = Column(Boolean, default=True)
    enable_sound_notifications = Column(Boolean, default=True)
    enable_email_notifications = Column(Boolean, default=False)
    notification_email = Column(String(255))
    # Auto-accept
    auto_accept_enabled = Column(Boolean, default=False)
    auto_accept_min_reward = Column(DECIMAL(10,2))
    auto_accept_max_reward = Column(DECIMAL(10,2))
    user = relationship("User", back_populates="watcher_config")

class UserWatcherState(Base):
    __tablename__ = "user_watcher_states"
    user_id = Column(UUID(as_uuid=True), ForeignKey("users.id", ondelete="CASCADE"), primary_key=True)
    # Deduplication
    last_seen_job_ids = Column(JSON, default=list)
    last_seen_rss_link = Column(Text)
    # Stats
    total_jobs_found = Column(Integer, default=0)
    total_jobs_accepted = Column(Integer, default=0)
    total_earnings = Column(DECIMAL(10,2), default=0.0)
    # Status
    watcher_status = Column(String(20), default="stopped")
    last_activity = Column(DateTime, default=datetime.utcnow)
    recent_job_history = Column(JSON, default=list)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow, nullable=False)
    user = relationship("User", back_populates="watcher_state")

class SubscriptionPlan(Base, TimestampMixin):
    __tablename__ = "subscription_plans"
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    name = Column(String(50), unique=True, nullable=False)
    price_cents = Column(Integer, default=0)
    interval = Column(String(20), default="month")
    features = Column(JSON, default={})
    stripe_price_id = Column(String(100))
    is_active = Column(Boolean, default=True)

class Subscription(Base, TimestampMixin):
    __tablename__ = "subscriptions"
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(UUID(as_uuid=True), ForeignKey("users.id", ondelete="CASCADE"), nullable=False)
    plan_id = Column(UUID(as_uuid=True), ForeignKey("subscription_plans.id"))
    stripe_customer_id = Column(String(100))
    stripe_subscription_id = Column(String(100), unique=True)
    stripe_subscription_status = Column(String(50))
    current_period_start = Column(TIMESTAMPTZ)
    current_period_end = Column(TIMESTAMPTZ)
    cancel_at_period_end = Column(Boolean, default=False)
    user = relationship("User", back_populates="subscription")

class BillingEvent(Base):
    __tablename__ = "billing_events"
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(UUID(as_uuid=True), ForeignKey("users.id", ondelete="SET NULL"))
    stripe_event_id = Column(String(100), unique=True)
    event_type = Column(String(50), nullable=False)
    event_data = Column(JSON)
    processed_at = Column(TIMESTAMPTZ, default=datetime.utcnow)

class AuditLog(Base):
    __tablename__ = "audit_log"
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    user_id = Column(UUID(as_uuid=True), ForeignKey("users.id", ondelete="SET NULL"))
    event_type = Column(String(50), nullable=False)
    event_data = Column(JSON)
    ip_address = Column(String(45))
    user_agent = Column(Text)
    created_at = Column(TIMESTAMPTZ, default=datetime.utcnow)
```

**Tests to create**:
```python
# tests/test_database_models.py
import pytest
from sqlalchemy.ext.asyncio import AsyncSession
from src.gengowatcher.database.models import User, APIKey, UserWatcherConfig

@pytest.mark.asyncio
async def test_create_user(db: AsyncSession):
    user = User(email="test@example.com", password_hash="hash")
    db.add(user)
    await db.commit()
    assert user.id is not None
    assert user.email == "test@example.com"

@pytest.mark.asyncio
async def test_user_watcher_config_relationship(db: AsyncSession):
    user = User(email="test@example.com")
    config = UserWatcherConfig(user=user, min_reward=5.0)
    db.add_all([user, config])
    await db.commit()
    assert user.watcher_config.min_reward == 5.0

@pytest.mark.asyncio
async def test_api_key_hash_stored(api_key_service):
    raw_key = api_key_service.generate_key()
    # Verify hash is stored, not raw key
    assert len(raw_key) > 20
    assert raw_key.startswith("gengo_sk_")
```

**Verification**:
```bash
pytest tests/test_database_models.py -v
alembic revision --autogenerate -m "Initial migration"
```

---

### Task SPRINT1-002: Alembic Setup

**Files to create**:
```bash
# Initialize Alembic
alembic init alembic

# alembic/env.py
from src.gengowatcher.database.models import Base
target_metadata = Base.metadata

# alembic/versions/001_initial.py (generated)
```

**Verification**:
```bash
alembic upgrade head
# Check tables created
sqlite3 gengowatcher.db ".tables"
```

---

### Task SPRINT1-003: Auth Security Module

**Files to create**:
```python
# src/gengowatcher/auth/security.py
from datetime import datetime, timedelta
from typing import Optional
from jose import JWTError, jwt
from passlib.context import CryptContext
import secrets

SECRET_KEY = os.getenv("SECRET_KEY", secrets.token_urlsafe(32))
ALGORITHM = "HS256"
ACCESS_TOKEN_EXPIRE_MINUTES = 15
REFRESH_TOKEN_EXPIRE_DAYS = 7

pwd_context = CryptContext(schemes=["argon2"], deprecated="auto")

def verify_password(plain_password: str, hashed_password: str) -> bool:
    return pwd_context.verify(plain_password, hashed_password)

def get_password_hash(password: str) -> str:
    return pwd_context.hash(password)

def create_access_token(data: dict) -> str:
    to_encode = data.copy()
    expire = datetime.utcnow() + timedelta(minutes=ACCESS_TOKEN_EXPIRE_MINUTES)
    to_encode.update({"exp": expire, "type": "access"})
    return jwt.encode(to_encode, SECRET_KEY, algorithm=ALGORITHM)

def create_refresh_token() -> str:
    """Create a random refresh token."""
    return secrets.token_urlsafe(32)

def decode_token(token: str) -> Optional[dict]:
    try:
        payload = jwt.decode(token, SECRET_KEY, algorithms=[ALGORITHM])
        return payload
    except JWTError:
        return None

# src/gengowatcher/auth/exceptions.py
class AuthException(Exception):
    def __init__(self, message: str, code: str, details: dict = None):
        self.message = message
        self.code = code
        self.details = details or {}
        super().__init__(message)

class UserNotFoundException(AuthException):
    def __init__(self, email: str):
        super().__init__(
            message=f"User with email {email} not found",
            code="USER_NOT_FOUND",
            details={"email": email}
        )

class InvalidCredentialsException(AuthException):
    def __init__(self):
        super().__init__(
            message="Invalid email or password",
            code="INVALID_CREDENTIALS"
        )
```

**Tests**:
```python
# tests/test_auth_security.py
def test_password_hashing():
    hash = get_password_hash("testpass")
    assert verify_password("testpass", hash)
    assert not verify_password("wrong", hash)

def test_access_token_creation():
    token = create_access_token({"sub": "user-id"})
    payload = decode_token(token)
    assert payload["sub"] == "user-id"
    assert payload["type"] == "access"
```

---

### Task SPRINT1-004: Auth Service

**Files to create**:
```python
# src/gengowatcher/auth/service.py
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import select
from src.gengowatcher.database.models import User, RefreshToken, UserWatcherConfig, UserWatcherState
from src.gengowatcher.auth.security import verify_password, get_password_hash, create_access_token, create_refresh_token
from src.gengowatcher.auth.exceptions import UserNotFoundException, InvalidCredentialsException
from datetime import datetime, timedelta
import uuid

class AuthService:
    def __init__(self, db: AsyncSession):
        self.db = db

    async def register_user(self, email: str, password: str) -> User:
        # Check if exists
        result = await self.db.execute(select(User).where(User.email == email))
        if result.scalar_one_or_none():
            raise AuthException("User already exists", "USER_EXISTS")

        # Create user
        user = User(
            email=email,
            password_hash=get_password_hash(password)
        )
        self.db.add(user)

        # Create default watcher config
        config = UserWatcherConfig(user=user)
        self.db.add(config)

        # Create default watcher state
        state = UserWatcherState(user=user)
        self.db.add(state)

        await self.db.commit()
        await self.db.refresh(user)
        return user

    async def authenticate_user(self, email: str, password: str) -> tuple[User, str, str]:
        result = await self.db.execute(select(User).where(User.email == email))
        user = result.scalar_one_or_none()
        if not user or not verify_password(password, user.password_hash or ""):
            raise InvalidCredentialsException()

        # Create tokens
        access_token = create_access_token({"sub": str(user.id)})
        refresh_token = create_refresh_token()

        # Store refresh token
        rt = RefreshToken(
            user_id=user.id,
            token=refresh_token,
            expires_at=datetime.utcnow() + timedelta(days=7)
        )
        self.db.add(rt)
        await self.db.commit()

        return user, access_token, refresh_token

    async def refresh_tokens(self, refresh_token: str) -> tuple[str, str] | None:
        result = await self.db.execute(
            select(RefreshToken, User)
            .join(User, RefreshToken.user_id == User.id)
            .where(RefreshToken.token == refresh_token)
            .where(RefreshToken.revoked_at == None)
        )
        row = result.first()
        if not row:
            return None

        rt, user = row

        # Revoke old token
        rt.revoked_at = datetime.utcnow()

        # Create new tokens
        new_access = create_access_token({"sub": str(user.id)})
        new_refresh = create_refresh_token()

        new_rt = RefreshToken(
            user_id=user.id,
            token=new_refresh,
            expires_at=datetime.utcnow() + timedelta(days=7)
        )
        self.db.add_all([rt, new_rt])
        await self.db.commit()

        return new_access, new_refresh
```

**Tests**:
```python
# tests/test_auth_service.py
@pytest.mark.asyncio
async def test_register_user(db: AsyncSession):
    service = AuthService(db)
    user = await service.register_user("test@example.com", "password")
    assert user.email == "test@example.com"
    assert user.watcher_config is not None

@pytest.mark.asyncio
async def test_authenticate_user(db: AsyncSession):
    service = AuthService(db)
    await service.register_user("test@example.com", "password")
    user, access, refresh = await service.authenticate_user("test@example.com", "password")
    assert access is not None
    assert refresh is not None

@pytest.mark.asyncio
async def test_wrong_password_raises(db: AsyncSession):
    service = AuthService(db)
    await service.register_user("test@example.com", "password")
    with pytest.raises(InvalidCredentialsException):
        await service.authenticate_user("test@example.com", "wrong")
```

---

### Task SPRINT1-005: Auth API Routes

**Files to create**:
```python
# src/gengowatcher/auth/routes.py
from fastapi import APIRouter, HTTPException, Depends, Response, Cookie
from pydantic import BaseModel, EmailStr
from sqlalchemy.ext.asyncio import AsyncSession
from src.gengowatcher.database.session import get_db
from src.gengowatcher.auth.service import AuthService
from src.gengowatcher.auth.exceptions import AuthException

router = APIRouter(prefix="/api/v1/auth", tags=["auth"])

class RegisterRequest(BaseModel):
    email: EmailStr
    password: str

class LoginRequest(BaseModel):
    email: EmailStr
    password: str

class AuthResponse(BaseModel):
    access_token: str
    refresh_token: str

@router.post("/register", response_model=AuthResponse)
async def register(req: RegisterRequest, db: AsyncSession = Depends(get_db)):
    try:
        service = AuthService(db)
        user = await service.register_user(req.email, req.password)
        _, access, refresh = await service.authenticate_user(req.email, req.password)
        return AuthResponse(access_token=access, refresh_token=refresh)
    except AuthException as e:
        raise HTTPException(status_code=400, detail={"error": e.message, "code": e.code})

@router.post("/login", response_model=AuthResponse)
async def login(req: LoginRequest, db: AsyncSession = Depends(get_db), response: Response = None):
    try:
        service = AuthService(db)
        user, access, refresh = await service.authenticate_user(req.email, req.password)

        # Set httpOnly cookie
        response = Response()
        response.set_cookie(
            key="session_token",
            value=refresh,
            httponly=True,
            secure=True,  # prod only
            samesite="lax"
        )

        return AuthResponse(access_token=access, refresh_token=refresh)
    except AuthException as e:
        raise HTTPException(status_code=401, detail={"error": e.message, "code": e.code})

# Standard error handler
@router.exception_handler(AuthException)
async def auth_exception_handler(request, exc: AuthException):
    return JSONResponse(
        status_code=400,
        content={"error": exc.message, "code": exc.code, "details": exc.details}
    )
```

**Tests**:
```python
# tests/test_auth_routes.py
from fastapi.testclient import TestClient

def test_register(client: TestClient):
    response = client.post("/api/v1/auth/register", json={
        "email": "test@example.com",
        "password": "password123"
    })
    assert response.status_code == 200
    data = response.json()
    assert "access_token" in data
    assert "refresh_token" in data

def test_login_wrong_password(client: TestClient):
    response = client.post("/api/v1/auth/login", json={
        "email": "test@example.com",
        "password": "wrong"
    })
    assert response.status_code == 401
```

---

### Task SPRINT1-006: Update Config

**Files to modify**:
```python
# src/gengowatcher/config.py - Add these sections to DEFAULT_CONFIG

"Database": {
    "url": "sqlite+aiosqlite:///./gengowatcher.db",
    "echo": False,
},

"Auth": {
    "secret_key": "",  # Generate if empty
    "access_token_expire_minutes": 15,
    "refresh_token_expire_days": 7,
},

"OAuth": {
    "google_client_id": "",
    "google_client_secret": "",
    "github_client_id": "",
    "github_client_secret": "",
},

"Email": {
    "provider": "resend",
    "resend_api_key": "",
    "from_email": "noreply@gengowatcher.com",
    "from_name": "GengoWatcher",
},

"Stripe": {
    "secret_key": "",
    "webhook_secret": "",
    "price_id_pro": "",
    "price_id_enterprise": "",
},

"Redis": {
    "url": "redis://localhost:6379/0",
},
```

---

## Sprint 2: UserWatcherManager (Extract & Refactor)

### Task SPRINT2-001: Extract RSS Monitor

**Strategy**: Copy working RSS logic from `watcher.py`, add `user_id` parameter

**Files to create**:
```python
# src/gengowatcher/watcher/rss.py
from typing import Callable, Awaitable
import feedparser
import asyncio
from dataclasses import dataclass

@dataclass
class Job:
    id: str
    title: str
    reward: float
    url: str
    user_id: str

class RSSMonitor:
    def __init__(self, feed_url: str, user_id: str, min_reward: float = 0.0):
        self.feed_url = feed_url
        self.user_id = user_id
        self.min_reward = min_reward
        self.seen_ids = set()
        self.running = False

    async def run(self, on_job: Callable[[Job], Awaitable[None]]):
        """Run RSS monitoring, calling on_job for each new job."""
        self.running = True
        while self.running:
            try:
                feed = feedparser.parse(self.feed_url)
                for entry in feed.entries:
                    job_id = entry.get("id", entry.link)
                    if job_id not in self.seen_ids:
                        reward = self._extract_reward(entry)
                        if reward >= self.min_reward:
                            job = Job(
                                id=job_id,
                                title=entry.title,
                                reward=reward,
                                url=entry.link,
                                user_id=self.user_id
                            )
                            await on_job(job)
                        self.seen_ids.add(job_id)
            except Exception as e:
                logger.error(f"RSS error for user {self.user_id}: {e}")

            await asyncio.sleep(30)  # Configurable

    def _extract_reward(self, entry) -> float:
        # Extract reward from entry
        # ... existing logic from watcher.py
        pass

    def stop(self):
        self.running = False
```

**Tests**:
```python
# tests/test_rss_monitor.py
@pytest.mark.asyncio
async def test_rss_monitor_calls_on_job():
    jobs_received = []
    async def collect(job):
        jobs_received.append(job)

    monitor = RSSMonitor("http://test.com/rss", "user-123", min_reward=5.0)
    # Mock feedparser.parse
    # Run monitor
    # Assert jobs_received
```

---

### Task SPRINT2-002: Extract WebSocket Monitor

**Similar to RSSMonitor** - Copy from `watcher.py`, add `user_id`

---

### Task SPRINT2-003: UserWatcherManager

**Files to create**:
```python
# src/gengowatcher/watcher/manager.py
import asyncio
from typing import Dict, Optional
from dataclasses import dataclass
from sqlalchemy.ext.asyncio import AsyncSession
from src.gengowatcher.database.models import User, UserWatcherConfig, UserWatcherState
from src.gengowatcher.watcher.rss import RSSMonitor
from src.gengowatcher.watcher.websocket import WebSocketMonitor

@dataclass
class WatcherInstance:
    user_id: str
    config: UserWatcherConfig
    rss_task: Optional[asyncio.Task]
    ws_task: Optional[asyncio.Task]
    status: str = "stopped"

class UserWatcherManager:
    def __init__(self, db: AsyncSession, redis_client):
        self.db = db
        self.redis = redis_client
        self._watchers: Dict[str, WatcherInstance] = {}
        self._lock = asyncio.Lock()

    async def start_watcher(self, user_id: str) -> bool:
        async with self._lock:
            if user_id in self._watchers:
                return True

            # Load config
            config = await self._load_config(user_id)
            if not config:
                return False

            # Create monitors
            rss = RSSMonitor(config.rss_feed_url, user_id, config.min_reward)
            ws = WebSocketMonitor(config.gengo_session_token, user_id)

            # Job callback
            async def on_job(job):
                await self._handle_new_job(user_id, job)

            # Start tasks
            rss_task = asyncio.create_task(rss.run(on_job))
            ws_task = asyncio.create_task(ws.run(on_job))

            self._watchers[user_id] = WatcherInstance(
                user_id=user_id,
                config=config,
                rss_task=rss_task,
                ws_task=ws_task,
                status="running"
            )

            # Update state
            await self._update_state(user_id, "running")
            await self._notify_user(user_id, "watcher_started")
            return True

    async def stop_watcher(self, user_id: str) -> bool:
        async with self._lock:
            watcher = self._watchers.get(user_id)
            if not watcher:
                return False

            watcher.rss_task.cancel()
            watcher.ws_task.cancel()
            del self._watchers[user_id]

            await self._update_state(user_id, "stopped")
            await self._notify_user(user_id, "watcher_stopped")
            return True

    async def get_status(self, user_id: str) -> dict:
        watcher = self._watchers.get(user_id)
        if watcher:
            return {"status": watcher.status}

        # Load from DB
        state = await self._load_state(user_id)
        return {"status": state.watcher_status if state else "stopped"}

    async def _handle_new_job(self, user_id: str, job):
        # Store in user's job history
        # Publish to Redis
        await self.redis.publish(f"user:{user_id}:jobs", job.json())

    async def _notify_user(self, user_id: str, event: str):
        await self.redis.publish(f"user:{user_id}:events", event)

    async def _load_config(self, user_id: str) -> Optional[UserWatcherConfig]:
        from sqlalchemy import select
        result = await self.db.execute(
            select(UserWatcherConfig).where(UserWatcherConfig.user_id == user_id)
        )
        return result.scalar_one_or_none()

    async def _load_state(self, user_id: str) -> Optional[UserWatcherState]:
        from sqlalchemy import select
        result = await self.db.execute(
            select(UserWatcherState).where(UserWatcherState.user_id == user_id)
        )
        return result.scalar_one_or_none()

    async def _update_state(self, user_id: str, status: str):
        from sqlalchemy import select, update
        stmt = (
            update(UserWatcherState)
            .where(UserWatcherState.user_id == user_id)
            .values(watcher_status=status, last_activity=datetime.utcnow())
        )
        await self.db.execute(stmt)
        await self.db.commit()
```

**Tests**:
```python
# tests/test_watcher_manager.py
@pytest.mark.asyncio
async def test_start_watcher(db: AsyncSession, redis):
    manager = UserWatcherManager(db, redis)
    # Create user with config
    user = User(email="test@example.com")
    config = UserWatcherConfig(user=user)
    db.add_all([user, config])
    await db.commit()

    result = await manager.start_watcher(str(user.id))
    assert result is True
    status = await manager.get_status(str(user.id))
    assert status["status"] == "running"

@pytest.mark.asyncio
async def test_user_isolation(db: AsyncSession, redis):
    # User A and User B should not see each other's jobs
    manager = UserWatcherManager(db, redis)
    # ... isolation test
```

---

## Sprint 3: WebSocket & Redis

### Task SPRINT3-001: Redis Pub/Sub Manager

**Files to create**:
```python
# src/gengowatcher/redis_pubsub.py
import redis.asyncio as redis
import json
from typing import Callable

class RedisPubSub:
    def __init__(self, url: str = "redis://localhost:6379/0"):
        self.client = redis.from_url(url, decode_responses=True)

    async def publish(self, channel: str, message: dict):
        await self.client.publish(channel, json.dumps(message))

    async def subscribe(self, channel: str, callback: Callable):
        async with self.client.pubsub() as pubsub:
            await pubsub.subscribe(channel)
            async for message in pubsub.listen():
                if message["type"] == "message":
                    await callback(json.loads(message["data"]))

    async def unsubscribe(self, channel: str):
        await self.client.unsubscribe(channel)
```

---

### Task SPRINT3-002: WebSocket Endpoint

**Files to modify**:
```python
# src/gengowatcher/web.py - Add this endpoint

from fastapi import WebSocket, WebSocketDisconnect, Query, Cookie
from src.gengowatcher.auth.security import decode_token
from src.gengowatcher.redis_pubsub import RedisPubSub

pubsub = RedisPubSub(settings.REDIS_URL)
active_connections: dict[str, list[WebSocket]] = {}

@app.websocket("/ws")
async def websocket_endpoint(
    websocket: WebSocket,
    token: str = Query(None)
):
    # Try cookie first
    session_token = websocket.cookies.get("session_token")
    if session_token:
        # Verify against refresh tokens in DB
        user = await get_user_by_refresh_token(session_token)
    elif token:
        payload = decode_token(token)
        if payload:
            user = await get_user_by_id(payload["sub"])
    else:
        await websocket.close(code=1008)
        return

    await websocket.accept()

    # Add to user's room
    room = f"user:{user.id}:ws"
    if room not in active_connections:
        active_connections[room] = []
    active_connections[room].append(websocket)

    try:
        # Subscribe to user's events
        async def handler(msg):
            await websocket.send_json(msg)

        await pubsub.subscribe(room, handler)
    except WebSocketDisconnect:
        active_connections[room].remove(websocket)
```

---

## Sprint 4: Frontend Auth

### Task SPRINT4-001: API Client with Auth

**Files to modify**:
```typescript
// frontend/src/lib/api.ts
import axios, { AxiosError } from 'axios';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8000';

export interface ApiError {
  error: string;
  code: string;
  details?: Record<string, unknown>;
}

export const api = axios.create({
  baseURL: API_URL,
  withCredentials: true,  // Send cookies
});

// Request interceptor (add access token)
api.interceptors.request.use((config) => {
  const token = sessionStorage.getItem('access_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor (handle 401, refresh token)
api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiError>) => {
    if (error.response?.status === 401) {
      // Try refresh
      const refreshToken = localStorage.getItem('refresh_token');
      if (refreshToken) {
        try {
          const { data } = await axios.post(`${API_URL}/api/v1/auth/refresh`, {
            refresh_token: refreshToken,
          });
          sessionStorage.setItem('access_token', data.access_token);
          localStorage.setItem('refresh_token', data.refresh_token);
          // Retry original request
          return api.request(error.config);
        } catch {
          // Redirect to login
          window.location.href = '/login';
        }
      }
    }
    return Promise.reject(error);
  }
);

// Auth API
export const authApi = {
  register: (email: string, password: string) =>
    api.post<{ access_token: string; refresh_token: string }>('/auth/register', {
      email,
      password,
    }),
  login: (email: string, password: string) =>
    api.post<{ access_token: string; refresh_token: string }>('/auth/login', {
      email,
      password,
    }),
  logout: () => api.post('/auth/logout'),
  me: () => api.get<User>('/auth/me'),
};
```

---

### Task SPRINT4-002: Auth Provider

**Files to create**:
```typescript
// frontend/src/contexts/AuthProvider.tsx
import { createContext, useContext, useEffect, useState } from 'react';

interface User {
  id: string;
  email: string;
  email_verified: boolean;
}

interface AuthContextValue {
  user: User | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  loading: boolean;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Check if logged in
    authApi.me()
      .then(({ data }) => setUser(data))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const login = async (email: string, password: string) => {
    const { data } = await authApi.login(email, password);
    sessionStorage.setItem('access_token', data.access_token);
    localStorage.setItem('refresh_token', data.refresh_token);
    const user = await authApi.me();
    setUser(user.data);
  };

  const register = async (email: string, password: string) => {
    const { data } = await authApi.register(email, password);
    sessionStorage.setItem('access_token', data.access_token);
    localStorage.setItem('refresh_token', data.refresh_token);
    const user = await authApi.me();
    setUser(user.data);
  };

  const logout = async () => {
    await authApi.logout();
    sessionStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, login, register, logout, loading }}>
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used within AuthProvider');
  return context;
};
```

---

### Task SPRINT4-003: Protected Route

**Files to create**:
```typescript
// frontend/src/routes/ProtectedRoute.tsx
import { Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthProvider';

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();

  if (loading) return <div>Loading...</div>;
  if (!user) return <Navigate to="/login" replace />;
  return <>{children}</>;
}
```

---

### Task SPRINT4-004: Login Page (Data Factory Design)

**Files to create**:
```typescript
// frontend/src/pages/login.tsx
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthProvider';
import './login.css';  // IBM Plex, bento cards, precision hover

export function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await login(email, password);
      navigate('/app');
    } catch (err) {
      setError('Invalid email or password');
    }
  };

  return (
    <div className="login-page">
      <div className="bento-card login-card">
        <h1 className="text-9xl font-light tracking-tighter">Sign In</h1>
        <form onSubmit={handleSubmit}>
          <input
            type="email"
            placeholder="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="text-input"
          />
          <input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="text-input"
          />
          {error && <div className="error-text">{error}</div>}
          <button type="submit" className="btn-primary">Sign In</button>
        </form>
        <div className="auth-divider">or</div>
        <button className="btn-oauth">Continue with Google</button>
        <button className="btn-oauth">Continue with GitHub</button>
        <div className="auth-footer">
          <a href="/register">Create account</a>
          <span className="text-mono text-[11px] uppercase tracking-widest">
            Don't have an account?
          </span>
        </div>
      </div>
    </div>
  );
}
```

**CSS** (Data Factory design):
```css
/* frontend/src/pages/login.css */
@import url('https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@300;400;600&family=IBM+Plex+Mono:wght@400;600&display=swap');

body {
  font-family: 'IBM Plex Sans', sans-serif;
  background: #f5f5f5;
  color: #171717;
}

.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.bento-card {
  background: #ffffff;
  border: 1px solid #e5e5e5;
  padding: 32px;
  width: 100%;
  max-width: 400px;
}

.bento-card:hover {
  border-color: #3b82f6;  /* Precision focus */
}

.text-9xl {
  font-size: 48px;
  font-weight: 300;
  letter-spacing: -0.02em;
  margin-bottom: 24px;
}

.text-input {
  width: 100%;
  padding: 12px;
  border: 1px solid #e5e5e5;
  font-family: 'IBM Plex Sans', sans-serif;
  font-size: 14px;
  margin-bottom: 16px;
}

.text-input:focus {
  outline: none;
  border-color: #3b82f6;
}

.btn-primary {
  width: 100%;
  padding: 12px;
  background: #171717;
  color: #ffffff;
  border: none;
  font-family: 'IBM Plex Sans', sans-serif;
  font-weight: 400;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-primary:hover {
  background: #3b82f6;
}

.btn-oauth {
  width: 100%;
  padding: 12px;
  background: #ffffff;
  color: #171717;
  border: 1px solid #e5e5e5;
  font-family: 'IBM Plex Sans', sans-serif;
  cursor: pointer;
  margin-bottom: 8px;
}

.btn-oauth:hover {
  border-color: #3b82f6;
}

.text-mono {
  font-family: 'IBM Plex Mono', monospace;
}

.text-[11px] {
  font-size: 11px;
}

.uppercase {
  text-transform: uppercase;
}

.tracking-widest {
  letter-spacing: 0.1em;
}
```

---

## Sprint 5: Email & Magic Links

### Task SPRINT5-001: Email Service

**Files to create**:
```python
# src/gengowatcher/email/service.py
import resend
from src.gengowatcher.config import AppConfig

resend.api_key = Config.get("Email", "resend_api_key")

class EmailService:
    def __init__(self):
        self.from_email = Config.get("Email", "from_email")

    async def send_magic_link(self, email: str, token: str):
        link = f"{Config.get('WebServer', 'frontend_url')}/auth/verify?token={token}"
        params = {
            "from": self.from_email,
            "to": email,
            "subject": "Sign in to GengoWatcher",
            "html": f"""
            <p>Click the link below to sign in:</p>
            <a href="{link}">Sign In</a>
            <p>This link expires in 15 minutes.</p>
            """
        }
        resend.Emails.send(params)

    async def send_verification(self, email: str, code: str):
        # Similar
        pass
```

---

## Sprint 6: Billing (Stripe)

**Files to create**:
```python
# src/gengowatcher/billing/routes.py
import stripe
from fastapi import APIRouter, Request, HTTPException

router = APIRouter(prefix="/api/v1/billing", tags=["billing"])
stripe.api_key = settings.STRIPE_SECRET_KEY

@router.post("/create-checkout-session")
async def create_checkout_session(user_id: str, price_id: str):
    session = stripe.checkout.Session.create(
        payment_method_types=["card"],
        line_items=[{"price": price_id, "quantity": 1}],
        mode="subscription",
        success_url=f"{settings.FRONTEND_URL}/app?subscription=success",
        cancel_url=f"{settings.FRONTEND_URL}/app?subscription=canceled",
        client_reference_id=user_id,
        metadata={"user_id": user_id, "plan_id": "pro"}
    )
    return {"url": session.url}

@router.post("/webhooks/stripe")
async def stripe_webhook(request: Request):
    # ... as shown in earlier plan
    pass
```

---

## Verification Commands

```bash
# Run after each sprint
./scripts/verify.sh

#!/bin/bash
set -e
echo "Running tests..."
pytest tests/ -v
echo "Running mypy..."
mypy src/gengowatcher/
echo "Running flake8..."
flake8 src/gengowatcher/
echo "Building frontend..."
cd frontend && npm run build
echo "All checks passed!"
```

---

## Dependencies to Add

```
# requirements.txt additions
sqlalchemy[asyncio]>=2.0.0
aiosqlite>=0.19.0
asyncpg>=0.29.0  # For PostgreSQL
alembic>=1.12.0
argon2-cffi>=23.0.0
python-jose[cryptography]>=3.3.0
passlib>=1.7.4
authlib>=1.2.1
resend>=0.8.0
stripe>=7.0.0
redis[hiredis]>=4.6.0
fastapi-limiter>=0.1.0
```

---

## Task Dependency Graph

```
SPRINT0 (Scaffolding - all parallel)
    ├─> SPRINT1-001 (DB Models)
    ├─> SPRINT0-003 (Alembic) ─> SPRINT1-002 (Initial Migration)
    ├─> SPRINT0-004 (pytest) ─> (supports all test writing)
    ├─> SPRINT0-005 (docker-compose) ─> (local dev environment)
    └─> SPRINT0-008 (CLAUDE.md) ─> (project documentation)

SPRINT1-001 (DB Models)
    ├─> SPRINT1-002 (Alembic migration)
    └─> SPRINT1-003 (Security)
SPRINT1-003
    └─> SPRINT1-004 (Service)
SPRINT1-004
    └─> SPRINT1-005 (Routes)
SPRINT1-001
    └─> SPRINT2-001 (Extract RSS)
SPRINT2-001
    └─> SPRINT2-003 (Manager)
SPRINT1-005
    └─> SPRINT4-001 (API Client)
SPRINT4-001
    ├─> SPRINT4-002 (Provider)
    └─> SPRINT4-004 (Login Page)
SPRINT2-003
    └─> SPRINT3-002 (WebSocket)
```

---

## Remaining Tasks (Post-MVP)

- Google/GitHub OAuth
- API key management UI
- Dashboard redesign (Data Factory)
- Deployment (Docker, Railway/Fly.io)
- Monitoring & logging
- Documentation
