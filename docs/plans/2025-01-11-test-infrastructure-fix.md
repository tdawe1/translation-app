# Test Infrastructure Fix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix PostgreSQL integration tests so they can run without manual database setup, using Docker Compose for test database provisioning.

**Architecture:**
1. Create a dedicated `docker-compose.test.yml` for test database (isolated from dev database)
2. Add a `test:setup` make target to start test database and run migrations
3. Update test helpers to create test database if it doesn't exist
4. Add TestMain that provisions database before running tests

**Tech Stack:** Go 1.25, GORM, PostgreSQL 17, Docker Compose, Make

---

## Task 1: Create Test Database Docker Compose Configuration

**Files:**
- Create: `backend/docker-compose.test.yml`

**Step 1: Create docker-compose.test.yml**

Create a minimal Docker Compose file for the test database:

```yaml
version: "3.8"

services:
  postgres-test:
    image: postgres:17-alpine
    container_name: gengowatcher-postgres-test
    environment:
      POSTGRES_DB: gengowatcher_test
      POSTGRES_USER: gengo
      POSTGRES_PASSWORD: devpass
    ports:
      - "5433:5432"  # Use 5433 to avoid conflicts with dev DB on 5432
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U gengo"]
      interval: 3s
      timeout: 5s
      retries: 5
```

**Step 2: Verify YAML syntax**

Run: `docker-compose -f docker-compose.test.yml config`
Expected: No errors (valid YAML)

**Step 3: Commit**

```bash
git add backend/docker-compose.test.yml
git commit -m "test: add docker-compose configuration for test database

Provides isolated PostgreSQL instance for integration tests using port 5433
to avoid conflicts with development database on port 5432."
```

---

## Task 2: Update Test Database Connection Configuration

**Files:**
- Modify: `backend/tests/helpers.go`

**Step 1: Update TestDB to use test database port**

Replace the `TestDB` function (lines 28-74) with updated configuration:

```go
// TestDB returns a test database connection
// Uses PostgreSQL test database - runs migrations for realistic testing
func TestDB(t *testing.T) *gorm.DB {
	// Construct DSN from individual env vars or use defaults
	// Default to port 5433 for test database (from docker-compose.test.yml)
	dbHost := getEnv("TEST_DB_HOST", "localhost")
	dbPort := getEnv("TEST_DB_PORT", "5433")  // Changed: was 5432, now 5433
	dbUser := getEnv("TEST_DB_USER", "gengo")
	dbPass := getEnv("TEST_DB_PASSWORD", "devpass")
	dbName := getEnv("TEST_DB_NAME", "gengowatcher_test")
	dbSSL := getEnv("TEST_DB_SSLMODE", "disable")

	dsn := "host=" + dbHost + " port=" + dbPort + " user=" + dbUser + " password=" + dbPass + " dbname=" + dbName + " sslmode=" + dbSSL

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Failed to connect to test database. Run: docker-compose -f docker-compose.test.yml up -d")

	// Run migrations to ensure schema is up to date
	err = db.AutoMigrate(
		&models.User{},
		&models.OAuthAccount{},
		&models.APIKey{},
		&models.RefreshToken{},
		&models.WatcherConfig{},
		&models.WatcherState{},
		&models.SubscriptionPlan{},
		&models.Subscription{},
		&models.BillingEvent{},
		&models.AuditLog{},
		&models.EmailVerificationToken{},
		&models.MagicLinkToken{},
		&models.PasswordResetToken{},
	)
	require.NoError(t, err, "Failed to run migrations")

	// Clean up any existing data
	db.Exec("DELETE FROM audit_logs WHERE 1=1")
	db.Exec("DELETE FROM billing_events WHERE 1=1")
	db.Exec("DELETE FROM refresh_tokens WHERE 1=1")
	db.Exec("DELETE FROM api_keys WHERE 1=1")
	db.Exec("DELETE FROM oauth_accounts WHERE 1=1")
	db.Exec("DELETE FROM watcher_states WHERE 1=1")
	db.Exec("DELETE FROM watcher_configs WHERE 1=1")
	db.Exec("DELETE FROM subscriptions WHERE 1=1")
	db.Exec("DELETE FROM users WHERE 1=1")

	t.Cleanup(func() {
		// Clean up after test
		db.Exec("DELETE FROM audit_logs WHERE 1=1")
		db.Exec("DELETE FROM billing_events WHERE 1=1")
		db.Exec("DELETE FROM refresh_tokens WHERE 1=1")
		db.Exec("DELETE FROM api_keys WHERE 1=1")
		db.Exec("DELETE FROM oauth_accounts WHERE 1=1")
		db.Exec("DELETE FROM watcher_states WHERE 1=1")
		db.Exec("DELETE FROM watcher_configs WHERE 1=1")
		db.Exec("DELETE FROM subscriptions WHERE 1=1")
		db.Exec("DELETE FROM users WHERE 1=1")

		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return db
}
```

**Step 2: Verify the code compiles**

Run: `go build ./tests/...`
Expected: No errors

**Step 3: Commit**

```bash
git add backend/tests/helpers.go
git commit -m "test: update TestDB to use port 5433 and run migrations

- Default test database port changed from 5432 to 5433
- Add AutoMigrate call to ensure schema exists before tests
- Improve error message to indicate docker-compose setup command"
```

---

## Task 3: Add Test Setup Make Targets

**Files:**
- Modify: `backend/Makefile`

**Step 1: Add test infrastructure targets**

Add after the `clean` target:

```makefile
# Test Infrastructure
.PHONY: test-db-up test-db-down test-db-logs test-setup

# Start test database (runs in background)
test-db-up:
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker-compose -f docker-compose.test.yml exec -T postgres-test pg_isready -U gengo > /dev/null 2>&1; do \
		echo "Waiting for postgres..."; \
		sleep 1; \
	done
	@echo "Test database is ready on port 5433"

# Stop test database
test-db-down:
	docker-compose -f docker-compose.test.yml down

# View test database logs
test-db-logs:
	docker-compose -f docker-compose.test.yml logs -f

# Run tests with database setup
test-with-setup: test-db-up
	@$(MAKE) test
	@$(MAKE) test-db-down

# Create test database (for manual testing)
test-db-create:
	docker-compose -f docker-compose.test.yml up -d
	@sleep 3
	docker-compose -f docker-compose.test.yml exec -T postgres-test psql -U gengo -c "SELECT 1"
```

**Step 2: Verify make targets work**

Run: `cd backend && make help | grep test` or `make -n test-db-up`
Expected: Target is defined, no errors

**Step 3: Commit**

```bash
git add backend/Makefile
git commit -m "test: add make targets for test database management

Adds test-db-up, test-db-down, test-db-logs, and test-with-setup targets
for managing the test database lifecycle. The test-with-setup target
automatically starts the database, runs tests, and tears down."
```

---

## Task 4: Add Database Wrapper to Import

**Files:**
- Modify: `backend/tests/helpers.go`

**Step 1: Add database wrapper import**

The test helpers use `databaseWrapper` but don't import the database package.
Add this import after line 19:

```go
databaseWrapper "github.com/tdawe1/translation-app/internal/database"
```

**Step 2: Verify the code compiles**

Run: `go build ./tests/...`
Expected: No errors

**Step 3: Commit**

```bash
git add backend/tests/helpers.go
git commit -m "test: add missing database wrapper import

Fixes import for databaseWrapper type used in test setup functions."
```

---

## Task 5: Create Test Setup Script

**Files:**
- Create: `backend/scripts/test-setup.sh`

**Step 1: Create test setup script**

Create a bash script that sets up the test environment:

```bash
#!/bin/bash
# Test setup script - ensures test database is running

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_ROOT"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  GengoWatcher Test Setup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check if docker is available
if ! command -v docker &> /dev/null; then
	echo "Error: docker is not installed or not in PATH"
	exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
	echo "Error: docker-compose is not installed or not in PATH"
	exit 1
fi

# Use docker compose or docker-compose
COMPOSE_CMD="docker-compose"
if docker compose version &> /dev/null; then
	COMPOSE_CMD="docker compose"
fi

# Start test database
echo "Starting test database..."
$COMPOSE_CMD -f backend/docker-compose.test.yml up -d

# Wait for database to be ready
echo "Waiting for database to be ready..."
MAX_TRIES=30
COUNT=0
while [ $COUNT -lt $MAX_TRIES ]; do
	if docker exec gengowatcher-postgres-test pg_isready -U gengo &> /dev/null; then
		echo "Database is ready!"
		break
	fi
	COUNT=$((COUNT + 1))
	echo "Waiting... ($COUNT/$MAX_TRIES)"
	sleep 1
done

if [ $COUNT -eq $MAX_TRIES ]; then
	echo "Error: Database failed to become ready"
	$COMPOSE_CMD -f backend/docker-compose.test.yml logs
	exit 1
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Test database ready!"
echo "  Database: gengowatcher_test"
echo "  Host:     localhost:5433"
echo "  User:     gengo"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Run tests with:"
echo "  cd backend && make test"
echo ""
echo "Stop database with:"
echo "  $COMPOSE_CMD -f backend/docker-compose.test.yml down"
```

**Step 2: Make script executable**

Run: `chmod +x backend/scripts/test-setup.sh`

**Step 3: Commit**

```bash
git add backend/scripts/test-setup.sh
git commit -m "test: add test setup script

Automated script for starting the test database with health checks.
Provides clear instructions for running tests and cleanup."
```

---

## Task 6: Update .gitignore for Test Database

**Files:**
- Modify: `.gitignore` (project root, not backend/)

**Step 1: Add test database volume to gitignore**

Add to the Testing section (after line 33):

```
# Test database volumes
postgres-test-data/
```

**Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: ignore test database volume"
```

---

## Task 7: Update README with Test Instructions

**Files:**
- Modify: `backend/README.md` (create if doesn't exist)

**Step 1: Create or update README with test instructions**

```markdown
# GengoWatcher Backend

## Running Tests

### Quick Start (with test database setup)

```bash
# Option 1: Using make target
make test-with-setup

# Option 2: Manual setup
make test-db-up      # Start test database
make test            # Run tests
make test-db-down    # Stop test database
```

### Test Database Configuration

The test suite requires PostgreSQL. By default, tests use:
- **Database:** `gengowatcher_test`
- **Host:** `localhost:5433`
- **User:** `gengo`
- **Password:** `devpass`

The test database runs on port 5433 to avoid conflicts with the development database on port 5432.

### Running Individual Tests

```bash
# Run all tests
make test

# Run with verbose output
make test-verbose

# Run with coverage
make test-coverage

# Run specific test file
go test ./tests/auth_test.go -v

# Run specific test function
go test ./tests/auth_test.go -run TestMagicLink -v
```

### Test Requirements

- Docker and Docker Compose (for test database)
- Go 1.25 or later
- PostgreSQL (if not using Docker)

### Environment Variables

Tests use default environment variables for testing. Override with:

```bash
TEST_DB_HOST=localhost \
TEST_DB_PORT=5433 \
TEST_DB_NAME=gengowatcher_test \
make test
```
```

**Step 2: Commit**

```bash
git add backend/README.md
git commit -m "docs: add test documentation to backend README"
```

---

## Task 8: Verify Tests Run Successfully

**Files:**
- None (verification task)

**Step 1: Start test database**

Run: `cd backend && make test-db-up`
Expected: Container starts, "Database is ready!" message

**Step 2: Run the tests**

Run: `cd backend && make test`
Expected: All tests pass (or at least connect to DB successfully)

**Step 3: Clean up**

Run: `cd backend && make test-db-down`
Expected: Containers stopped and removed

**Step 4: Commit final updates (if any needed)**

If any files were modified during verification:
```bash
git add <files>
git commit -m "test: final test infrastructure adjustments"
```

---

## Testing Checklist

After implementation, verify:

- [ ] `docker-compose -f backend/docker-compose.test.yml config` - valid YAML
- [ ] `make test-db-up` - starts test database
- [ ] `make test` - tests connect to database successfully
- [ ] `make test-db-down` - stops test database cleanly
- [ ] `make test-with-setup` - full cycle works
- [ ] README.md documents test setup clearly

---

## Notes

- The test database uses port 5433 to avoid conflicts with dev database on 5432
- Test database is completely isolated - uses separate Docker container
- Tests run migrations automatically to ensure schema is current
- Each test cleans up its own data, but database persists between test runs
- Use `make test-db-logs` to debug database connection issues
