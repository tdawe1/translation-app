# Admin Seeding System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a database-backed admin seeding system that generates valid JWT tokens with the `role` claim required by `RequireAdmin` middleware.

**Architecture:** Create an `AdminSeeder` that creates or updates a user with `role="admin"` in the database, then generates a valid JWT using the configured `JWT_SECRET`. A CLI tool at `cmd/admin_seed` provides a convenient interface for development/testing.

**Tech Stack:** Go 1.25, GORM, PostgreSQL, golang-jwt/jwt, bcrypt

---

## Why This Matters

**Current Issue (P0 Critical):**
The `cmd/gen_jwt/gen_jwt.go` tool generates hardcoded admin tokens that:
- **Missing `role` claim**: `RequireAdmin` middleware expects `claimMap["role"] == "admin"`
- **Hardcoded JWT_SECRET**: Uses embedded secret instead of `JWT_SECRET` env var
- **No database backing**: User ID may not exist in database

**Impact:**
- Smoke tests fail with 403 Forbidden on admin endpoints
- Tests cannot authenticate as admin users
- No reliable way to create admin users for testing

---

## Task 1: Create AdminSeeder Package

**Files:**
- Create: `backend/internal/seeds/admin_seeder.go`

**Step 1: Create the package skeleton**

Create `backend/internal/seeds/admin_seeder.go`:

```go
package seeds

import (
	"fmt"
	"log"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// AdminSeeder handles creating and updating admin users
type AdminSeeder struct {
	db       database.Database
	tokenSvc *auth.TokenService
}

// NewAdminSeeder creates a new admin seeder
func NewAdminSeeder(db database.Database, tokenSvc *auth.TokenService) *AdminSeeder {
	return &AdminSeeder{
		db:       db,
		tokenSvc: tokenSvc,
	}
}

// EnsureAdminUser creates or updates an admin user with the given credentials.
// Returns the user and a valid JWT access token.
func (s *AdminSeeder) EnsureAdminUser(email, password string) (*models.User, string, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Check if user exists
	var user models.User
	result := s.db.Where("email = ?", email).First(&user)

	if result.Error != nil {
		// User doesn't exist - create new admin user
		user = models.User{
			Email:        email,
			PasswordHash: string(hashedPassword),
			IsActive:     true,
			Role:         models.RoleAdmin,
		}

		// Create within transaction for dependent records
		tx := s.db.Begin()
		if tx.Error != nil {
			return nil, "", fmt.Errorf("failed to begin transaction: %w", tx.Error)
		}

		// Create user
		if err := tx.Create(&user).Error; err != nil {
			tx.Rollback()
			return nil, "", fmt.Errorf("failed to create user: %w", err)
		}

		// Create WatcherConfig
		config := models.WatcherConfig{
			UserID:                user.ID,
			IncludedLanguagePairs: "[]",
		}
		if err := tx.Create(&config).Error; err != nil {
			tx.Rollback()
			return nil, "", fmt.Errorf("failed to create watcher config: %w", err)
		}

		// Create WatcherState
		state := models.WatcherState{
			UserID:          user.ID,
			WatcherStatus:   "stopped",
			LastSeenJobIDs:  "[]",
			RecentJobHistory: "[]",
		}
		if err := tx.Create(&state).Error; err != nil {
			tx.Rollback()
			return nil, "", fmt.Errorf("failed to create watcher state: %w", err)
		}

		if err := tx.Commit().Error; err != nil {
			return nil, "", fmt.Errorf("failed to commit transaction: %w", err)
		}

		log.Printf("[AdminSeeder] Created new admin user: %s (ID: %s)", email, user.ID)
	} else {
		// User exists - update role to admin if needed
		if user.Role != models.RoleAdmin {
			user.Role = models.RoleAdmin
			if err := s.db.Save(&user).Error; err != nil {
				return nil, "", fmt.Errorf("failed to update user role: %w", err)
			}
			log.Printf("[AdminSeeder] Updated user to admin: %s (ID: %s)", email, user.ID)
		} else {
			log.Printf("[AdminSeeder] Admin user already exists: %s (ID: %s)", email, user.ID)
		}
	}

	// Generate valid JWT token with admin role
	token, err := s.tokenSvc.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return &user, token, nil
}
```

**Step 2: Verify package compiles**

Run: `cd /home/thomas/translation-app/backend && go build ./internal/seeds/`
Expected: No errors (may have unused warning, that's ok)

---

## Task 2: Create CLI Tool

**Files:**
- Create: `backend/cmd/admin_seed/main.go`

**Step 1: Create the CLI tool**

Create `backend/cmd/admin_seed/main.go`:

```go
package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/seeds"
)

func main() {
	// Parse flags
	email := flag.String("email", "admin@example.com", "Admin user email")
	password := flag.String("password", "", "Password (auto-generated if empty)")
	flag.Parse()

	// Load config
	cfg := config.Load()

	// Connect to database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create services
	tokenSvc := auth.NewTokenService(cfg.JWTSecret)
	seeder := seeds.NewAdminSeeder(db, tokenSvc)

	// Generate password if not provided
	userPassword := *password
	if userPassword == "" {
		userPassword = generateRandomPassword()
		fmt.Printf("Generated password: %s\n", userPassword)
	}

	// Seed admin user
	user, token, err := seeder.EnsureAdminUser(*email, userPassword)
	if err != nil {
		log.Fatalf("Failed to seed admin user: %v", err)
	}

	// Output results
	fmt.Printf("\n✅ Admin user created/updated:\n")
	fmt.Printf("   ID:    %s\n", user.ID)
	fmt.Printf("   Email: %s\n", user.Email)
	fmt.Printf("   Role:  %s\n", user.Role)
	fmt.Printf("\n🔑 Access Token (valid for 15 minutes):\n%s\n", token)
	fmt.Printf("\n💡 Use with: Authorization: Bearer %s\n", token)
}

// generateRandomPassword creates a 16-character random password
func generateRandomPassword() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		log.Fatal(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}
```

**Step 2: Verify CLI tool compiles**

Run: `cd /home/thomas/translation-app/backend && go build -o /tmp/admin_seed ./cmd/admin_seed/`
Expected: No errors

**Step 3: Test the CLI tool**

Run: `cd /home/thomas/translation-app/backend && /tmp/admin_seed -email test-admin@example.com`
Expected: Output showing user ID, email, role, and JWT token

---

## Task 3: Update Admin Tests to Use Seeder

**Files:**
- Modify: `backend/tests/admin_test.go`

**Step 1: Check if admin_test.go exists**

Run: `ls -la backend/tests/admin_test.go`
Expected: File exists (if not, create it)

**Step 2: Write the failing test (or update existing)**

Create or modify `backend/tests/admin_test.go`:

```go
package tests

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/handlers"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/seeds"
)

func TestRequireAdmin_AtomicConsume(t *testing.T) {
	db := RequireDB(t)
	tokenSvc := auth.NewTokenService(testJWTSecret)
	seeder := seeds.NewAdminSeeder(db, tokenSvc)

	// Create admin user with valid token containing role claim
	user, token, err := seeder.EnsureAdminUser(
		"test-admin@example.com",
		"TestAdminPassword123!",
	)
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, models.RoleAdmin, user.Role)
	require.NotEmpty(t, token)

	// Set up Fiber app with admin middleware
	app := fiber.New(fiber.Config{
		AppName:               "Admin Test",
		DisableStartupMessage: true,
	})

	// Create handler
	adminHandler := handlers.NewAdminHandler(db)

	// Protected admin route with JWT + admin check
	app.Get("/api/v1/admin/users",
		middleware.JWTValidator(middleware.NewJWTConfig(middleware.WithSecret(testJWTSecret))),
		middleware.RequireAdmin(),
		adminHandler.ListUsers,
	)

	// Test with valid admin token
	req := httptest.NewRequest("GET", "/api/v1/admin/users?page_size=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should succeed with admin token
	assert.Equal(t, 200, resp.StatusCode)
}

func TestRequireAdmin_RejectsNonAdmin(t *testing.T) {
	db := RequireDB(t)
	tokenSvc := auth.NewTokenService(testJWTSecret)

	// Create regular user (role = "user")
	token, err := tokenSvc.GenerateAccessToken(
		userIDForTest(t, db), // helper to get/create test user
		models.RoleUser,
	)
	require.NoError(t, err)

	app := fiber.New(fiber.Config{
		AppName:               "Admin Test",
		DisableStartupMessage: true,
	})

	adminHandler := handlers.NewAdminHandler(db)

	app.Get("/api/v1/admin/users",
		middleware.JWTValidator(middleware.NewJWTConfig(middleware.WithSecret(testJWTSecret))),
		middleware.RequireAdmin(),
		adminHandler.ListUsers,
	)

	req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should fail with 403 for non-admin
	assert.Equal(t, 403, resp.StatusCode)
}

func TestRequireAdmin_RejectsNoToken(t *testing.T) {
	app := fiber.New(fiber.Config{
		AppName:               "Admin Test",
		DisableStartupMessage: true,
	})

	db := RequireDB(t)
	adminHandler := handlers.NewAdminHandler(db)

	app.Get("/api/v1/admin/users",
		middleware.JWTValidator(nil),
		middleware.RequireAdmin(),
		adminHandler.ListUsers,
	)

	req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Should fail with 401 for no token
	assert.Equal(t, 401, resp.StatusCode)
}
```

**Step 3: Add helper function if needed**

Add to `tests/helpers.go` if `userIDForTest` doesn't exist:

```go
// userIDForTest creates or returns a test user ID
func userIDForTest(t *testing.T, db database.Database) string {
	t.Helper()
	user := CreateTestUser(t, db)
	return user.ID.String()
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/thomas/translation-app/backend && go test ./tests/admin_test.go -v -run TestRequireAdmin_AtomicConsume`
Expected: PASS

---

## Task 4: Update Smoke Tests to Use Admin Seeder

**Files:**
- Modify: `frontend/tests/smoke/admin.spec.ts` (or equivalent)

**Step 1: Locate smoke test files**

Run: `find /home/thomas/translation-app/frontend -name "*smoke*" -o -name "*admin*"`
Expected: List of smoke test files

**Step 2: Update smoke test setup**

Add to your smoke test setup (exact location depends on your test framework):

```typescript
// Before tests, seed admin user
const adminSeedResponse = await fetch('http://localhost:8000/dev/seed-admin', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    email: 'smoke-test-admin@example.com',
    password: 'SmokeTest123!Admin'
  })
});
const { token } = await adminSeedResponse.json();

// Use token in tests
const headers = { Authorization: `Bearer ${token}` };
```

**Note:** You may need to add a dev-only endpoint for seeding, or run the CLI tool before tests.

---

## Task 5: Remove Old gen_jwt Tool

**Files:**
- Delete: `backend/cmd/gen_jwt/gen_jwt.go`

**Step 1: Remove the old tool**

Run: `rm -rf /home/thomas/translation-app/backend/cmd/gen_jwt/`

**Step 2: Verify no references remain**

Run: `grep -r "gen_jwt" /home/thomas/translation-app/backend/`
Expected: No results (except this step in plan)

**Step 3: Commit removal**

Run: `git rm -r backend/cmd/gen_jwt/`

---

## Task 6: Update Documentation

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Add admin seed command to CLAUDE.md**

Add to "Development Commands" section in `CLAUDE.md`:

```markdown
### Admin Seeding

```bash
cd backend

# Create/update admin user (uses JWT_SECRET from env)
go run ./cmd/admin_seed/main.go -email admin@example.com

# With custom password
go run ./cmd/admin_seed/main.go -email admin@example.com -password MySecurePassword123!
```

**Output includes:**
- User ID
- Email
- Role
- JWT access token (15 minute expiry)

**Note:** This tool creates a real admin user in the database. Use only in development/testing.
```

**Step 2: Commit**

Run:
```bash
git add CLAUDE.md
git commit -m "docs: document admin seeding CLI tool"
```

---

## Task 7: Final Integration Test

**Step 1: Run all backend tests**

Run: `cd /home/thomas/translation-app/backend && make test`
Expected: All tests pass

**Step 2: Run smoke tests**

Run: `cd /home/thomas/translation-app/frontend && npm run test`
Expected: Smoke tests pass with admin authentication

**Step 3: Verify CLI tool end-to-end**

Run:
```bash
cd /home/thomas/translation-app/backend
go run ./cmd/admin_seed/main.go -email e2e-test@example.com
# Copy the token
curl -H "Authorization: Bearer <token>" http://localhost:8000/api/v1/admin/users
```
Expected: 200 OK with users list

---

## Verification Steps

After implementation, verify the fix:

**1. Create admin user:**
```bash
cd /home/thomas/translation-app/backend
go run ./cmd/admin_seed/main.go -email verify-admin@example.com
```

**2. Copy the output token and test admin endpoint:**
```bash
export TOKEN="<paste-token-here>"
curl -H "Authorization: Bearer $TOKEN" http://localhost:8000/api/v1/admin/users
```
Expected: 200 OK with JSON users array

**3. Verify token has role claim:**
```bash
# Decode JWT (needs jwt-cli or similar)
echo $TOKEN | jwt decode -
```
Expected: `"role": "admin"` in claims

---

## Success Criteria

- [ ] `AdminSeeder.EnsureAdminUser` creates user with `role="admin"`
- [ ] Generated JWT includes `role` claim with value `"admin"`
- [ ] `RequireAdmin` middleware accepts generated token
- [ ] WatcherConfig and WatcherState created for admin user
- [ ] Existing user gets role updated when re-seeded
- [ ] CLI tool outputs user ID, email, role, and access token
- [ ] All admin tests pass
- [ ] Old `gen_jwt` tool removed
- [ ] Documentation updated

---

## Rollback Plan

If issues arise:

```bash
# Revert all commits from this plan
git revert HEAD~6..

# Or reset to before changes
git reset --hard HEAD~6
```

To restore `gen_jwt` after rollback:
```bash
git checkout HEAD~1 -- backend/cmd/gen_jwt/
```
