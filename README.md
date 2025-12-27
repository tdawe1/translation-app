# GengoWatcher SaaS

Multi-tenant job monitoring SaaS with per-user watcher instances.

## Overview

Transforming GengoWatcher from a localhost-only tool to a remotely-hosted multi-tenant SaaS application.

## Features

- **Per-User Watcher Instances**: Each user gets isolated RSS + WebSocket monitoring
- **Multi-Method Authentication**: Email/password, magic links, OAuth (Google/GitHub), API keys
- **Subscription Billing**: Stripe integration with multiple tiers
- **Real-Time Notifications**: Redis pub/sub for instant job alerts

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | FastAPI, SQLAlchemy 2.0 async |
| Database | PostgreSQL (prod) / SQLite (local) |
| Auth | Argon2id, JWT, httpOnly cookies |
| Real-time | Redis pub/sub |
| Billing | Stripe |
| Email | Resend |
| Frontend | React 18, TypeScript, Vite |

## Development

### Prerequisites
- Python 3.11+
- Docker & Docker Compose
- Node.js 18+ (for frontend)

### Setup

```bash
# Clone and navigate
cd /home/thomas/GengoWatcher-SaaS

# Start services (PostgreSQL, Redis, MailHog)
docker-compose up -d

# Create virtual environment
python -m venv .venv
source .venv/bin/activate  # or `.venv\Scripts\activate` on Windows

# Install dependencies
pip install -r requirements-dev.txt

# Copy environment template
cp .env.example .env
# Edit .env with your values

# Install pre-commit hooks
pre-commit install

# Run tests
pytest tests/ -v
```

### Running

```bash
# API server
uvicorn src.gengowatcher.main:app --reload --host 0.0.0.0 --port 8000

# Frontend dev server
cd frontend && npm run dev
```

## Project Status

**Current Sprint**: Sprint 0 (Scaffolding)

See [CLAUDE.md](./CLAUDE.md) for architecture and conventions.

## Reference Implementation

The original GengoWatcher at `/home/thomas/GengoWatcher` serves as the reference for RSS/WebSocket parsing logic.
