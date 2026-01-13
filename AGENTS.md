# AGENTS.md - GengoWatcher SaaS

> **Navigation**: See also [backend/AGENTS.md](backend/AGENTS.md), [backend/cmd/translation-worker/AGENTS.md](backend/cmd/translation-worker/AGENTS.md), [frontend/AGENTS.md](frontend/AGENTS.md)

## Project Overview

Multi-tenant job monitoring SaaS transforming GengoWatcher from localhost-only to remotely-hosted.

**Stack**: Go 1.25 (Fiber v2) backend | Next.js 16 (React 19) frontend | Python translation-worker

## Quick Commands

```bash
./scripts/dev.sh up          # Start full dev stack
./scripts/dev.sh down        # Stop everything
cd backend && make test      # Run Go tests
cd frontend && npm run test  # Run frontend tests
```

## Directory Map

```
translation-app/
├── backend/                    # Go 1.25 + Fiber v2 → See backend/AGENTS.md
│   ├── cmd/
│   │   ├── server/            # Application entry point
│   │   └── translation-worker/ # Python subsystem → See AGENTS.md there
│   ├── internal/              # Private packages (handlers, models, watcher, auth)
│   └── tests/                 # Integration tests (mirrors internal/)
├── frontend/                   # Next.js 16 → See frontend/AGENTS.md
│   ├── app/                   # App Router (auth, dashboard, settings)
│   ├── components/            # UI components (auth, watcher, ui/base)
│   ├── lib/                   # Utilities, API client
│   └── store/                 # Zustand stores
├── scripts/                    # Development automation (dev.sh)
├── docker-compose.yml          # PostgreSQL, Redis, MailHog
└── CLAUDE.md                  # Full development guide (~500 lines)
```

## Critical Constraints (MUST NOT violate)

| Constraint | Reason |
|------------|--------|
| **Never return specific email-exists errors** | Account enumeration prevention |
| **Never use localhost/private IPs in URLs** | SSRF protection |
| **Never skip JWT signature verification** | Auth bypass prevention |
| **Never use global `models.DB`** | Use dependency injection |
| **Always filter queries by `user_id`** | Multi-tenancy isolation |
| **JWT_SECRET must be 32+ chars** | Server fails startup without it |

## Key Patterns

### Multi-Tenancy
- All DB queries MUST filter by `user_id`
- Redis keys: `user:{user_id}:*`
- WebSocket rooms: `user:{user_id}:ws`

### Auth Flow
- httpOnly cookies for refresh tokens
- 15min access tokens, 7d refresh
- OAuth: Google, GitHub supported
- Magic links with atomic token consumption

### Memory Management
- LRU cache for in-memory deduplication
- Redis sets with TTL for persistent tracking
- Never use unbounded data structures

## Environment Variables (Required)

```bash
JWT_SECRET=                    # 32+ chars, REQUIRED
DATABASE_URL=                  # Postgres connection
REDIS_URL=                     # Redis connection
```

## Agent Artifacts

Files in `docs/.agents/` are for AI agents:
- `plans/` - Implementation plans, specs
- `reports/` - Code analysis, audits
- `todos/` - Task tracking

---

*For detailed conventions, see CLAUDE.md. For domain-specific guidance, see subdirectory AGENTS.md files.*
