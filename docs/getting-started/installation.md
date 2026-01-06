# Installation Guide

This guide provides detailed instructions for installing and configuring GengoWatcher SaaS across different platforms.

## System Requirements

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| Operating System | Linux (Ubuntu 22.04+) | Linux / macOS |
| Memory | 2GB RAM | 4GB RAM |
| Disk | 1GB free space | 5GB free space |
| Network | Constant internet access | High-speed connection |

## Development Prerequisites

### 1. Backend: Go 1.23+
Install the Go runtime to build and run the backend API.
- **macOS**: `brew install go@1.23`
- **Linux**: Follow instructions at [go.dev/dl](https://go.dev/dl/)

### 2. Frontend: Node.js 22+ & pnpm
We use `pnpm` for efficient package management.
- **Node.js**: [nodejs.org](https://nodejs.org/)
- **pnpm**: `npm install -g pnpm`

### 3. Infrastructure: Docker & Docker Compose
Used for running PostgreSQL, Redis, and local mail testing services.
- **Docker Desktop**: [docker.com](https://www.docker.com/)

---

## Installation Steps

### 1. Repository Setup
Clone the codebase and navigate into the project directory:
```bash
git clone https://github.com/your-org/translation-app.git
cd translation-app
```

### 2. Environment Configuration
GengoWatcher uses environment variables for configuration.

**Backend Configuration:**
```bash
cd backend
cp .env.example .env
```
Generate a secure JWT secret:
```bash
openssl rand -hex 32
```
Paste this into `JWT_SECRET` in your `backend/.env` file.

**Frontend Configuration:**
```bash
cd ../frontend
cp .env.example .env.local
```

### 3. Database & Services
Launch the infrastructure services:
```bash
cd ..
docker-compose up -d
```
This starts:
- **PostgreSQL 17** (Port 5433)
- **Redis 7.4** (Port 6379)
- **MailHog** (SMTP: 1025, Web UI: 8025)

### 4. Database Migrations
Apply the initial schema to your PostgreSQL database:
```bash
cd backend
# Ensure you have alembic installed
pip install alembic
alembic upgrade head
```

### 5. Running the Application

**Start the Backend:**
```bash
go run ./cmd/server
```

**Start the Frontend:**
```bash
cd ../frontend
pnpm install
pnpm dev
```

---

## Platform Specific Notes

### macOS
If you encounter issues with `psql` or `redis-cli`, ensure they are in your PATH via Homebrew:
```bash
brew install postgresql redis
```

### Windows (WSL2)
We strongly recommend using **WSL2 (Ubuntu)** for development. Docker Desktop for Windows should be configured to use the WSL2 backend.

---

## Troubleshooting

### Port Conflicts
If services fail to start, check for port conflicts:
- **8000**: Backend API
- **3000**: Frontend Dashboard
- **5433**: PostgreSQL
- **6379**: Redis

### Docker Errors
If Docker fails to pull images, ensure you are logged in and have internet access. You can reset your environment with:
```bash
docker-compose down -v
docker-compose up -d
```

## Next Steps
- [Authentication Setup](../getting-started/authentication.md)
- [API Reference](../api/overview.md)
