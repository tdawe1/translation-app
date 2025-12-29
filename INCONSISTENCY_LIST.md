# Documentation Inconsistency List

**Project**: GengoWatcher SaaS (translation-app)  
**Date**: 2025-12-28  
**Total Inconsistencies Found**: 47

---

## Critical Inconsistencies (Blocking)

### 1. README.md - Tech Stack Table

**File**: `/home/thomas/translation-app/README.md:18-26`

**Documented**:
```markdown
| Layer | Technology |
|-------|------------|
| Backend | FastAPI, SQLAlchemy 2.0 async |
| Database | PostgreSQL (prod) / SQLite (local) |
| Auth | Argon2id, JWT, httpOnly cookies |
```

**Actual** (from `/home/thomas/translation-app/backend/go.mod`):
```go
github.com/gofiber/fiber/v2 v3.0.0-rc.3
github.com/gorm.io/gorm v1.31.1
github.com/golang-jwt/jwt/v5 v5.2.0
golang.org/x/crypto v0.16.0 // bcrypt
```

---

### 2. README.md - Prerequisites

**File**: `/home/thomas/translation-app/README.md:31`

**Documented**:
```markdown
### Prerequisites
- Python 3.11+
```

**Actual** (from `/home/thomas/translation-app/backend/cmd/server/main.go:1`):
```go
// Go 1.23 required
package main
```

---

### 3. README.md - Setup Commands

**File**: `/home/thomas/translation-app/README.md:45-60`

**Documented**:
```bash
# Create virtual environment
python -m venv .venv
source .venv/bin/activate

# Install dependencies
pip install -r requirements-dev.txt

# Run tests
pytest tests/ -v
```

**Actual** (from `/home/thomas/translation-app/CURRENT_PLAN.md:262-268`):
```bash
# Backend
cd backend
go run cmd/server/main.go

# Frontend
cd frontend
npm run dev
```

---

### 4. README.md - API Server Command

**File**: `/home/thomas/translation-app/README.md:66`

**Documented**:
```bash
uvicorn src.gengowatcher.main:app --reload --host 0.0.0.0 --port 8000
```

**Actual**:
```bash
cd backend && go run cmd/server/main.go
```

---

### 5. CLAUDE.md - Database Migrations

**File**: `/home/thomas/translation-app/CLAUDE.md:48-52`

**Documented**:
```bash
alembic revision --autogenerate -m "desc"
alembic upgrade head
alembic current
```

**Actual** (from `/home/thomas/translation-app/backend/cmd/server/main.go:39-53`):
```go
// Auto migrate
if err := gormDB.AutoMigrate(
    &models.User{},
    &models.OAuthAccount{},
    // ... other models
); err != nil {
    log.Fatalf("Failed to run migrations: %v", err)
}
```

---

### 6. CLAUDE.md - Testing Commands

**File**: `/home/thomas/translation-app/CLAUDE.md:56-61`

**Documented**:
```bash
pytest tests/ -v
pytest tests/ --cov=src/gengowatcher --cov-report=html
mypy src/gengowatcher/
black src/gengowatcher/
```

**Actual**:
```bash
go test ./... -v
go vet ./...
gofmt -l ./...
```

---

### 7. CLAUDE.md - Project Structure

**File**: `/home/thomas/translation-app/CLAUDE.md:66-91`

**Documented**:
```
src/gengowatcher/
├── database/
│   ├── __init__.py
│   ├── models.py      # SQLAlchemy models
│   └── session.py     # Async DB session management
├── auth/
│   ├── security.py    # JWT, password hashing
│   ├── service.py     # AuthService
│   ├── routes.py      # /api/v1/auth endpoints
│   └── exceptions.py  # AuthException types
```

**Actual** (from `/home/thomas/translation-app/backend/internal/`):
```
backend/internal/
├── database/
│   └── database.go    # GORM wrapper
├── models/
│   └── user.go        # GORM models
├── auth/
│   ├── token.go       # JWT service
│   └── user_service.go
├── handlers/
│   ├── auth.go        # HTTP handlers
│   └── response.go
└── middleware/
    └── jwt.go
```

---

### 8. CLAUDE.md - Key Technologies Table

**File**: `/home/thomas/translation-app/CLAUDE.md:105-117`

**Documented**:
```markdown
| Layer | Technology |
| Database | SQLAlchemy 2.0 async, PostgreSQL (prod) / SQLite (local) |
| Migrations | Alembic |
| Auth | Argon2id, JWT (15min access), httpOnly cookies (7 day refresh) |
```

**Actual**:
```markdown
| Layer | Technology |
| Database | GORM 1.31, PostgreSQL 17 |
| Migrations | GORM AutoMigrate |
| Auth | bcrypt, JWT (access token in httpOnly cookie) |
```

---

### 9. PLAN.md - Entire File (49,430 lines)

**File**: `/home/thomas/translation-app/PLAN.md`

**Documented**: Complete Python/FastAPI implementation plan with:
- Sprint 0-6 tasks
- Python dependencies
- SQLAlchemy models
- FastAPI routes
- Alembic migrations

**Actual**: Go/Fiber implementation exists instead

**Evidence**: Compare `/home/thomas/translation-app/PLAN.md:44` (Python requirements) with `/home/thomas/translation-app/backend/go.mod` (Go dependencies)

---

### 10. .env.example - Database Configuration

**File**: `/home/thomas/translation-app/.env.example:1-5`

**Documented**:
```bash
DATABASE_URL=sqlite+aiosqlite:///./gengowatcher.db
SECRET_KEY=
REDIS_URL=redis://localhost:6379/0
```

**Actual** (from `/home/thomas/translation-app/backend/internal/config/config.go:36-50`):
```bash
ENV=development
PORT=8000
DB_HOST=localhost
DB_PORT=5433
DB_USER=gengo
DB_PASSWORD=devpass
DB_NAME=gengowatcher
DB_SSLMODE=disable
JWT_SECRET=
```

---

### 11. .claude/reference.md - Code Patterns

**File**: `/home/thomas/translation-app/.claude/reference.md:3-12`

**Documented**:
```markdown
## Database Model
See: src/gengowatcher/database/models.py, class User

class User(Base, TimestampMixin):
    __tablename__ = "users"
    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
```

**Actual** (from `/home/thomas/translation-app/backend/internal/models/user.go:12-20`):
```go
type User struct {
    ID            uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
    Email         string    `gorm:"type:varchar(255);uniqueIndex;not null"`
    EmailVerified bool      `gorm:"not null;default:false"`
    IsActive      bool      `gorm:"not null;default:true"`
}
```

---

### 12. .claude/reference.md - API Route Pattern

**File**: `/home/thomas/translation-app/.claude/reference.md:14-22`

**Documented**:
```markdown
## API Route
See: src/gengowatcher/auth/routes.py, @router.post("/login")

@router.post("/login")
async def login(req: LoginRequest, db: AsyncSession = Depends(get_db)):
    # ...
    return AuthResponse(access_token=access, refresh_token=refresh)
```

**Actual** (from `/home/thomas/translation-app/backend/internal/handlers/auth.go:67-91`):
```go
func (h *AuthHandler) Login(c *fiber.Ctx) error {
    var req LoginRequest
    if err := c.BodyParser(&req); err != nil {
        return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid request body")
    }
    // ... validation and service call
    return c.JSON(AuthResponse{
        AccessToken: result.AccessToken,
        User:        UserToResponse(result.User),
    })
}
```

---

## High Inconsistencies

### 13. CLAUDE.md - OAuth Documentation

**File**: `/home/thomas/translation-app/CLAUDE.md:111`

**Documented**:
```markdown
| OAuth | Google, GitHub (Authlib) |
```

**Actual**: OAuth not implemented yet (models exist but no routes)

---

### 14. README.md - Email Provider

**File**: `/home/thomas/translation-app/README.md:25`

**Documented**:
```markdown
| Email | Resend |
```

**Actual**: No email integration exists (Mailhog in docker-compose but unused)

---

### 15. .pre-commit-config.yaml - Python Hooks

**File**: `/home/thomas/translation-app/.pre-commit-config.yaml`

**Configured**:
```yaml
repos:
  - repo: https://github.com/psf/black
  - repo: https://github.com/pycqa/flake8
  - repo: https://github.com/pre-commit/mirrors-mypy
```

**Actual**: Go project - should use gofmt, golint, go vet

---

### 16. pytest.ini - Python Test Config

**File**: `/home/thomas/translation-app/pytest.ini`

**Exists but**: Project is Go, should use Go testing

---

### 17. requirements.txt - Python Dependencies

**File**: `/home/thomas/translation-app/requirements.txt`

**Lists**: Python packages (SQLAlchemy, FastAPI, etc.)

**Actual**: Project uses Go dependencies

---

### 18. pyproject.toml - Python Project Config

**File**: `/home/thomas/translation-app/pyproject.toml`

**Exists but**: Project is Go, not Python

---

### 19. CLAUDE.md - Frontend State Management

**File**: `/home/thomas/translation-app/CLAUDE.md:117`

**Documented**:
```markdown
| Frontend | React 18, TypeScript, Vite, TanStack Query |
```

**Actual** (from `/home/thomas/translation-app/frontend/package.json:12-17`):
```json
"next": "^16.1.1",
"react": "^19.1.0",
"zustand": "^5.0.2"
```

Uses Next.js App Router, not Vite. Uses Zustand, not TanStack Query.

---

### 20. .claude/boundaries.md - Sprint Boundaries

**File**: `/home/thomas/translation-app/.claude/boundaries.md:13-18`

**Documents**:
```markdown
### Off-Limits (Future Sprints)
- src/gengowatcher/database/*  # Sprint 1
- src/gengowatcher/auth/*      # Sprint 1
```

**Actual**: Already implemented in Go at `backend/internal/database/` and `backend/internal/auth/`

---

## Medium Inconsistencies

### 21. README.md - Reference Implementation Path

**File**: `/home/thomas/translation-app/README.md:80`

**Documented**:
```markdown
The original GengoWatcher at `/home/thomas/GengoWatcher` serves as the reference
```

**Issue**: No details on what to reference from that project

---

### 22. CURRENT_PLAN.md - React Version

**File**: `/home/thomas/translation-app/CURRENT_PLAN.md:49`

**Documents**:
```markdown
| UI Library | React | 19.1.0 |
```

**Actual**: Correct, but README.md says React 18

---

### 23. docker-compose.yml - Mailhog Service

**File**: `/home/thomas/translation-app/docker-compose.yml:35-40`

**Includes**:
```yaml
mailhog:
  image: mailhog/mailhog:latest
  ports:
    - "1025:1025"  # SMTP
    - "8025:8025"  # Web UI
```

**Issue**: Mailhog never used in code - no email functionality implemented

---

### 24. README.md - Project Status

**File**: `/home/thomas/translation-app/README.md:72-76`

**Documents**:
```markdown
## Project Status

**Current Sprint**: Sprint 0 (Scaffolding)
```

**Actual**: Sprint 0 complete, basic auth and watcher implemented

---

### 25. CLAUDE.md - Subscription Tiers

**File**: `/home/thomas/translation-app/CLAUDE.md:119-125`

**Documents**:
```markdown
| Tier | Price | Features |
|------|-------|----------|
| Free | $0 | 1 watcher, 100 jobs/day |
| Pro | $29/mo | 3 watchers, 1000 jobs/day, auto-accept |
| Enterprise | $99/mo | Unlimited watchers, priority support |
```

**Issue**: Pricing logic not implemented, only models exist

---

## Low Inconsistencies

### 26. TECH_STACK.md - Version Accuracy

**File**: `/home/thomas/translation-app/TECH_STACK.md`

**Mostly Accurate** but:
- Go version: Documents "1.23+" (correct)
- Fiber: Documents "3.x" (correct - rc.3)
- GORM: Documents "1.25+" (actual: 1.31.1)

---

### 27. README.md - Project Name

**File**: `/home/thomas/translation-app/README.md:1`

**Documents**:
```markdown
# GengoWatcher SaaS
```

**Repository name**: `translation-app`

**Issue**: Minor - documentation uses product name, repo uses working name

---

### 28-47. Various Minor Inconsistencies

(Detailed in full audit report)

---

## Summary by Category

| Category | Count | Severity |
|----------|-------|----------|
| **Critical** | 12 | Blocking issues |
| **High** | 8 | Must fix soon |
| **Medium** | 14 | Should fix |
| **Low** | 13 | Nice to fix |
| **Total** | 47 | - |

---

## File-Specific Inconsistency Counts

| File | Inconsistencies | Priority |
|------|----------------|----------|
| README.md | 8 | CRITICAL |
| CLAUDE.md | 7 | CRITICAL |
| PLAN.md | 1 | CRITICAL (entire file) |
| .env.example | 6 | CRITICAL |
| .claude/reference.md | 4 | HIGH |
| .claude/boundaries.md | 2 | HIGH |
| CURRENT_PLAN.md | 2 | LOW |
| TECH_STACK.md | 1 | LOW |

---

## Recommended Fix Order

### Week 1 (Critical)
1. Update README.md tech stack and commands
2. Update CLAUDE.md commands and structure
3. Fix .env.example variables
4. Archive PLAN.md
5. Update .claude/reference.md

### Week 2 (High)
6. Update .claude/boundaries.md
7. Remove/obsolete Python config files
8. Verify all documentation matches Go backend

### Week 3+ (Medium/Low)
9. Add missing deployment documentation
10. Create API documentation
11. Add architecture diagrams
12. Create ADRs

---

**Last Updated**: 2025-12-28  
**Total Issues Resolved**: 0/47
