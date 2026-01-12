# GengoWatcher SaaS - Agent Development Guide

**Multi-tenant job monitoring SaaS with per-user watcher instances.**

## Quick Reference

```bash
./scripts/dev.sh up          # Start full dev stack
./scripts/dev.sh down        # Stop everything
./scripts/dev.sh status      # Check service status
./scripts/dev.sh logs        # Tail all logs

cd backend && make test      # Run Go tests
cd frontend && npm run test  # Run frontend tests
```

---

## Overview

Transforming GengoWatcher from a localhost-only tool to a remotely-hosted multi-tenant SaaS:
- **Per-User Watchers**: Isolated RSS + WebSocket monitoring per user
- **Multi-Method Auth**: Email/password, magic links, OAuth (Google/GitHub), API keys
- **Subscription Billing**: LemonSqueezy integration
- **Real-Time Notifications**: Redis pub/sub for instant job alerts

**Reference Implementation**: `/home/thomas/GengoWatcher` - Domain knowledge for RSS/WebSocket parsing

---

## Tech Stack

| Layer | Technology | Notes |
|-------|------------|-------|
| **Backend** | Go 1.25, Fiber v2, GORM | Fast HTTP framework, excellent stdlib |
| **Database** | PostgreSQL | GORM auto-migration (no Alembic) |
| **Auth** | golang-jwt/jwt, bcrypt, httpOnly cookies | 15min access, 7d refresh |
| **OAuth** | Google, GitHub | Custom handler in `internal/handlers/oauth.go` |
| **Real-time** | Redis pub/sub, WebSocket | User-isolated channels |
| **Email** | Resend | Transactional email service |
| **Billing** | LemonSqueezy | Webhook handler in `internal/handlers/lemonsqueezy.go` |
| **Frontend** | Next.js 16, React 19, Zustand | Custom fetch client in `lib/api.ts` |
| **Testing** | Go testing, Vitest | Backend: `make test`, Frontend: `npm run test` |

---

## Project Structure

```
translation-app/
├── backend/                    # Go backend service
│   ├── cmd/
│   │   ├── server/            # Application entry point
│   │   ├── admin_seed/        # Admin seeding CLI tool
│   │   └── translation-worker/ # Python translation service (docs/)
│   ├── internal/              # Private application code
│   │   ├── auth/              # JWT, password hashing, user service
│   │   ├── config/            # Environment-based configuration
│   │   ├── database/          # GORM connection, pooling
│   │   ├── email/             # Resend email service
│   │   ├── errors/            # Error definitions
│   │   ├── handlers/          # HTTP request handlers (routes)
│   │   ├── middleware/        # Fiber middleware (JWT, CORS, etc.)
│   │   ├── models/            # GORM models (User, Watcher, Subscription, etc.)
│   │   ├── oauth/             # OAuth provider logic
│   │   ├── password/          # Password hashing utilities
│   │   ├── seeds/             # Admin seeding for development/testing
│   │   ├── service/           # Token service (verification, magic link, reset)
│   │   └── watcher/           # RSS/WebSocket monitoring logic
│   ├── tests/                 # Backend integration tests
│   ├── Makefile               # Test commands
│   ├── go.mod                 # Go dependencies
│   └── main.go                # Server entry point with dependency injection
│
├── frontend/                   # Next.js 16 frontend
│   ├── app/                   # Next.js App Router (pages, layouts)
│   ├── components/            # React components
│   ├── lib/                   # Utilities, API client
│   ├── store/                 # Zustand state stores
│   ├── hooks/                 # Custom React hooks
│   └── tests/                 # Vitest tests
│
├── scripts/                    # Development automation
│   ├── dev.sh                 # Main dev environment controller
│   ├── functions/             # Modular bash functions
│   └── _lib.sh                # Shared bash library
│
├── docker-compose.yml          # PostgreSQL, Redis, MailHog
├── docs/.agents/              # Agent artifacts (plans, reports, todos)
└── CLAUDE.md                  # This file
```

---

## Development Workflow

### Primary Development Script

**Use `./scripts/dev.sh` for all development operations.**

```bash
./scripts/dev.sh up          # Start: docker → backend → frontend
./scripts/dev.sh down        # Stop: frontend → backend → docker
./scripts/dev.sh restart     # Restart all services
./scripts/dev.sh status      # Show status of all services
./scripts/dev.sh logs        # Tail logs from all services

# Individual service control
./scripts/dev.sh backend start|stop|restart|status|logs
./scripts/dev.sh frontend start|stop|restart|status|logs
./scripts/dev.sh docker start|stop|restart|status|logs

# Log management
./scripts/dev.sh logs list    # List all log files
./scripts/dev.sh logs clear   # Clear all log files

# Environment validation
./scripts/dev.sh check        # Validate dev environment setup
```

### Backend (Go)

```bash
cd backend

# Run server (hot reload via 'air' if installed, or manual restart)
go run ./cmd/server

# Run tests
make test              # Run all tests
make test-verbose      # Verbose test output
make test-coverage     # Generate coverage.html

# Build
go build -o server ./cmd/server

# Type checking
go vet ./...
```

**Test Environment Variables**:
- `JWT_SECRET` - Required for tests (use `test-secret-for-testing-only-32-chars-min` in Makefile)

### Frontend (Next.js)

```bash
cd frontend

# Development server (Turbopack)
npm run dev

# Production build
npm run build
npm run start

# Testing
npm run test            # Run Vitest tests
npm run test:ui         # Vitest UI
npm run test:coverage   # Coverage report

# Linting
npm run lint
```

### Admin Seeding (Development & Testing)

The admin seeding system creates or updates admin users atomically with proper JWT tokens.

**Via CLI Tool**:
```bash
cd backend
go run ./cmd/admin_seed -email admin@example.com -password AdminPass123!
# Output: User ID, email, role, and JWT token (valid 15 minutes)
```

**Via HTTP Endpoint** (dev-only):
```bash
# Only available when ENV=development
POST /dev/seed-admin
Content-Type: application/json

{
  "email": "admin@example.com",
  "password": "SecurePass123!"
}

# Response: { "user_id": "...", "email": "...", "role": "admin", "token": "..." }
```

**In Tests**:
```go
import "github.com/tdawe1/translation-app/internal/seeds"

// Creates admin user or returns existing if email matches
adminSeeder := seeds.NewAdminSeeder(db, tokenSvc)
user, token, err := adminSeeder.EnsureAdminUser("admin@test.com", "Pass123!")
```

The `AdminSeeder` atomically creates:
- `User` with `role=admin`
- `WatcherConfig` (default settings)
- `WatcherState` (stopped)

### Docker Services

```bash
# Start infrastructure only
docker-compose up -d

# Services exposed:
# - PostgreSQL: localhost:5433
# - Redis: localhost:6380
# - MailHog UI: http://localhost:8025

# Check service health
docker-compose ps

# Stop and remove volumes
docker-compose down -v
```

---

## Architecture Conventions

### Backend API

**Route Pattern**: `/api/v1/*` (e.g., `/api/v1/auth/register`)

**Error Response Format**:
```json
{
  "error": "Human-readable message",
  "code": "ERROR_CODE",
  "details": {}  // Optional additional context
}
```

**Async/Await**: Go uses goroutines for concurrency. Database operations use GORM's synchronous API (connection pooling handled internally).

**Testing Structure**: Tests mirror `internal/` structure under `tests/` directory.

### User Isolation (Multi-Tenancy)

- All database queries **must** be filtered by `user_id`
- Redis keys use pattern: `user:{user_id}:*`
- WebSocket rooms: `user:{user_id}:ws`
- Watcher instances are per-user, managed by `UserWatcherManager`

### Authentication Flow

1. **Email/Password**: POST `/api/v1/auth/register` → returns JWT + httpOnly refresh cookie
2. **Magic Link**: POST `/api/v1/auth/magic-link/send` → email with token → POST `/api/v1/auth/magic-link/verify`
3. **OAuth**: GET `/api/v1/oauth/{provider}` → redirect → callback with code → JWT issuance
4. **API Keys**: Header `X-API-Key: {key}` → user lookup via `models.APIKey`

### Frontend State Management

- **Global State**: Zustand stores in `frontend/store/`
- **API Client**: Custom fetch client in `lib/api.ts` with request deduplication
- **Auth Tokens**: Stored in sessionStorage (access_token, refresh_token)

---

## Go Code Patterns

### Error Handling

**Use typed errors from `internal/errors`** - never return raw strings as errors.

```go
import apperrors "github.com/tdawe1/translation-app/internal/errors"

// Create a typed error
err := apperrors.New(apperrors.ErrUserNotFound, "user not found")

// Add details for debugging
err.WithDetails(map[string]interface{}{"user_id": userID})

// In handlers, use response helpers
handlers.RespondWithError(c, 404, apperrors.ErrUserNotFound, "User not found")
handlers.RespondWithAPIError(c, 400, apiErr)
```

**Available error codes** (from `internal/errors/errors.go`):
- `INVALID_REQUEST`, `WEAK_PASSWORD`, `INVALID_USER_ID` (validation)
- `NOT_AUTHENTICATED`, `INVALID_CREDENTIALS`, `INACTIVE_USER`, `TOKEN_ERROR` (auth)
- `USER_EXISTS`, `USER_NOT_FOUND`, `CREATE_ERROR`, `UPDATE_ERROR` (user operations)
- `DATABASE_ERROR`, `INTERNAL_ERROR` (system)

### Context Usage

**For database operations**: GORM's context is handled internally. Pass context when using Redis:

```go
import "context"

// Redis operations require context
ctx := context.Background()
redisClient.Keys(ctx, "user:*").Result()
redisClient.Del(ctx, key)

// For request-scoped context, use Fiber's ctx.Context()
ctx := c.Context()
```

**For cancellation**: Create context with timeout when appropriate:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
result, err := redisClient.Get(ctx, key).Result()
```

### Handler Pattern

**Use Fiber's `*fiber.Ctx`** as the receiver for all handler methods. Helpers are in `internal/handlers/response.go`:

```go
func (h *MyHandler) DoThing(c *fiber.Ctx) error {
    // Parse user ID from path param
    userID, err := handlers.ParseUserID(c.Params("id"))
    if err != nil {
        return handlers.RespondWithError(c, 400, handlers.ErrInvalidUserID, "invalid user id")
    }

    // Success response
    return c.JSON(fiber.Map{"data": result})
}
```

### UUID Handling

**All IDs are UUIDs** (stored in DB as `uuid` type, sent as strings in JSON):

```go
import "github.com/google/uuid"

// Parse from string (e.g., URL param)
userID, err := uuid.Parse(c.Params("id"))
if err != nil {
    return handlers.RespondWithError(c, 400, handlers.ErrInvalidUserID, "invalid id")
}

// Convert to string for JSON response
return c.JSON(fiber.Map{"id": user.ID.String()})
```

### Memory Management Patterns

**Use LRU Cache for tracking seen items** - prevents unbounded memory growth:

```go
import "github.com/tdawe1/translation-app/internal/watcher"

// Create LRU cache with max size (evicts oldest when full)
cache := watcher.NewLRUCache(1000)

// Add returns true if already seen (deduplication)
if exists := cache.Add(jobID); exists {
    return nil // Skip duplicate
}

// For Redis sets, always set TTL to prevent unbounded growth
_ = redisClient.Expire(ctx, "seen_jobs:"+userID, 24*time.Hour).Err()
```

**When to use LRU vs Redis**:
- Use `LRUCache` for in-memory deduplication during watcher lifecycle (evicts old entries automatically)
- Use Redis with TTL for persistent seen-job tracking across restarts (24-hour expiration)

---

## Frontend Patterns

### Zustand Stores

**All stores follow this pattern** (`frontend/store/`):

```typescript
import { create } from "zustand";
import { persist } from "zustand/middleware";

interface MyStoreState {
  data: DataType | null;
  loading: boolean;
  error: string | null;

  // Actions
  fetchData: () => Promise<void>;
  clear: () => void;
}

export const useMyStore = create<MyStoreState>()(
  persist(
    (set, get) => ({
      // Initial state
      data: null,
      loading: false,
      error: null,

      // Actions
      fetchData: async () => {
        set({ loading: true, error: null });
        try {
          const data = await apiClient.getData();
          set({ data, loading: false });
        } catch (error) {
          set({ error: getMessage(error), loading: false });
        }
      },

      clear: () => set({ data: null, error: null, loading: false }),
    }),
    {
      name: "gengowatcher-mystore",
      partialize: (state) => ({ data: state.data }), // Only persist specific fields
    }
  )
);
```

**Existing stores**:
- `useAuthStore` - User authentication state, persists user + isAuthenticated
- `useWatcherStore` - Watcher config/state, no persistence (fetches from API)
- `useToastStore` - Toast notifications, ephemeral
- `useJobsStore` - Jobs list, ephemeral

### Next.js App Router Structure

**File-based routing** in `frontend/app/`:

```
app/
├── layout.tsx           # Root layout (providers, global styles)
├── page.tsx             # Home page (/)
├── globals.css          # Global styles
├── auth/                # Auth routes
│   ├── login/
│   │   └── page.tsx     # /auth/login
│   └── register/
│       └── page.tsx     # /auth/register
├── dashboard/           # Protected routes
│   └── page.tsx         # /dashboard
└── settings/            # Settings pages
    └── page.tsx         # /settings
```

**Route conventions**:
- Public routes: `/auth/*`, `/`
- Protected routes: `/dashboard/*`, `/settings/*` (wrap components with auth check)
- Use `<Link>` from `next/link` for navigation

### API Client

**All API calls through `frontend/lib/api.ts`**:

```typescript
import { authApi, watcherApi } from "@/lib/api";

// Auth endpoints
await authApi.register(email, password)
await authApi.login(email, password)
await authApi.me()  // Get current user (returns null if not authenticated)

// Watcher endpoints
await watcherApi.getConfig()
await watcherApi.updateConfig({ enabled: true })
await watcherApi.getState()
await watcherApi.start()
await watcherApi.stop()
```

---

## Design Language (Data Factory)

**Note**: This section describes the intended visual system for the frontend.

- **Fonts**: IBM Plex Sans (headings), IBM Plex Mono (labels)
- **Cards**: Bento style, 1px border, sharp corners
- **Hover**: Precision focus (border color change, NO shadow lift)
- **Accents**: ROYGBIV spectrum for headings ONLY
- **Spacing**: Generous (py-24 to pt-44 sections)

---

## Database Models (GORM)

All models inherit from `Base` (ID, CreatedAt, UpdatedAt) with UUID primary keys:

| Model | Purpose | Key Fields |
|-------|---------|------------|
| `User` | User accounts | Email, PasswordHash, Role, Provider |
| `OAuthAccount` | Linked OAuth providers | Provider, ProviderID |
| `APIKey` | API authentication | KeyHash, LastUsed |
| `RefreshToken` | JWT refresh tokens | TokenHash, ExpiresAt |
| `WatcherConfig` | Per-user watcher settings | URL, Filters, Enabled |
| `WatcherState` | Runtime watcher state | LastCheck, Status |
| `SubscriptionPlan` | Available tiers | LemonSqueezy variant IDs |
| `Subscription` | User subscriptions | Status, CurrentPeriodEnd |
| `BillingEvent` | Webhook events | EventType, Processed |
| `AuditLog` | Security audit trail | Action, IPAddress |
| `EmailVerificationToken` | Email verification | TokenHash, ExpiresAt |
| `MagicLinkToken` | Magic link auth | TokenHash, ExpiresAt |
| `PasswordResetToken` | Password reset | TokenHash, ExpiresAt |

**Migrations**: Handled via GORM AutoMigrate in `main.go`. No Alembic.

---

## Environment Variables

### Required

```bash
JWT_SECRET=                    # JWT signing (32+ chars REQUIRED, server fails without it)
DATABASE_URL=                  # Postgres connection string
REDIS_URL=                     # Redis connection string
```

**IMPORTANT**: `JWT_SECRET` must be set in all non-test environments. The application will fail to start at startup if `JWT_SECRET` is missing or too short. Generate one with:
```bash
openssl rand -hex 32  # Produces 64-character hex string (recommended)
```

### Optional (with defaults)

```bash
# Backend
PORT=8000                      # Fiber server port
COOKIE_SECURE=false            # Set true in production

# Database Connection Pool
DB_MAX_OPEN_CONNS=25           # Maximum open connections (default: 25)
DB_MAX_IDLE_CONNS=10           # Maximum idle connections (default: 10)
DB_CONN_MAX_LIFETIME=1h        # Connection lifetime before refresh (default: 1h)
DB_CONN_MAX_IDLE_TIME=10m      # Idle time before connection closed (default: 10m)

# Pool Tuning Guidelines:
# - Small deployment (1-2 cores): DB_MAX_OPEN_CONNS=15
# - Medium deployment (4-8 cores): DB_MAX_OPEN_CONNS=50
# - Large deployment (16+ cores): DB_MAX_OPEN_CONNS=100
# - Set DB_MAX_IDLE_CONNS to ~40% of DB_MAX_OPEN_CONNS

# Email (Resend)
RESEND_API_KEY=
FROM_EMAIL=
FROM_NAME=

# OAuth
OAUTH_REDIRECT_URL=            # Frontend URL for callbacks
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=

# Billing (LemonSqueezy)
LEMONSQUEEZY_WEBHOOK_SECRET=
LEMONSQUEEZY_API_KEY=
LEMONSQUEEZY_STORE_ID=

# Testing
JWT_SECRET=test-secret-for-testing-only-32-chars-min  # For tests
```

---

## Subscription Tiers

| Tier | Price | Features |
|------|-------|----------|
| Free | $0 | 1 watcher, 100 jobs/day |
| Pro | $29/mo | 3 watchers, 1000 jobs/day, auto-accept |
| Enterprise | $99/mo | Unlimited watchers, priority support |

**Implementation**: LemonSqueezy webhooks update `Subscription` model; feature enforcement in watcher logic.

---

## Translation Worker

The translation worker (`backend/cmd/translation-worker/`) is a Python-based service for document translation using multi-provider LLM support.

### Quick Start

```bash
cd backend/cmd/translation-worker

# Install dependencies
pip install anthropic openai requests redis click

# Configure (create config.toml)
cat > config.toml << EOF
[worker]
id = "worker-1"
max_concurrent = 3

[translation]
default_provider = "anthropic"
default_model = "claude-sonnet-4-5-20250929"

[cache.redis]
host = "localhost"
port = 6379

[job_queue]
enabled = true
backend = "redis"
EOF

# Set API key
export ANTHROPIC_API_KEY="sk-ant-..."

# Run worker
python main.py
```

### CLI Usage

```bash
# Translate text
python -m review translate "こんにちは" --provider anthropic

# Batch process
python -m review batch -i sources.txt -o translations.txt -p anthropic

# Compare translations
python -m review judge original.txt translation_a.txt translation_b.txt
```

### Documentation

- **README**: `backend/cmd/translation-worker/README.md` - Overview and setup
- **CLI Reference**: `backend/cmd/translation-worker/docs/CLI_REFERENCE.md` - Complete CLI guide
- **API Reference**: `backend/cmd/translation-worker/docs/API_REFERENCE.md` - Job queue API
- **LLM Providers**: `backend/cmd/translation-worker/docs/LLM_PROVIDERS.md` - Provider configuration
- **Troubleshooting**: `backend/cmd/translation-worker/docs/TROUBLESHOOTING.md` - Common issues

### Architecture

- **Job Queue**: Redis-backed with priority support (urgent/normal/bulk)
- **Fault Tolerance**: Checkpoint/resume for long-running jobs
- **Parsers**: DOCX, PPTX, PDF, XLSX with layout preservation
- **Glossary**: Term consistency across translations
- **Cache**: Avoid re-translating identical content

---

## Agent Artifacts (docs/.agents/)

The `docs/.agents/` directory contains files **primarily intended for AI agents**:

- **plans/** - Implementation plans, technical specifications, task breakdowns
- **reports/** - Code analysis reports, audits, technical assessments
- **todos/** - TODO lists, task tracking files, backlog items

**Rule of thumb**: If a file is meant to be read primarily by an AI agent for context, planning, or execution, it belongs in `docs/.agents/`. Human-facing documentation stays in `docs/` (outside the `.agents/` subdirectory).

---

## Key Implementation Notes for Agents

### When Adding New API Endpoints

1. Create handler in `backend/internal/handlers/`
2. Register routes in `backend/cmd/server/main.go`
3. Add corresponding types in `backend/internal/models/` if needed
4. Mirror tests in `backend/tests/handlers/`
5. Add frontend client functions in `frontend/lib/api.ts`

### When Modifying Database Schema

1. Update model in `backend/internal/models/`
2. Add to `AutoMigrate` call in `backend/cmd/server/main.go`
3. **Note**: GORM auto-migrate is used; no manual migration files

### When Working with WebSockets

- Use `UserWatcherManager` from `internal/watcher/`
- Each user gets isolated Redis pub/sub channel
- Connection lifecycle managed by `WebSocketHandler`

### Security Considerations

- **Account Enumeration**: Return generic errors to prevent email harvesting
- **JWT Secret**: Must be 32+ characters; server fails to start without `JWT_SECRET` in non-test environments
- **httpOnly Cookies**: Used for refresh tokens; XSS protection
- **CORS**: Configured via Fiber middleware
- **Rate Limiting**: Not yet implemented (planned)
- **SSRF Protection**: RSS feed URLs are validated before fetching to prevent Server-Side Request Forgery
  - Blocks localhost, private IP ranges (RFC 1918), and invalid schemes
  - Only HTTP/HTTPS protocols allowed
  - DNS resolution validates actual IP addresses (prevents DNS rebinding)
  - See `internal/watcher/url_validator.go` for implementation
- **Memory Management**: All in-memory caches use LRU eviction; Redis sets have TTL to prevent exhaustion

### Production Security Checklist

Before deploying to production, ensure:

- [ ] `JWT_SECRET` is set to cryptographically random value (32+ chars) - generate with `openssl rand -hex 32`
- [ ] `JWT_SECRET` is not empty in environment (server will fail to start if missing)
- [ ] Docker resource limits are configured in `docker-compose.production.yml`
- [ ] Database credentials are strong and not default values
- [ ] HTTPS/TLS is enabled (set `COOKIE_SECURE=true` in production)
- [ ] CORS `ALLOWED_ORIGINS` only includes production domains

---

## Deployment

### Docker Backend

**Production build** uses multi-stage Dockerfile (`backend/Dockerfile`):

```dockerfile
# Stage 1: Build
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 2: Runtime
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8000
CMD ["./server"]
```

**Build and run**:
```bash
cd backend
docker build -t gengowatcher-backend .
docker run -p 8000:8000 --env-file .env gengowatcher-backend
```

### Production Environment

**Required production variables** (see `.env.production.example`):

```bash
# Database (PostgreSQL, not SQLite)
DATABASE_URL=postgres://user:pass@host:5432/dbname

# Redis
REDIS_URL=redis://user:pass@host:6379/0

# Security (never use defaults)
JWT_SECRET=             # Generate with: openssl rand -hex 32
COOKIE_SECURE=true     # Required for HTTPS

# Email
RESEND_API_KEY=

# OAuth callbacks
OAUTH_REDIRECT_URL=https://yourdomain.com

# Billing
LEMONSQUEEZY_WEBHOOK_SECRET=  # From LemonSqueezy webhook settings
```

### Health Checks

**Backend endpoints**:
- `GET /healthz` - Liveness probe (always returns 200 if running)
- `GET /readyz` - Readiness probe (checks DB and Redis connections)

Configure these in your orchestrator (Kubernetes, Docker Compose, etc.).

### Docker Compose Production

**Production compose file**: `docker-compose.production.yml`

```bash
docker-compose -f docker-compose.production.yml up -d
```

Includes:
- PostgreSQL with health check
- Redis with persistence
- Backend service with proper environment passing

---

## Testing Patterns

### Backend (Go)

**Test file location**: Mirror `internal/` structure under `tests/` directory.

**Test helpers** are in `tests/helpers.go`:
- `RequireDB(t)` - Get test database (PostgreSQL on localhost:5433)
- `RequireRedis(t)` - Get test Redis client (DB 15)
- `CreateTestUser(t, db, email)` - Create a test user
- `CreateTestWatcher(t, db, userID)` - Create test watcher config

**Test naming convention**:
```go
func TestFeature_Behavior(t *testing.T) { ... }
func TestFeature_AtomicConsume(t *testing.T) { ... }
```

Use descriptive names that document what is being tested.

**Standard test pattern**:
```go
func TestMagicLink_AtomicConsume(t *testing.T) {
    db := RequireDB(t)
    redisClient := RequireRedis(t)
    require.NotNil(t, redisClient, "Redis required for magic link tests")

    // Create test config
    cfg := &config.Config{
        JWTSecret: "test-secret-for-testing-only-32-chars-min",
        // ... other config
    }

    // Create test app with Fiber
    app := fiber.New(fiber.Config{
        AppName:               "GengoWatcher Test",
        DisableStartupMessage: true,
    })

    // Register routes
    app.Post("/api/v1/auth/magic-link", authHandler.RequestMagicLink)

    t.Run("Subtest description", func(t *testing.T) {
        // Arrange
        reqBody := bytes.NewBufferString(`{"email":"test@example.com"}`)
        req := httptest.NewRequest("POST", "/api/v1/auth/magic-link", reqBody)
        req.Header.Set("Content-Type", "application/json")

        // Act
        resp, err := app.Test(req)
        require.NoError(t, err)

        // Assert
        assert.Equal(t, 200, resp.StatusCode)
    })
}
```

**Running specific tests**:
```bash
cd backend
go test ./tests/auth_test.go -v           # Single file
go test ./tests/... -run TestMagicLink     # By name
go test ./tests/... -v -run "Atomic.*"     # By pattern
```

### Frontend (Vitest)

**Test location**: Co-locate with components, or use `tests/` directory.

**Test file pattern**: `*.test.{ts,tsx}` (configured in `vitest.config.ts`)

**Setup file**: `vitest.setup.ts` - runs before each test file

**Basic test pattern**:
```typescript
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MyComponent } from '@/components/MyComponent';

describe('MyComponent', () => {
  it('renders user email when authenticated', () => {
    const user = { email: 'test@example.com' };
    render(<MyComponent user={user} />);
    expect(screen.getByText('test@example.com')).toBeInTheDocument();
  });
});
```

**Testing async actions** (Zustand stores):
```typescript
it('fetches and sets config', async () => {
  const store = useWatcherStore.getState();
  // Mock API call
  vi.mock('@/lib/api', () => ({
    watcherApi: { getConfig: vi.fn().mockResolvedValue({ enabled: true }) }
  }));

  await store.fetchConfig();
  expect(store.config).toEqual({ enabled: true });
});
```

**Coverage reports**:
```bash
cd frontend
npm run test:coverage    # Generates coverage/ directory
```

---
