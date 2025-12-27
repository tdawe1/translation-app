# Canonical Code Patterns

## Database Model
See: src/gengowatcher/database/models.py, class User

```python
class User(Base, TimestampMixin):
    __tablename__ = "users"
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    email = Column(String(255), unique=True, nullable=False, index=True)
    # ... relationships
```

## API Route
See: src/gengowatcher/auth/routes.py, @router.post("/login")

```python
@router.post("/login")
async def login(req: LoginRequest, db: AsyncSession = Depends(get_db)):
    # ...
    return AuthResponse(access_token=access, refresh_token=refresh)
```

## Test Pattern
See: tests/test_auth_security.py

```python
@pytest.mark.asyncio
async def test_password_hashing():
    hash = get_password_hash("testpass")
    assert verify_password("testpass", hash)
```

Always reference before creating similar.
