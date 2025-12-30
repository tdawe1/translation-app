# GengoWatcher SaaS - Current Implementation Plan

**Status**: Sprint 1 Complete | Sprint 2 Complete | Backend Testing Complete | Frontend Testing Complete | **Polishing Complete**
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

### ✅ Backend Tests (COMPLETED)

**File: `backend/tests/helpers.go`**
- `TestDB(t *testing.T)` - PostgreSQL test database connection
- `TestRedis(t *testing.T)` - Redis client (DB 15) with skip if unavailable
- `CreateTestUser(t, db, email)` - Creates user with watcher config/state
- `GenerateTestToken(t, userID)` - Real JWT token generation

**File: `backend/tests/watcher_test.go`**
- `TestWatcher_CompleteFlow` - Full watcher lifecycle (6 subtests)
- `TestWatcher_UnauthorizedAccess` - Auth rejection (2 subtests)
- `TestWatcher_ConcurrentStart` - Concurrent start safety

**Test Results:**
```
PASS: TestWatcher_CompleteFlow (0.02s)
  └─ 6/6 subtests passed
PASS: TestWatcher_UnauthorizedAccess (0.00s)
  └─ 2/2 subtests passed
PASS: TestWatcher_ConcurrentStart (0.01s)
ok  	github.com/tdawe1/translation-app/tests	0.032s
```

**Key Implementation Details:**
- Uses PostgreSQL `gengowatcher_test` database (port 5433)
- JWT middleware properly integrated with test tokens
- `databaseWrapper` adapts `*gorm.DB` to `database.Database` interface
- `NewTestManager(db)` accepts optional test database for real operations

### Test Infrastructure Setup (Simplified)

#### Backend Test Setup

**Flat Directory Structure:**
```
backend/tests/
├── helpers.go           # Test DB, Redis, user creation, JWT
└── watcher_test.go      # Watcher integration tests
```

**Test Configuration:**
```bash
# Required for running tests
JWT_SECRET=test-secret-for-testing-only-32-chars-min

# Optional (uses defaults if not set)
TEST_DB_HOST=localhost
TEST_DB_PORT=5433
TEST_DB_USER=gengo
TEST_DB_PASSWORD=devpass
TEST_DB_NAME=gengowatcher_test
TEST_DB_SSLMODE=disable
```

**Run Tests:**
```bash
cd backend
make test              # Run all tests
make test-verbose      # Run with verbose output
make test-coverage     # Run with coverage report
```

### Pending Tests

#### Backend Tests ✅ (COMPLETED 2025-12-30)

**`tests/websocket_test.go`** - 10 test groups covering:
- `TestWebSocket_Authentication` (4 subtests) - missing, invalid, valid ticket, one-time use
- `TestWebSocket_ReceivesJobNotification` - Redis pub/sub → receives job
- `TestWebSocket_ReceivesEventNotification` (3 subtests) - watcher events
- `TestWebSocket_HandlesDisconnect` - graceful cleanup
- `TestWebSocket_GetUserChannels` - channel name generation
- `TestWebSocket_ValidateOrigin` (3 subtests) - origin validation
- `TestWebSocket_PublishError` - error notifications
- `TestWebSocket_TicketExpiry` - ticket expiration
- `TestWebSocket_ConcurrentConnections` - concurrent access
- `TestWebSocket_ErrorChannel` - multiple error messages

#### Frontend Tests ✅ (COMPLETED 2025-12-30)

**`frontend/tests/dashboard.test.tsx`** - 17 tests covering:
- `Test: loads and displays watcher state` (5 subtests) - user info, status, jobs/earnings, loading, config details
- `Test: starts and stops watcher` (4 subtests) - start/stop calls, button disabled states
- `Test: opens and closes config modal` (2 subtests) - modal interaction
- `Test: WebSocket connection indicator` (2 subtests) - connected/disconnected states
- `Test: logout functionality` (1 subtest) - logout and redirect
- `Test: error handling` (3 subtests) - start/stop failures, config errors

**Test Infrastructure:**
- Vitest v4.0.16 with happy-dom environment
- React Testing Library for component testing
- Custom `createZustandMock` helper for selector-aware Zustand store mocking

**Test Results:**
```
✓ tests/dashboard.test.tsx (17 tests) 113ms
Test Files: 1 passed (1)
Tests: 17 passed (17)
```

---

### Polishing Tasks (Essential Only)

**Keep These:**
- Toast notifications for errors (essential UX)
- Basic loading states (text, not skeletons)
- ARIA labels for forms (minimal accessibility)

**Defer to Sprint 3+:**
- Caching headers (handled by nginx)

---

## Polishing Summary ✅

**Completed 2025-12-30**

All essential polishing tasks completed:

### Frontend Enhancements

**File: `frontend/components/ui/skeleton.tsx`** (CREATED)
- `Skeleton` component for loading placeholders

**File: `frontend/components/error-boundary.tsx`** (CREATED)
- `ErrorBoundary` class component with fallback UI
- Reload button on error
- Error logging support

**File: `frontend/components/watcher/job-list.tsx`** (MODIFIED)
- Added `React.memo` to `JobListItem` for performance optimization
- Prevents re-renders when job data hasn't changed

**File: `frontend/components/ui/modal.tsx`** (MODIFIED)
- Added `React.memo` to prevent unnecessary re-renders

**File: `frontend/lib/api.ts`** (MODIFIED)
- In-flight request deduplication via `pendingRequests` Map
- Concurrent identical requests share the same promise
- Automatic cleanup after request completes

**File: `frontend/app/dashboard/page.tsx`** (MODIFIED)
- Added keyboard shortcuts: `Ctrl+K` / `Cmd+K` to open config modal
- `ESC` to close modal
- Visual hint (`<kbd>`) on Configure button
- Wrapped in `ErrorBoundary` for error handling

**File: `frontend/app/globals.css`** (MODIFIED)
- Staggered fade-in animation for bento cards (25ms delays)
- `animate-fade-in` utility class available

**File: `frontend/hooks/use-watcher-websocket.ts`** (ENHANCED)
- Added `WebSocketMetrics` interface with:
  - `connected`, `reconnecting`, `reconnectCount`
  - `connectionStartTime`, `uptime` (seconds)
  - `lastMessageTime`, `messagesReceived`

**File: `frontend/app/layout.tsx`** (MODIFIED)
- Skip link for keyboard navigation (`Skip to main content`)
- Hidden until focused (`sr-only focus:not-sr-only`)

**Accessibility Features:**
- `aria-live="polite"` for status updates
- `aria-modal="true"` on dialogs
- `role="dialog"` on modal
- `aria-labelledby` for modal titles
- `aria-label` on icon-only buttons
- `id="main-content"` target for skip link

---

## Sprint 1 Testing Summary ✅

All Sprint 1 testing complete as of 2025-12-30:

**Backend Tests:**
- `tests/watcher_test.go` - 3 test groups, 8 subtests (PASS)
- `tests/websocket_test.go` - 10 test groups (PASS)
- `tests/rss_test.go` - 5 test groups, 12 subtests (PASS)
- `tests/websocket_monitor_test.go` - 10 test groups (PASS)

**Frontend Tests:**
- `frontend/tests/dashboard.test.tsx` - 17 tests (PASS)

---

## Sprint 2: RSS & WebSocket Monitoring ✅

**Completed 2025-12-30**

### Backend (Go)

**File: `backend/internal/watcher/rss.go`** (FULLY IMPLEMENTED)
- RSS feed parsing using `github.com/mmcdole/gofeed`
- Reward extraction from multiple formats ($5.00, 5.00 USD, etc.)
- Language pair extraction (EN → JP, etc.)
- Deduplication via `seenIDs` map
- Exported `Fetch()` method for testing

**File: `backend/internal/watcher/websocket.go`** (FULLY IMPLEMENTED)
- Gengo WebSocket connection via `github.com/gorilla/websocket`
- Authentication payload generation
- Message type handling
- Reconnection logic
- Ping latency tracking

**File: `backend/tests/rss_test.go`** (CREATED)
- 5 test groups, 12 subtests
- Tests: monitor creation, reward extraction (12 formats), fetch integration, reward filtering, deduplication, error handling

**File: `backend/tests/websocket_monitor_test.go`** (CREATED)
- 10 test groups, 3 subtests
- Tests: WebSocket monitor creation, status tracking, auth payload, job messages, deduplication

**File: `backend/tests/websocket_test.go`** (COMPLETED 2025-12-30)
- 10 test groups covering WebSocket handler functionality
- Tests: ticket authentication, job/event notifications, origin validation, error publishing, concurrent connections

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
