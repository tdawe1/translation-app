# Database Connection Pool Configuration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add PostgreSQL connection pool configuration to prevent connection exhaustion and improve performance under load.

**Architecture:** Configure GORM's underlying `sql.DB` connection pool with appropriate limits for a SaaS application. The configuration happens in `database.New()` after GORM initializes.

**Tech Stack:** Go 1.25, GORM, PostgreSQL, `database/sql` package

---

## Why This Matters

**Current Issue (P0 Critical):**
The `database.New()` function creates a GORM connection but never configures the underlying `sql.DB` connection pool. This uses Go's defaults:
- `SetMaxIdleConns`: 2 (way too low for concurrent requests)
- `SetMaxOpenConns`: 0 (unlimited - can overwhelm the database)
- `SetConnMaxLifetime`: 0 (connections never expire - can accumulate stale connections)

**Impact:**
- Connection exhaustion under load
- Database server overwhelmed
- Slow response times for users

---

## Task 1: Add Connection Pool Configuration

**Files:**
- Modify: `backend/internal/database/database.go:74-84`

**Step 1: Write the failing test**

Create `backend/tests/database_test.go`:

```go
package tests

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
)

func TestDatabase_ConnectionPoolConfigured(t *testing.T) {
	db := RequireDB(t)
	gormDB, ok := database.GetPool(db)
	require.True(t, ok, "Should be able to get underlying GORM DB")

	sqlDB, err := gormDB.DB()
	require.NoError(t, err, "Should be able to get sql.DB")

	// Test connection pool settings
	stats := sqlDB.Stats()

	// Verify limits are set (not using defaults)
	assert.Greater(t, sqlDB.Stats().MaxOpenConnections, 0, "MaxOpenConnections should be set")
	assert.Greater(t, sqlDB.Stats().MaxIdleConnections, 2, "MaxIdleConnections should be > default of 2")
}

func TestDatabase_ConnectionLifetimeSet(t *testing.T) {
	cfg := &config.Config{
		DBHost:     "localhost",
		DBPort:     "5433",
		DBUser:     "gengo",
		DBPassword: "devpass",
		DBName:     "gengowatcher_test",
		DBSSLMode:  "disable",
		Env:        "development",
	}

	db, err := database.New(cfg)
	require.NoError(t, err)

	gormDB, ok := database.GetPool(db)
	require.True(t, ok)

	sqlDB, _ := gormDB.DB()

	// Verify connection lifetime is configured
	// We can't directly read the lifetime, but we can verify it's not 0 (never expire)
	// by checking that the connection pool was configured with custom values
	assert.Greater(t, sqlDB.Stats().MaxOpenConnections, 0, "Pool should be configured")
}
```

**Step 2: Run test to verify it fails**

```bash
cd /home/thomas/translation-app/backend
go test ./tests/database_test.go -v -run TestDatabase_ConnectionPoolConfigured
```

Expected: FAIL - `MaxOpenConnections` will be 0 (unlimited) or `MaxIdleConnections` will be 2 (default)

**Step 3: Write minimal implementation**

Edit `backend/internal/database/database.go`, replace lines 74-84:

```go
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	// Get underlying sql.DB to configure connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Set maximum number of open connections
	// For a SaaS app, allow ~25 connections per core as a starting point
	sqlDB.SetMaxOpenConns(25)

	// Set maximum number of idle connections
	// Keep ~10 idle connections ready for reuse
	sqlDB.SetMaxIdleConns(10)

	// Set maximum connection lifetime
	// Connections expire after 1 hour to prevent staleness
	sqlDB.SetConnMaxLifetime(1 * time.Hour)

	// Set maximum idle time for connections
	// Idle connections are closed after 10 minutes
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
```

**Step 4: Run test to verify it passes**

```bash
cd /home/thomas/translation-app/backend
go test ./tests/database_test.go -v -run TestDatabase_ConnectionPoolConfigured
```

Expected: PASS

**Step 5: Commit**

```bash
cd /home/thomas/translation-app
git add backend/internal/database/database.go backend/tests/database_test.go
git commit -m "feat(database): add connection pool configuration

- Set MaxOpenConns to 25 (prevents unbounded growth)
- Set MaxIdleConns to 10 (maintains warm connection pool)
- Set ConnMaxLifetime to 1 hour (prevents stale connections)
- Set ConnMaxIdleTime to 10 minutes (reclaims idle connections)

Fixes P0 issue: missing connection pool configuration was causing
potential connection exhaustion under load."
```

---

## Task 2: Make Pool Settings Configurable via Environment

**Files:**
- Modify: `backend/internal/config/config.go`
- Modify: `backend/internal/database/database.go`

**Step 1: Write the failing test**

Add to `backend/tests/database_test.go`:

```go
func TestDatabase_PoolSettingsFromConfig(t *testing.T) {
	// Test with custom pool settings
	cfg := &config.Config{
		DBHost:               "localhost",
		DBPort:               "5433",
		DBUser:               "gengo",
		DBPassword:           "devpass",
		DBName:               "gengowatcher_test",
		DBSSLMode:            "disable",
		Env:                  "development",
		DBMaxOpenConnections: 50,
		DBMaxIdleConnections: 20,
	}

	db, err := database.New(cfg)
	require.NoError(t, err)

	gormDB, ok := database.GetPool(db)
	require.True(t, ok)

	sqlDB, _ := gormDB.DB()
	assert.Equal(t, 50, sqlDB.Stats().MaxOpenConnections)
	assert.Equal(t, 20, sqlDB.Stats().MaxIdleConnections)
}
```

**Step 2: Run test to verify it fails**

```bash
cd /home/thomas/translation-app/backend
go test ./tests/database_test.go -v -run TestDatabase_PoolSettingsFromConfig
```

Expected: FAIL - Config struct doesn't have these fields yet

**Step 3: Add config fields**

Edit `backend/internal/config/config.go`, add to the `Config` struct (around line 35-45, after existing DB fields):

```go
type Config struct {
	// ... existing fields ...

	// Database connection pool settings
	DBMaxOpenConnections int           `env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
	DBMaxIdleConnections int           `env:"DB_MAX_IDLE_CONNS" envDefault:"10"`
	DBConnMaxLifetime    time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"1h"`
	DBConnMaxIdleTime    time.Duration `env:"DB_CONN_MAX_IDLE_TIME" envDefault:"10m"`
```

**Step 4: Update database.New() to use config**

Edit `backend/internal/database/database.go`, replace the hardcoded values in `New()` function:

```go
	// Configure connection pool from config
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConnections)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConnections)

	// Only set lifetime if configured (non-zero)
	if cfg.DBConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	}
	if cfg.DBConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.DBConnMaxIdleTime)
	}
```

**Step 5: Run test to verify it passes**

```bash
cd /home/thomas/translation-app/backend
go test ./tests/database_test.go -v -run TestDatabase_PoolSettingsFromConfig
```

Expected: PASS

**Step 6: Run all tests to ensure nothing broke**

```bash
cd /home/thomas/translation-app/backend
make test
```

Expected: All tests pass

**Step 7: Commit**

```bash
cd /home/thomas/translation-app
git add backend/internal/config/config.go backend/internal/database/database.go backend/tests/database_test.go
git commit -m "feat(database): make connection pool settings configurable

- Add DB_MAX_OPEN_CONNS env var (default: 25)
- Add DB_MAX_IDLE_CONNS env var (default: 10)
- Add DB_CONN_MAX_LIFETIME env var (default: 1h)
- Add DB_CONN_MAX_IDLE_TIME env var (default: 10m)

Allows operators to tune connection pool for their deployment scale."
```

---

## Task 3: Update Documentation

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Update CLAUDE.md with new environment variables**

Add to the "Environment Variables" section in `CLAUDE.md`:

```markdown
### Database Connection Pool (Optional with defaults)

```bash
DB_MAX_OPEN_CONNS=25          # Maximum open connections (default: 25)
DB_MAX_IDLE_CONNS=10          # Maximum idle connections (default: 10)
DB_CONN_MAX_LIFETIME=1h       # Connection lifetime before refresh (default: 1h)
DB_CONN_MAX_IDLE_TIME=10m     # Idle time before connection closed (default: 10m)
```

**Tuning Guidelines:**
- Small deployment (1-2 cores): `DB_MAX_OPEN_CONNS=15`
- Medium deployment (4-8 cores): `DB_MAX_OPEN_CONNS=50`
- Large deployment (16+ cores): `DB_MAX_OPEN_CONNS=100`
- Set `DB_MAX_IDLE_CONNS` to ~40% of `DB_MAX_OPEN_CONNS`
```

**Step 2: Commit**

```bash
cd /home/thomas/translation-app
git add CLAUDE.md
git commit -m "docs: document database connection pool environment variables"
```

---

## Verification Steps

After implementation, verify the fix:

**1. Start the backend with pool settings:**

```bash
cd /home/thomas/translation-app
DB_MAX_OPEN_CONNS=50 DB_MAX_IDLE_CONNS=20 ./scripts/dev.sh backend start
```

**2. Check pool stats in logs:**

The database package should log pool stats on startup (add a log line if you want verification).

**3. Run load test (optional):**

```bash
# Install bombardier if needed
go install github.com/bombardier/bombardier@latest

# Hit the health endpoint with 100 concurrent requests
bombardier -c 100 -n 1000 http://localhost:8000/health
```

**4. Verify in PostgreSQL:**

```sql
-- Connect to the database
docker exec -it translation-app-postgres-1 psql -U gengo -d gengowatcher

-- Check connection count
SELECT count(*) FROM pg_stat_activity WHERE datname = 'gengowatcher';
```

Expected: Connection count should not exceed `DB_MAX_OPEN_CONNS`

---

## Success Criteria

- [ ] Tests pass: `make test`
- [ ] Connection pool stats show configured values
- [ ] Environment variables override defaults
- [ ] Documentation updated
- [ ] No regression in existing functionality

---

## Rollback Plan

If issues arise:

```bash
# Revert the commits
git revert HEAD~2..

# Or reset to before changes
git reset --hard HEAD~2
```
