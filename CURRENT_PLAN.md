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

### Test Infrastructure Setup

#### 1. Backend Test Setup

**Directory Structure:**
```
backend/tests/
├── handlers/
│   ├── websocket_test.go
│   ├── watcher_test.go
│   └── auth_test.go
├── middleware/
│   └── jwt_test.go
├── integration/
│   ├── watcher_flow_test.go
│   └── websocket_flow_test.go
├── testutil/
│   └── setup.go
└── fixtures/
    └── sample_data.go
```

**Test Configuration:**
```bash
# .env.test
DATABASE_URL=host=localhost user=gengo password=testpass dbname=gengowatcher_test sslmode=disable
JWT_SECRET=test-secret-for-testing-only
REDIS_URL=redis://localhost:6379/1
```

**Dependencies to Add:**
- `github.com/stretchr/testify` - Assertions and test suites
- `github.com/gorilla/websocket` - WebSocket client for testing
- `github.com/testcontainers/testcontainers-go` - Docker containers for integration tests

#### 2. Frontend Test Setup

**Directory Structure:**
```
frontend/
├── tests/
│   ├── unit/
│   │   ├── store/
│   │   │   ├── watcher.test.ts
│   │   │   └── jobs.test.ts
│   │   ├── hooks/
│   │   │   └── use-watcher-websocket.test.ts
│   │   └── components/
│   │       └── watcher/
│   │           ├── config-form.test.tsx
│   │           └── job-list.test.tsx
│   ├── integration/
│   │   └── dashboard-flow.test.tsx
│   └── setup.ts
├── vitest.config.ts
└── msw.mock.ts
```

**Dependencies to Add:**
- `vitest` - Fast test runner
- `@testing-library/react` - Component testing utilities
- `@testing-library/user-event` - User interaction simulation
- `msw` - API mocking for HTTP/WebSocket
- `happy-dom` or `jsdom` - DOM environment

---

### Unit Tests to Implement

#### Backend Unit Tests

**`tests/handlers/websocket_test.go`**
```go
func TestWebSocketHandler_HandleWebSocket(t *testing.T)
func TestWebSocketHandler_GetUserChannels(t *testing.T)
func TestWebSocketHandler_PublishJob(t *testing.T)
func TestWebSocketHandler_PublishEvent(t *testing.T)
func TestWebSocketHandler_PublishError(t *testing.T)
func TestWebSocketHandler_MissingToken(t *testing.T)
func TestWebSocketHandler_InvalidToken(t *testing.T)
```

**`tests/handlers/watcher_test.go`**
```go
func TestWatcherHandler_GetConfig(t *testing.T)
func TestWatcherHandler_GetConfig_NotFound(t *testing.T)
func TestWatcherHandler_UpdateConfig(t *testing.T)
func TestWatcherHandler_GetState(t *testing.T)
func TestWatcherHandler_StartWatcher(t *testing.T)
func TestWatcherHandler_StopWatcher(t *testing.T)
func TestWatcherHandler_Unauthorized(t *testing.T)
```

**`tests/middleware/jwt_test.go`**
```go
func TestJWTValidator_ValidToken(t *testing.T)
func TestJWTValidator_InvalidToken(t *testing.T)
func TestJWTValidator_MissingToken(t *testing.T)
func TestWebSocketAuth_QueryParameter(t *testing.T)
func TestExtractTokenFromQuery(t *testing.T)
```

#### Frontend Unit Tests

**`tests/unit/store/watcher.test.ts`**
```typescript
describe('useWatcherStore', () => {
  it('initializes with empty state')
  it('fetches config successfully')
  it('handles fetch config error')
  it('updates config')
  it('fetches state')
  it('starts watcher')
  it('stops watcher')
})
```

**`tests/unit/store/jobs.test.ts`**
```typescript
describe('useJobsStore', () => {
  it('initializes empty')
  it('adds job to beginning of list')
  it('enforces maxJobs limit')
  it('clears all jobs')
  it('removes specific job')
})
```

**`tests/unit/hooks/use-watcher-websocket.test.ts`**
```typescript
describe('useWatcherWebSocket', () => {
  it('connects on mount when enabled')
  it('does not connect without token')
  it('handles connected message')
  it('handles event message')
  it('handles error message')
  it('handles job message')
  it('reconnects on disconnect with backoff')
  it('stops reconnecting after max attempts')
})
```

---

### Integration Tests to Implement

#### Backend Integration Tests

**`tests/integration/watcher_flow_test.go`**
```go
func TestWatcherFlow_Complete(t *testing.T) {
    // 1. Register user
    // 2. Login
    // 3. Get config (404 expected)
    // 4. Update config
    // 5. Get config (verify)
    // 6. Start watcher
    // 7. Get state (verify running)
    // 8. Stop watcher
    // 9. Get state (verify stopped)
}
```

**`tests/integration/websocket_flow_test.go`**
```go
func TestWebSocketFlow_Complete(t *testing.T) {
    // 1. Register and login user
    // 2. Connect to WebSocket with token
    // 3. Receive connected message
    // 4. Publish job via Redis
    // 5. Receive job via WebSocket
    // 6. Publish event via Redis
    // 7. Receive event via WebSocket
    // 8. Disconnect and reconnect
}
```

#### Frontend Integration Tests

**`tests/integration/dashboard-flow.test.tsx`**
```typescript
describe('Dashboard Flow', () => {
  it('shows loading state initially')
  it('displays watcher config after loading')
  it('starts watcher on button click')
  it('stops watcher on button click')
  it('opens config modal')
  it('updates config via form')
  it('displays new jobs from WebSocket')
  it('shows error on failure')
})
```

---

### Polishing Tasks

#### UI/UX Improvements

1. **Loading States**
   - Add skeleton loaders for config display
   - Add loading spinners for start/stop actions
   - Add loading state for config form submission

2. **Error Handling**
   - Add toast notifications for errors
   - Add error boundaries for React components
   - Improve error messages from API
   - Add retry buttons for failed operations

3. **Visual Polish**
   - Add transition animations for state changes
   - Add hover states for all interactive elements
   - Improve mobile responsiveness
   - Add keyboard shortcuts (e.g., `Cmd+K` for config)

4. **Accessibility**
   - Add ARIA labels to interactive elements
   - Ensure keyboard navigation works
   - Add screen reader announcements for WebSocket events
   - Test with screen reader

#### Performance Optimizations

1. **Frontend**
   - Add React.memo for JobListItem
   - Virtualize job list for large datasets
   - Debounce config form submissions
   - Add request deduplication for API calls

2. **Backend**
   - Add response compression
   - Add caching headers for static assets
   - Optimize database queries
   - Add connection pooling metrics

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
