# GengoWatcher SaaS

Multi-tenant job monitoring SaaS with per-user watcher instances.

## Overview

Transforming GengoWatcher from a localhost-only tool to a remotely-hosted multi-tenant SaaS application.

## Features

- **Per-User Watcher Instances**: Each user gets isolated RSS + WebSocket monitoring
- **Multi-Method Authentication**: Email/password, magic links, OAuth (Google/GitHub), API keys
- **Subscription Billing**: LemonSqueezy integration with multiple tiers
- **Real-Time Notifications**: Redis pub/sub for instant job alerts

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.23, Fiber 3.x, GORM |
| Database | PostgreSQL |
| Auth | bcrypt, golang-jwt/jwt, httpOnly cookies |
| Real-time | Redis pub/sub |
| Billing | LemonSqueezy |
| Email | Resend |
| Frontend | Next.js 16, React 19, Zustand |

## Development

### Prerequisites
- Go 1.23+
- Docker & Docker Compose
- Node.js 18+ (for frontend)

### Setup

```bash
# Clone and navigate
cd translation-app

# Start services (PostgreSQL, Redis, MailHog)
docker-compose up -d

# Copy environment template
cp .env.example .env
# Edit .env with your values

# Set required environment variables
export JWT_SECRET=$(openssl rand -hex 32)
export LEMONQUEEZY_WEBHOOK_SECRET=your_webhook_secret
```

### Running

```bash
# Backend API server
cd backend && go run ./cmd/server

# Frontend dev server
cd frontend && npm run dev
```

### Development Commands

```bash
# Run Go tests
cd backend && go test ./...

# Build backend
cd backend && go build -o bin/server ./cmd/server

# Type check backend
cd backend && go vet ./...
```

## Project Status

**Current Sprint**: Sprint 0 (Scaffolding)

See [CLAUDE.md](./CLAUDE.md) for architecture and conventions.

## Reference Implementation

The original GengoWatcher at `/home/thomas/GengoWatcher` serves as the reference for RSS/WebSocket parsing logic.
