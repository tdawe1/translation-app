# GengoWatcher SaaS

Multi-tenant job monitoring SaaS with per-user watcher instances.

## Overview

Transforming GengoWatcher from a localhost-only tool to a remotely-hosted multi-tenant SaaS application with:
- Per-user watcher instances (RSS + WebSocket monitoring)
- Subscription billing (Stripe)
- Multi-method authentication (email/password, magic link, OAuth, API keys)
- Real-time notifications (Redis pub/sub)

**Reference Implementation**: `/home/thomas/GengoWatcher` - Domain knowledge for RSS/WebSocket parsing

## Current Sprint

**SPRINT0** (Scaffolding)

## Architecture Conventions

### Backend
- Routes: `/api/v1/*`
- Error format: `{"error": str, "code": str, "details": dict}`
- Async/await for all DB operations
- Tests mirror `src/` structure

### User Isolation
- All queries filtered by `user_id`
- Redis keys: `user:{user_id}:*`
- WebSocket rooms: `user:{user_id}:ws`

### Design Language (Data Factory)
- **Fonts**: IBM Plex Sans (headings), IBM Plex Mono (labels)
- **Cards**: Bento style, 1px border, sharp corners
- **Hover**: Precision focus (border color change, NO shadow lift)
- **Accents**: ROYGBIV for headings ONLY
- **Spacing**: Generous (py-24 to pt-44 sections)

## Development Commands

### Environment
```bash
docker-compose up -d              # Start PostgreSQL, Redis, MailHog
docker-compose ps                 # Check service status
docker-compose down               # Stop services
```

### Database
```bash
alembic revision --autogenerate -m "desc"  # Create migration
alembic upgrade head                       # Apply migrations
alembic current                             # Check current version
```

### Testing
```bash
pytest tests/ -v                                    # Run all tests
pytest tests/ --cov=src/gengowatcher --cov-report=html  # Coverage
mypy src/gengowatcher/                             # Type checking
black src/gengowatcher/                            # Format code
flake8 src/gengowatcher/                           # Linting
```

## Project Structure

```
src/gengowatcher/
├── database/
│   ├── __init__.py
│   ├── models.py      # SQLAlchemy models (User, Subscription, etc.)
│   └── session.py     # Async DB session management
├── auth/
│   ├── security.py    # JWT, password hashing
│   ├── service.py     # AuthService (register, login)
│   ├── routes.py      # /api/v1/auth endpoints
│   └── exceptions.py  # AuthException types
├── watcher/
│   ├── manager.py     # UserWatcherManager (per-user instances)
│   ├── rss.py         # RSSMonitor with user_id
│   └── websocket.py   # WebSocketMonitor with user_id
├── billing/
│   └── routes.py      # Stripe checkout, webhooks
├── email/
│   └── service.py     # Resend email service
└── redis_pubsub.py    # Redis pub/sub manager

tests/
├── auth/
├── watcher/
└── database/

frontend/src/
├── contexts/
│   └── AuthProvider.tsx
├── routes/
│   └── ProtectedRoute.tsx
├── pages/
│   └── auth/
├── components/
│   └── auth/
└── lib/
    └── api.ts
```

## Key Technologies

| Layer | Technology |
|-------|------------|
| Database | SQLAlchemy 2.0 async, PostgreSQL (prod) / SQLite (local) |
| Migrations | Alembic |
| Auth | Argon2id, JWT (15min access), httpOnly cookies (7 day refresh) |
| OAuth | Google, GitHub (Authlib) |
| Email | Resend |
| Billing | Stripe |
| Real-time | Redis pub/sub |
| API | FastAPI |
| Frontend | React 18, TypeScript, Vite, TanStack Query |

## Subscription Tiers

| Tier | Price | Features |
|------|-------|----------|
| Free | $0 | 1 watcher, 100 jobs/day |
| Pro | $29/mo | 3 watchers, 1000 jobs/day, auto-accept |
| Enterprise | $99/mo | Unlimited watchers, priority support |
