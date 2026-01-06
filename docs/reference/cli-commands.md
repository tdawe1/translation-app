# CLI Commands Reference

GengoWatcher provides several command-line interfaces for development, database management, and operational tasks.

## 1. Backend (Go)

Commands are run from the `backend/` directory.

### Server Commands
| Command | Description |
|---------|-------------|
| `go run ./cmd/server` | Start the API server in development mode. |
| `go build -o bin/server ./cmd/server` | Build the production binary. |
| `go test ./...` | Run all unit and integration tests. |
| `go vet ./...` | Run static analysis. |

### Tooling
| Command | Description |
|---------|-------------|
| `air` | Start the server with hot-reload (requires `air` installed). |
| `swag init` | Regenerate OpenAPI/Swagger documentation. |

---

## 2. Database (Alembic)

Commands are run from the `backend/` directory.

| Command | Description |
|---------|-------------|
| `alembic upgrade head` | Apply all pending migrations. |
| `alembic downgrade -1` | Rollback the last migration. |
| `alembic revision --autogenerate -m "desc"` | Create a new migration script. |
| `alembic current` | Show the current database version. |

---

## 3. Frontend (pnpm)

Commands are run from the `frontend/` directory.

| Command | Description |
|---------|-------------|
| `pnpm dev` | Start the Next.js development server. |
| `pnpm build` | Create a production build of the application. |
| `pnpm start` | Run the production build locally. |
| `pnpm lint` | Run ESLint to check for code issues. |
| `pnpm format` | Run Prettier to format the codebase. |

---

## 4. Infrastructure (Docker)

Commands are run from the project root.

| Command | Description |
|---------|-------------|
| `docker-compose up -d` | Start all infrastructure services. |
| `docker-compose ps` | Check the status of running services. |
| `docker-compose logs -f` | Tail logs for all services. |
| `docker-compose down -v` | Stop services and remove volumes (Wipe Data). |

## Next Steps
- [Development Setup](../getting-started/installation.md)
- [Production Deployment](../deployment/production-setup.md)
