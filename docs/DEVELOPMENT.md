# Development Documentation

## Overview

This guide covers setting up a local development environment for GengoWatcher.

## Prerequisites

### Required Software

| Software | Version | Purpose |
|----------|---------|---------|
| Go | 1.23+ | Backend runtime |
| Docker | 24.x+ | Database/Redis |
| Node.js | 20.x | Frontend runtime |
| pnpm | 9.x | Frontend package manager |
| Git | Any | Version control |

### Installation

**Go:**

```bash
# macOS
brew install go@1.23

# Linux (download from https://go.dev/dl/)
wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.zshrc
```

**Node.js and pnpm:**

```bash
# macOS
brew install node@20 pnpm

# Linux
curl -fsSL https://get.pnpm.io/install.sh | sh -
```

**Docker:**

```bash
# macOS
brew install --cask docker

# Linux
sudo apt-get install docker.io docker-compose
sudo usermod -aG docker $USER
```

---

## Project Setup

### 1. Clone Repository

```bash
git clone https://github.com/your-org/translation-app.git
cd translation-app
```

### 2. Start Infrastructure

```bash
# Start PostgreSQL, Redis, MailHog
docker-compose up -d

# Verify services
docker-compose ps

# Expected output:
# NAME                    IMAGE                COMMAND    SERVICE    CREATED   STATUS
# gengowatcher-postgres   postgres:17-alpine   "docker-..." postgres   Up        5433/tcp
# gengowatcher-redis      redis:7.4-alpine    "docker-..." redis      Up        6379/tcp
# gengowatcher-mailhog    mailhog/mailhog     "docker-..." mailhog    Up        1025/tcp, 8025/tcp
```

### 3. Backend Setup

```bash
cd backend

# Verify Go installation
go version

# Set environment variables
export JWT_SECRET=dev-secret-change-in-production
export DATABASE_URL=postgres://gengo:devpass@localhost:5433/gengowatcher
export REDIS_URL=redis://localhost:6379/0

# Run database migrations
# Note: May need to install alembic first
pip install alembic
alembic upgrade head

# Start backend server
go run ./cmd/server
```

**Backend runs at:** `http://localhost:8000`

### 4. Frontend Setup

```bash
cd frontend

# Install dependencies
pnpm install

# Start development server
pnpm dev
```

**Frontend runs at:** `http://localhost:3000`

---

## Project Structure

```
translation-app/
├── backend/                    # Go backend
│   ├── cmd/server/             # Application entrypoint
│   ├── internal/               # Private application code
│   │   ├── auth/              # Authentication
│   │   ├── database/          # Database layer
│   │   ├── handlers/          # HTTP handlers
│   │   ├── middleware/        # HTTP middleware
│   │   ├── email/             # Email service
│   │   ├── oauth/             # OAuth providers
│   │   ├── watcher/           # Job watching
│   │   └── config/            # Configuration
│   ├── alembic/               # Database migrations
│   └── tests/                 # Test files
├── frontend/                   # Next.js frontend
│   ├── app/                   # App Router pages
│   │   ├── auth/              # Auth pages
│   │   ├── dashboard/         # Dashboard
│   │   └── settings/          # Settings
│   ├── components/            # React components
│   ├── lib/                   # Utilities
│   ├── store/                 # State management
│   └── types/                 # TypeScript types
├── deploy/                    # Deployment configs
│   ├── k8s/                   # Kubernetes manifests
│   ├── nginx/                 # Nginx config
│   └── scripts/               # Deployment scripts
└── docs/                      # Documentation
```

---

## Development Commands

### Backend Commands

| Command | Description |
|---------|-------------|
| `go run ./cmd/server` | Start development server |
| `go build ./...` | Build all packages |
| `go test ./...` | Run all tests |
| `go vet ./...` | Static analysis |
| `go fmt ./...` | Format code |

**With environment:**

```bash
JWT_SECRET=test-secret-32-chars-minimum-here go test ./internal/...
```

### Frontend Commands

| Command | Description |
|---------|-------------|
| `pnpm dev` | Start dev server |
| `pnpm build` | Production build |
| `pnpm lint` | Run linter |
| `pnpm type-check` | TypeScript check |

### Database Commands

```bash
# Connect to PostgreSQL
psql postgres://gengo:devpass@localhost:5433/gengowatcher

# Connect to Redis
redis-cli -h localhost -p 6379

# View MailHog emails
# Open http://localhost:8025 in browser

# Run migrations
cd backend
alembic upgrade head

# Create new migration
alembic revision -m "description"
```

---

## Environment Variables

### Development (.env)

Create `backend/.env` (or use defaults in docker-compose):

```bash
# Database
DATABASE_URL=postgres://gengo:devpass@localhost:5433/gengowatcher

# Redis
REDIS_URL=redis://localhost:6379/0

# JWT (development default)
JWT_SECRET=dev-secret-change-in-production

# Email (development - logs to console)
RESEND_API_KEY=

# OAuth (empty = development mode)
GOOGLE_OAUTH_CLIENT_ID=
GOOGLE_OAUTH_CLIENT_SECRET=
GITHUB_OAUTH_CLIENT_ID=
GITHUB_OAUTH_CLIENT_SECRET=

# Server
ENV=development
PORT=8000
```

### Frontend

Create `frontend/.env.local`:

```bash
NEXT_PUBLIC_API_URL=http://localhost:8000
```

---

## Testing

### Running Tests

```bash
# Backend tests
cd backend
JWT_SECRET=test-secret-32-chars-minimum-here go test ./... -v

# Frontend tests
cd frontend
pnpm test

# Specific package
go test ./internal/password/... -v

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Test Files

| Location | Description |
|----------|-------------|
| `backend/internal/password/password_test.go` | Password hashing tests |
| `backend/internal/middleware/ratelimit_test.go` | Rate limiting tests |
| `backend/internal/handlers/oauth_test.go` | OAuth handler tests |
| `backend/internal/config/config_test.go` | Config tests |

### Writing Tests

```go
// backend/internal/password/password_test.go
func TestHashPassword(t *testing.T) {
    password := "TestPassword123!"
    
    hash, err := HashPassword(password)
    assert.NoError(t, err)
    assert.NotEmpty(t, hash)
    
    // Verify bcrypt cost is 12
    cost, _ := bcrypt.Cost([]byte(hash))
    assert.Equal(t, 12, cost)
}
```

---

## Code Style

### Go Formatting

```bash
# Format all files
gofmt -w .

# Organize imports
goimports -w .
```

### TypeScript Formatting

```bash
# Format all files
pnpm format

# Check formatting
pnpm format:check
```

### Linting

```bash
# Go
go vet ./...

# Frontend
pnpm lint
```

---

## Debugging

### Backend Debugging

**Using Delve:**

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug server
dlv debug ./cmd/server
```

**Common Issues:**

| Issue | Solution |
|-------|----------|
| "connection refused" to DB | Check docker-compose is running |
| "module not found" | Run `go mod download` |
| "JWT_SECRET not set" | Export environment variable |

### Frontend Debugging

**React DevTools:** Install browser extension

**Debugging API calls:**

```typescript
// Enable debug logging in API client
import { client } from '@/lib/api';

// Add logging interceptor
client.interceptors.request.use((config) => {
  console.log('API Request:', config.method?.toUpperCase(), config.url);
  return config;
});
```

---

## Hot Reload

### Backend

Use `air` for hot reload:

```bash
# Install
go install github.com/air-verse/air@latest

# Run
air
```

### Frontend

Next.js has built-in hot reload. Changes to `app/`, `components/`, `lib/` auto-refresh the page.

---

## Git Workflow

### 1. Create Branch

```bash
git checkout -b feature/my-new-feature
```

### 2. Make Changes

```bash
# Edit files
# Run tests
go test ./... && pnpm test
```

### 3. Commit

```bash
# Stage changes
git add .

# Commit with conventional message
git commit -m "feat: add new feature"

# Push
git push origin feature/my-new-feature
```

### 4. Create Pull Request

Open GitHub PR and request review.

---

## Common Tasks

### Adding a New Endpoint

1. Create handler in `backend/internal/handlers/`
2. Add route in `backend/cmd/server/main.go`
3. Add frontend API function in `frontend/lib/api.ts`
4. Create frontend page in `frontend/app/`
5. Add tests

### Adding a Database Migration

```bash
cd backend
alembic revision -m "add_user_nickname"

# Edit generated file in alembic/versions/
```

### Updating Dependencies

```bash
# Backend
cd backend
go get -u ./...
go mod tidy

# Frontend
cd frontend
pnpm up --latest
```

---

## Troubleshooting

### Docker Issues

```bash
# Reset all containers and volumes
docker-compose down -v
docker-compose up -d

# Check logs
docker-compose logs -f
```

### Port Conflicts

```bash
# Kill process on port
lsof -ti:8000 | xargs kill -9

# Or use different port
PORT=8001 go run ./cmd/server
```

### Database Connection

```bash
# Test connection
psql postgres://gengo:devpass@localhost:5433/gengowatcher -c "SELECT 1"

# Check if Docker is running
docker ps
```

---

## Additional Resources

| Resource | URL |
|----------|-----|
| Go Documentation | https://go.dev/doc/ |
| Fiber Framework | https://docs.gofiber.io/ |
| Next.js Docs | https://nextjs.org/docs |
| React Docs | https://react.dev/ |
| GORM Docs | https://gorm.io/docs/ |

---

**Last Updated**: January 2026
**Version**: 1.0.0
