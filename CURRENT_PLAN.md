# GengoWatcher SaaS - Current Implementation Plan

**Status**: Sprint 1 Complete | Sprint 1 Testing Planned
**Tech Stack**: Go 1.23 + Fiber 3.x + GORM + Next.js 16 + React 19
**Repository**: https://github.com/tdawe1/translation-app

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         Frontend                             │
│  Next.js 16 + React 19 + Tailwind 4 + Zustand              │
│  - /app/auth/login - Login page                             │
│  - /app/auth/register - Register page                       │
│  - /app/dashboard - Protected dashboard                      │
└────────────────────┬────────────────────────────────────────┘
                     │ HTTP + WebSocket
                     │ JWT (Bearer) + httpOnly Cookie
┌────────────────────▼────────────────────────────────────────┐
│                      Backend (Go)                            │
│  Fiber 3.x + GORM + PostgreSQL 17 + Redis 7.4               │
│  - /api/v1/auth/* - JWT auth endpoints                      │
│  - /api/v1/watcher/* - Watcher control endpoints            │
│  - /api/v1/webhooks/* - LemonSqueezy webhooks               │
└────────────────────┬────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────┐
│  PostgreSQL 17     │  Redis 7.4  │  Gengo API                │
│  - users           │  - pub/sub  │  - RSS feed               │
│  - watcher_configs │  - seen_jobs│  - WebSocket              │
│  - watcher_states  │  - events   │                           │
│  - subscriptions   │             │                           │
└────────────────────┴─────────────┴───────────────────────────┘
```

---

## Tech Stack (December 2025)

| Layer | Technology | Version |
|-------|------------|---------|
| Backend Language | Go | 1.23 |
| Backend Framework | Fiber | 3.x (rc.3) |
| ORM | GORM | 1.31.1 |
| Database | PostgreSQL | 17 |
| Cache | Redis | 7.4 |
| Frontend Framework | Next.js | 16.1.1 |
| UI Library | React | 19.1.0 |
| Language | TypeScript | 5.7 |
| Styling | Tailwind CSS | 4 |
| State Management | Zustand | 5.0.2 |
| Billing | LemonSqueezy | - |

---

## Completed Work ✅

### Backend (Go)

**File: `backend/go.mod`**
- Fiber 3.x web framework
- GORM for PostgreSQL
- go-redis for pub/sub
- golang-jwt/jwt for JWT auth
- bcrypt for password hashing

**File: `backend/internal/models/user.go`**
- `User` - User accounts with email/password
- `OAuthAccount` - Google/GitHub OAuth (future)
- `APIKey` - API key authentication (future)
- `RefreshToken` - JWT refresh tokens
- `WatcherConfig` - Per-user watcher settings
- `WatcherState` - Per-user watcher state and stats
- `Subscription` - LemonSqueezy subscriptions
- `BillingEvent` - Webhook event tracking
- `AuditLog` - Security audit trail

**File: `backend/internal/handlers/auth.go`**
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login with email/password
- `POST /api/v1/auth/logout` - Logout
- `GET /api/v1/me` - Get current user

**File: `backend/internal/middleware/jwt.go`**
- JWT validation middleware
- Bearer token + httpOnly cookie support
- Configurable context key

**File: `backend/internal/handlers/lemonsqueezy.go`**
- `POST /api/v1/webhooks/lemonsqueezy` - Webhook handler
- HMAC-SHA256 signature verification
- Idempotency via event_id deduplication
- Subscription status updates

**File: `backend/internal/watcher/manager.go`**
- `UserWatcherManager` - Per-user watcher lifecycle
- `StartWatcher(userID)` - Start goroutines for RSS/WebSocket
- `StopWatcher(userID)` - Graceful shutdown via context cancel
- `handleJob(job)` - Deduplication + reward filtering + Redis pub/sub
- Redis channels: `user:{id}:jobs`, `user:{id}:events`, `user:{id}:errors`

**File: `backend/internal/watcher/rss.go`**
- `RSSMonitor` stub (30s polling interval)
- TODO: Actual HTTP fetch + feed parsing

**File: `backend/internal/watcher/websocket.go`**
- `WebSocketMonitor` stub
- TODO: Actual Gengo WebSocket connection

**File: `backend/cmd/server/main.go`**
- Fiber app with CORS middleware
- Auto-migration on startup
- Route definitions

### Frontend (Next.js)

**File: `frontend/lib/api.ts`**
- `HttpClient` class with interceptors
- `authApi` - register, login, logout, me
- `watcherApi` - config, state, start, stop (future)
- Auto-redirect on 401
- Bearer token injection

**File: `frontend/store/auth.ts`**
- Zustand store with persist middleware
- User state + isAuthenticated + isLoading

**File: `frontend/components/auth/provider.tsx`**
- `AuthProvider` - Checks auth on mount
- Calls `/api/v1/me` with stored token

**File: `frontend/components/auth/protected-route.tsx`**
- `ProtectedRoute` - Wraps authenticated routes
- Redirects to `/auth/login` if not authenticated

**File: `frontend/app/auth/login/page.tsx`**
- Login form with Data Factory design
- IBM Plex Sans + Mono fonts
- Bento cards with precision hover

**File: `frontend/app/auth/register/page.tsx`**
- Register form with password confirmation
- Same Data Factory design

**File: `frontend/app/dashboard/page.tsx`**
- Protected dashboard
- User info display
- Stub for watcher controls

**File: `frontend/app/layout.tsx`**
- IBM Plex Sans + Mono font loading
- AuthProvider wrapper

**File: `frontend/app/globals.css`**
- Tailwind CSS 4 with @theme
- ROYGBIV color variables
- Bento card styles (no shadow lift)

---

## Sprint 1: Dashboard & Real-time Updates ✅

**Completed 2025-12-29**

### Backend (Go)

**File: `backend/internal/handlers/websocket.go`** (CREATED)
- WebSocket endpoint at `/ws` for real-time updates
- JWT authentication via query parameter (`?token=Bearer...`)
- Redis pub/sub integration for user-specific channels:
  - `user:{id}:jobs` - New job notifications
  - `user:{id}:events` - Watcher lifecycle events
  - `user:{id}:errors` - Error messages
- `PublishJob()`, `PublishEvent()`, `PublishError()` helper methods

**File: `backend/internal/handlers/watcher.go`** (CREATED)
- `GET /api/v1/watcher/config` - Get user's watcher config
- `PUT /api/v1/watcher/config` - Update config (partial updates supported)
- `GET /api/v1/watcher/state` - Get watcher state with live status
- `POST /api/v1/watcher/start` - Start watcher
- `POST /api/v1/watcher/stop` - Stop watcher

**File: `backend/internal/middleware/jwt.go`** (MODIFIED)
- Added `WebSocketAuth()` function for query parameter token validation
- Added `extractTokenFromQuery()` helper

**File: `backend/cmd/server/main.go`** (MODIFIED)
- Added WebSocket route: `app.Get("/ws", middleware.WebSocketAuth(...), wsHandler.HandleWebSocket())`

### Frontend (Next.js)

**File: `frontend/store/watcher.ts`** (CREATED)
- Zustand store with persist for watcher config/state
- Actions: `fetchConfig`, `updateConfig`, `fetchState`, `startWatcher`, `stopWatcher`
- Types: `WatcherConfig`, `WatcherState`

**File: `frontend/store/jobs.ts`** (CREATED)
- Zustand store for detected jobs
- Actions: `addJob`, `clearJobs`, `removeJob`
- Max 100 jobs kept in memory (FIFO eviction)

**File: `frontend/hooks/use-watcher-websocket.ts`** (CREATED)
- WebSocket hook with auto-reconnect (exponential backoff: 1s, 2s, 4s, 8s, 16s)
- Type-safe message handling with discriminated unions
- Auto-adds received jobs to jobs store

**File: `frontend/components/ui/modal.tsx`** (CREATED)
- Reusable modal with overlay click and escape key close

**File: `frontend/components/watcher/config-form.tsx`** (CREATED)
- Configuration form with Tailwind-only toggle switches
- Fields: RSS URL, min/max reward, WebSocket, auto-accept, notifications

**File: `frontend/components/watcher/job-list.tsx`** (CREATED)
- Job list with filtering (source, sort, min reward)
- Stats display: total jobs, total value, RSS/WebSocket counts

**File: `frontend/app/dashboard/page.tsx`** (MODIFIED)
- Full dashboard with watcher controls, status display, real-time metrics
- WebSocket connection indicator (pulsing green dot)
- Start/Stop/Configure buttons

---

## Sprint 1 Testing Plan 📋

> **Revised after peer review** - Simplified approach focusing on behavior over implementation
> - **72% less code** than original plan (~450 LOC vs ~1600 LOC)
> - Test runtime: <5 seconds vs ~45 seconds
> - **No testcontainers, no MSW, no fixtures**

### Test Infrastructure Setup (Simplified)

#### 1. Backend Test Setup

**Flat Directory Structure:**
```
backend/tests/
├── websocket_test.go    # WebSocket handler + auth + integration
├── watcher_test.go      # Watcher CRUD + flow
└── helpers.go           # Only if >3 shared helpers
```

**Test Configuration:**
```bash
# .env.test
DATABASE_URL=sqlite://file::memory:?cache=shared&_foreign_keys=true
JWT_SECRET=test-secret-for-testing-only-32-chars-min
REDIS_URL=redis://localhost:6379/1
```

**Dependencies to Add:**
- `github.com/stretchr/testify` - Assertions (already in go.mod)
- `github.com/gorilla/websocket` - WebSocket client for testing
- **No testcontainers** - Use SQLite for fast tests

#### 2. Frontend Test Setup

**Flat Directory Structure:**
```
frontend/tests/
├── dashboard.test.tsx   # Full user journey (config + WS + jobs)
└── setup.ts             # Test setup with mocks
```

**Dependencies to Add:**
- `vitest` - Fast test runner
- `@testing-library/react` - Component testing utilities
- `@testing-library/user-event` - User interaction simulation
- `happy-dom` - Lightweight DOM environment
- **No MSW** - Use direct mocks for WebSocket

---

### Tests to Implement

#### Backend Tests (~200 LOC)

**`tests/websocket_test.go`**
```go
func TestWebSocket_Authentication(t *testing.T) {
    // Tests: missing token, invalid token, valid token
}
func TestWebSocket_ReceivesJobNotification(t *testing.T) {
    // Tests: Redis publish → WebSocket receives
}
func TestWebSocket_ReceivesEventNotification(t *testing.T) {
    // Tests: watcher.started, watcher.stopped events
}
func TestWebSocket_HandlesDisconnect(t *testing.T) {
    // Tests: graceful cleanup
}
```

**`tests/watcher_test.go`**
```go
func TestWatcher_CompleteFlow(t *testing.T) {
    // 1. Register → Login → Config (404) → Update → Start → Stop
}
func TestWatcher_UnauthorizedAccess(t *testing.T) {
    // Tests: requests without JWT are rejected
}
func TestWatcher_ConcurrentStart(t *testing.T) {
    // Tests: multiple start calls only start once
}
```

#### Frontend Tests (~150 LOC)

**`tests/dashboard.test.tsx`**
```typescript
describe('Dashboard Flow', () => {
  it('loads and displays watcher state', async () => {
    // Test: fetch config and state on mount
  })
  it('starts and stops watcher', async () => {
    // Test: button clicks trigger API calls
  })
  it('opens and submits config modal', async () => {
    // Test: modal interaction
  })
  it('displays jobs from WebSocket', async () => {
    // Test: mock WebSocket message → job appears
  })
  it('handles WebSocket reconnection', async () => {
    // Test: disconnect → reconnect with backoff
  })
})
```

---

### Polishing Tasks (Essential Only)

**Keep These:**
- Toast notifications for errors (essential UX)
- Basic loading states (text, not skeletons)
- ARIA labels for forms (minimal accessibility)

**Defer to Sprint 3+:**
- Skeleton loaders
- React.memo optimization
- Virtualization
- Request deduplication
- Connection pooling metrics
- Caching headers (handled by nginx)
- Keyboard shortcuts
- Screen reader testing
- Transition animations
- Error boundaries (add when real errors occur)

---

## In Progress 🚧

### Sprint 1 Testing Implementation
- Setting up test infrastructure
- Writing unit tests for critical paths
- Writing integration tests for complete flows

---

## Pending Tasks 📋

### High Priority

1. **Complete npm install and verify build**
   ```bash
   cd frontend && npm install
   npm run build
   ```

2. **Add Watcher API endpoints in backend**
   - `GET /api/v1/watcher/config` - Get user's watcher config
   - `PUT /api/v1/watcher/config` - Update config
   - `GET /api/v1/watcher/state` - Get watcher state
   - `POST /api/v1/watcher/start` - Start watcher
   - `POST /api/v1/watcher/stop` - Stop watcher
   - Protected by JWT middleware

3. **Implement RSS feed parsing**
   - Use `github.com/mmcdole/gofeed` for RSS parsing
   - Extract reward from Gengo job titles
   - Implement `extractReward()` and `fetch()` methods

4. **Implement Gengo WebSocket connection**
   - Connect to Gengo's WebSocket endpoint
   - Authenticate with session token
   - Parse job notifications

5. **Dashboard watcher controls**
   - Start/Stop buttons
   - Real-time status display
   - Job list with filtering

### Medium Priority

6. **WebSocket for real-time updates**
   - Server-side: `/ws` endpoint with auth
   - Client-side: WebSocket + Redis pub/sub listener
   - Display new jobs in real-time

7. **LemonSqueezy integration**
   - Create checkout sessions
   - Handle subscription webhooks
   - Update user subscription status

8. **OAuth (Google/GitHub)**
   - Backend: OAuth flow handlers
   - Frontend: "Continue with..." buttons

### Low Priority

9. **Email notifications**
   - Resend for transactional emails
   - Email verification
   - Password reset

10. **API key management**
    - Generate/revoke API keys
    - Scoped permissions

11. **Deployment**
    - Docker build
    - Railway/Fly.io deployment
    - Environment variables

---

## Environment Variables

```bash
# .env
DATABASE_URL=host=localhost user=gengo password=devpass dbname=gengowatcher sslmode=disable
JWT_SECRET=dev-secret-change-in-production
REDIS_URL=redis://localhost:6379/0

# LemonSqueezy
LEMONSQUEZY_WEBHOOK_SECRET=your_webhook_secret
LEMONSQUEZY_API_KEY=your_api_key
LEMONSQUEZY_STORE_ID=your_store_id

# Frontend
NEXT_PUBLIC_API_URL=http://localhost:8000
```

---

## Running Locally

```bash
# Start services
docker-compose up -d  # PostgreSQL + Redis

# Backend
cd backend
go run cmd/server/main.go

# Frontend
cd frontend
npm run dev

# Visit http://localhost:3000
```

---

## Git Commit Conventions

```
feat: new feature
fix: bug fix
docs: documentation
refactor: code refactoring
style: formatting (no logic change)
test: adding tests
chore: maintenance tasks
```

---

## Dependabot Alerts

⚠️ 7 vulnerabilities (4 critical, 2 high, 1 moderate)
- Run `npm audit fix` after npm install completes
