# GengoWatcher SaaS

Multi-tenant job monitoring SaaS transforming GengoWatcher from localhost-only to remotely-hosted.

## Overview

GengoWatcher SaaS provides isolated job monitoring for multiple users, featuring real-time notifications, automated translation workflows, and a modern dashboard.

## Features

- **Isolated Watchers**: Per-user RSS and WebSocket monitoring instances.
- **Multi-tenant Auth**: Magic links, OAuth (Google/GitHub), and API key support.
- **Real-time Alerts**: Instant job notifications powered by Redis pub/sub.
- **Automated Processing**: Background translation worker for job automation.
- **Launch Scope**: First online release focuses on job detection, API/WebSocket delivery, and auto-opening job pages from the web dashboard. Payment/provider billing is intentionally out of scope.

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.25, Fiber v2, GORM |
| Frontend | Next.js 16, React 19, Zustand |
| Worker | Python (translation-worker) |
| Database | PostgreSQL |
| Real-time | Redis |
| Email | MailHog (dev) / Resend (prod) |

## Quick Start

The primary development workflow uses the `dev.sh` controller.

```bash
# Start full development stack (Docker + Backend + Frontend)
./scripts/dev.sh up

# View logs from all services
./scripts/dev.sh logs

# Stop everything
./scripts/dev.sh down
```

## Infrastructure Services

Managed via `docker-compose.yml`:

- **PostgreSQL**: Primary data store for user and watcher metadata.
- **Redis**: Real-time event distribution and token revocation.
- **MailHog**: Local SMTP server for testing email delivery.
- **translation-worker**: Python subsystem for background processing.

## Development

### Backend

```bash
cd backend
go run ./cmd/server    # Start API server
make test              # Run all tests
```

### Frontend

```bash
cd frontend
npm run dev            # Start development server
npm run test           # Run Vitest suite
```

## Testing

### Backend Tests

Requires a running test database.

```bash
cd backend
make test-with-setup   # Automates test DB lifecycle and runs tests
```

### Frontend Tests

```bash
cd frontend
npm run test           # Unit and integration tests
npm run test:smoke     # Playwright E2E smoke tests
```

## Documentation

- [CLAUDE.md](./CLAUDE.md) - Primary guide for development conventions and architecture.
- [AGENTS.md](./AGENTS.md) - Authoritative project overview and directory map.
- [backend/AGENTS.md](./backend/AGENTS.md) - Go backend technical documentation.
- [frontend/AGENTS.md](./frontend/AGENTS.md) - Next.js frontend technical documentation.

## Reference

The original GengoWatcher implementation serves as the logic reference for parsing and monitoring patterns.
