# GengoWatcher SaaS - Current Implementation Plan

**Status**: In Progress
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

## In Progress 🚧

### Frontend Dependencies
- npm install running (node_modules exists)

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
