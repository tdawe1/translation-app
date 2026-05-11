# AGENTS.md - GengoWatcher SaaS

> **Navigation**: See also [backend/AGENTS.md](backend/AGENTS.md), [frontend/AGENTS.md](frontend/AGENTS.md), [backend/cmd/translation-worker/AGENTS.md](backend/cmd/translation-worker/AGENTS.md)

## Quick Commands

```bash
# Full dev stack (sources .env automatically)
./scripts/dev.sh up          # docker → backend → frontend
./scripts/dev.sh down        # reverse order + port cleanup
./scripts/dev.sh status      # all services
./scripts/dev.sh logs        # tail everything
./scripts/dev.sh check       # validate environment

# Backend (Go)
cd backend
make test                    # requires PostgreSQL on localhost:5433
make test-with-setup         # spins up test DB, runs tests, tears down
go vet ./...

# Frontend (Next.js) — uses pnpm, not npm
cd frontend
pnpm dev                     # Turbopack dev server (port 37180)
pnpm build                   # uses --webpack (not Turbopack)
pnpm test                    # Vitest (happy-dom)
pnpm test:smoke              # Playwright
pnpm lint
```

## Service Ports

| Service | Default Port | Notes |
|---------|-------------|-------|
| Backend API/WebSocket | 37181 | `HOST`/`PORT` env vars |
| Frontend | 37180 | `FRONTEND_PORT` env var |
| PostgreSQL | 5433 | `POSTGRES_PORT` env var (not 5432) |
| Redis | 6380 | `REDIS_PORT` env var |
| MailHog SMTP | 1025 | `MAILHOG_SMTP_PORT` env var |
| MailHog UI | 8025 | `MAILHOG_UI_PORT` env var |

**Trap**: Backend tests connect to PostgreSQL on **5433**, but Redis tests connect to **localhost:6379 DB 15** (standard Redis port, not 6380).

## Directory Map

```
translation-app/
├── backend/                    # Go 1.25 + Fiber v2
│   ├── cmd/server/            # Entry point, GORM AutoMigrate, DI wiring
│   ├── cmd/admin_seed/        # CLI: create admin users with JWT tokens
│   ├── cmd/translation-worker/# Python subsystem (separate AGENTS.md)
│   ├── internal/              # Private packages
│   │   ├── handlers/          # HTTP handlers (18 files, highest churn)
│   │   ├── watcher/           # RSS/WebSocket monitoring
│   │   ├── middleware/        # JWT, CORS
│   │   ├── models/            # GORM models
│   │   ├── auth/              # JWT, password hashing, token service
│   │   ├── config/            # Env-based config, fail-fast in production
│   │   └── errors/            # Typed error codes
│   └── tests/                 # Integration tests (mirror internal/)
├── frontend/                   # Next.js 16, React 19, pnpm
│   ├── app/                   # App Router (auth, dashboard, settings, [locale])
│   ├── components/            # Domain (auth, watcher, realtime) + ui/base
│   ├── lib/api/               # HttpClient with interceptors + deduplication
│   ├── store/                 # Zustand stores (auth, watcher, jobs, realtime, toast)
│   └── tests/smoke/           # Playwright smoke tests
├── scripts/                    # Dev automation (dev.sh + functions/*.sh)
├── docker-compose.yml          # PostgreSQL, Redis, MailHog, translation-worker
├── backend/docker-compose.test.yml  # Test PostgreSQL on port 5433 (host networking)
├── pyproject.toml              # Root Python deps (FastAPI, Alembic, SQLAlchemy)
└── CLAUDE.md                  # Full 900-line development guide
```

## Critical Constraints (MUST NOT violate)

| Constraint | Reason |
|------------|--------|
| **Never return specific email-exists errors** | Account enumeration prevention |
| **Never use localhost/private IPs in URLs** | SSRF protection (see `internal/watcher/url_validator.go`) |
| **Never skip JWT signature verification** | Auth bypass prevention |
| **Never use global `models.DB`** | Use dependency injection |
| **Always filter queries by `user_id`** | Multi-tenancy isolation |
| **JWT_SECRET must be 32+ chars** | Server fails startup without it |
| **Never ignore `blocklist.Add()` errors** | Silent logout failure / token revocation leak |
| **Redis keys must be `user:{id}:*`** | Multi-tenant Redis isolation |

## Testing

### Backend (Go)

**Prerequisites**: PostgreSQL running on `localhost:5433` (user `gengo`, password `devpass`, db `gengowatcher_test`).

```bash
# Option A: Test DB already running
cd backend && make test

# Option B: Full lifecycle (docker up → test → docker down)
cd backend && make test-with-setup

# Run specific test
go test ./tests/... -run TestMagicLink -v
```

**Test helpers** (`tests/helpers.go`):
- `RequireDB(t)` → PostgreSQL on 5433, auto-migrates, cleans data on cleanup
- `RequireRedis(t)` → Redis on **localhost:6379 DB 15**, flushes on cleanup
- `CreateTestUser(t, db, email)` → creates user + watcher config + watcher state
- `GenerateTestToken(t, userID)` → signed JWT for test requests

### Frontend (Next.js)

Uses **Vitest** with `happy-dom`, not jsdom. `vitest.setup.ts` mocks `next-intl`.

```bash
cd frontend
pnpm test                    # unit/integration
pnpm test:coverage           # V8 coverage
pnpm test:smoke              # Playwright
```

### Python

```bash
pytest                       # from root; expects tests/ directory
pytest --cov=src/gengowatcher --cov-report=html
```

Pre-commit runs: **Black** (120 chars), **flake8** (`--ignore=E203,W503,E501 --max-line-length=120`), **mypy** (`--ignore-missing-imports`).

## Environment & Secrets

**Fail-fast behavior**: `backend/internal/config/config.go` panics on startup if required secrets are missing in production:
- `JWT_SECRET` (always required except `TEST_ENV=true` or `ENV=test`)
- `REDIS_URL`
- `DB_PASSWORD`
- `RESEND_API_KEY`

**Test bypass**: Set `TEST_ENV=true` or `ENV=test` to skip secret validation. The Makefile already sets `JWT_SECRET` for `make test`.

**Backend dev.sh behavior**: Automatically sources `.env` from the repo root before starting the Go binary. Frontend receives `NEXT_PUBLIC_API_URL` and `NEXT_PUBLIC_WS_URL` injected by dev.sh.

## Toolchain Quirks

- **Package manager**: Frontend uses **pnpm** (`pnpm-lock.yaml` present). `npm ci` works in CI but prefer pnpm locally.
- **Next.js build**: `pnpm build` runs `next build --webpack` (explicitly webpack, not Turbopack).
- **Next.js dev**: `pnpm dev` runs `next dev --turbopack`.
- **Database migrations**: Go uses **GORM AutoMigrate** only (`cmd/server/main.go`). No Alembic for Go schema. Python side has Alembic (`alembic.ini` at root).
- **i18n**: Frontend uses `next-intl` with `[locale]` dynamic route segment. Vitest setup mocks `useTranslations`.
- **Health endpoints**: `GET /healthz` (liveness), `GET /readyz` (readiness, checks DB + Redis).
- **Watcher browser worker**: Server-owned Chromium is controlled by the Go watcher through DevTools. Configure with `WATCHER_BROWSER_BINARY`, `WATCHER_BROWSER_PROFILE_ROOT`, `WATCHER_BROWSER_ARTIFACT_ROOT`, `WATCHER_BROWSER_HEADLESS`, and `WATCHER_BROWSER_ACCEPT_SELECTOR`; dashboard pop-ups are not the authoritative browser state.

## CI Notes

Two workflow files exist with inconsistent Go versions:
- `.github/workflows/ci.yml` → Go **1.25**
- `.github/workflows/test.yml` → Go **1.23**

Backend tests in CI run with `TEST_ENV=true` against Postgres 16 on port 5433.

---

*For full conventions and architecture details, see `CLAUDE.md`. For language-specific patterns, see child `AGENTS.md` files.*
