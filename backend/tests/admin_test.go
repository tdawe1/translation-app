package tests

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/handlers"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/seeds"
)

// TestAdminHandler_ListUsers tests the admin list users endpoint
func TestAdminHandler_ListUsers(t *testing.T) {
	gormDB := RequireDB(t)
	db := database.Wrap(gormDB)

	tokenSvc := auth.NewTokenService("test-secret-for-testing-only-32-chars-min")
	seeder := seeds.NewAdminSeeder(db, tokenSvc)

	// Create admin user and get valid token
	admin, adminToken, err := seeder.EnsureAdminUser("admin-list-test@example.com", "AdminPass123!")
	require.NoError(t, err)
	require.Equal(t, models.RoleAdmin, admin.Role)

	// Create test users to list
	regularUser := CreateTestUser(t, gormDB, "regular-list@example.com")

	// Create test app
	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
	}

	adminHandler := handlers.NewAdminHandler(db)
	jwtCfg := middleware.NewJWTConfig(middleware.WithSecret(cfg.JWTSecret))
	jwtMiddleware := middleware.JWTValidator(jwtCfg)
	adminMiddleware := middleware.RequireAdmin()

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	// Register admin routes with JWT auth and admin role check
	app.Get("/api/v1/admin/users", jwtMiddleware, adminMiddleware, adminHandler.ListUsers)

	t.Run("ListUsers returns all users for admin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result handlers.ListUsersResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(result.Users), 2) // At least admin and regular user
		assert.Equal(t, result.TotalCount, int64(len(result.Users)))
	})

	t.Run("ListUsers filters by role", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/users?role=user", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result handlers.ListUsersResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		// Should only return users with role=user
		for _, user := range result.Users {
			assert.Equal(t, models.RoleUser, user.Role)
		}
	})

	t.Run("ListUsers filters by search", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/users?search="+regularUser.Email[:10], nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result handlers.ListUsersResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Greater(t, len(result.Users), 0)
	})

	t.Run("ListUsers requires admin role", func(t *testing.T) {
		// Create a regular user with a valid token (not admin)
		// Use the token service directly to generate a token with "user" role
		regularToken, err := tokenSvc.GenerateAccessToken(regularUser.ID, models.RoleUser)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+regularToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		// RequireAdmin middleware should reject non-admin users with 403
		assert.Equal(t, 403, resp.StatusCode)
	})

	t.Run("ListUsers requires authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
		// No authorization header

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})
}

// TestAdminHandler_UpdateUserRole tests the admin update user role endpoint
func TestAdminHandler_UpdateUserRole(t *testing.T) {
	gormDB := RequireDB(t)
	db := database.Wrap(gormDB)

	tokenSvc := auth.NewTokenService("test-secret-for-testing-only-32-chars-min")
	seeder := seeds.NewAdminSeeder(db, tokenSvc)

	// Create admin user
	admin, adminToken, err := seeder.EnsureAdminUser("admin-role-test@example.com", "AdminPass123!")
	require.NoError(t, err)

	// Create regular user to update
	regularUser := CreateTestUser(t, gormDB, "role-update-test@example.com")
	require.Equal(t, models.RoleUser, regularUser.Role)

	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
	}

	adminHandler := handlers.NewAdminHandler(db)
	jwtCfg := middleware.NewJWTConfig(middleware.WithSecret(cfg.JWTSecret))
	jwtMiddleware := middleware.JWTValidator(jwtCfg)
	adminMiddleware := middleware.RequireAdmin()

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	app.Put("/api/v1/admin/users/:id/role", jwtMiddleware, adminMiddleware, adminHandler.UpdateUserRole)

	t.Run("UpdateUserRole promotes user to admin", func(t *testing.T) {
		reqBody := bytes.NewBufferString(`{"role":"admin"}`)
		req := httptest.NewRequest("PUT", "/api/v1/admin/users/"+regularUser.ID.String()+"/role", reqBody)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		user := result["user"].(map[string]interface{})
		assert.Equal(t, "admin", user["role"])
	})

	t.Run("UpdateUserRole demotes admin to user", func(t *testing.T) {
		// Create another admin
		otherAdmin, _, _ := seeder.EnsureAdminUser("other-admin@example.com", "AdminPass123!")

		reqBody := bytes.NewBufferString(`{"role":"user"}`)
		req := httptest.NewRequest("PUT", "/api/v1/admin/users/"+otherAdmin.ID.String()+"/role", reqBody)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		user := result["user"].(map[string]interface{})
		assert.Equal(t, "user", user["role"])
	})

	t.Run("UpdateUserRole prevents changing own role", func(t *testing.T) {
		// SECURITY: Admin cannot change their own role to prevent privilege escalation
		// or accidental lockout. This test verifies the UUID comparison fix.

		reqBody := bytes.NewBufferString(`{"role":"user"}`)
		req := httptest.NewRequest("PUT", "/api/v1/admin/users/"+admin.ID.String()+"/role", reqBody)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode, "Should prevent changing own role")
	})

	t.Run("UpdateUserRole requires valid role", func(t *testing.T) {
		reqBody := bytes.NewBufferString(`{"role":"superadmin"}`)
		req := httptest.NewRequest("PUT", "/api/v1/admin/users/"+regularUser.ID.String()+"/role", reqBody)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

// TestAdminHandler_DeleteUser tests the admin delete user endpoint
func TestAdminHandler_DeleteUser(t *testing.T) {
	gormDB := RequireDB(t)
	db := database.Wrap(gormDB)

	tokenSvc := auth.NewTokenService("test-secret-for-testing-only-32-chars-min")
	seeder := seeds.NewAdminSeeder(db, tokenSvc)

	// Create admin user
	admin, adminToken, err := seeder.EnsureAdminUser("admin-delete-test@example.com", "AdminPass123!")
	require.NoError(t, err)

	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
	}

	adminHandler := handlers.NewAdminHandler(db)
	jwtCfg := middleware.NewJWTConfig(middleware.WithSecret(cfg.JWTSecret))
	jwtMiddleware := middleware.JWTValidator(jwtCfg)
	adminMiddleware := middleware.RequireAdmin()

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	app.Delete("/api/v1/admin/users/:id", jwtMiddleware, adminMiddleware, adminHandler.DeleteUser)

	t.Run("DeleteUser removes user account", func(t *testing.T) {
		// Create user to delete
		userToDelete := CreateTestUser(t, gormDB, "delete-me@example.com")

		req := httptest.NewRequest("DELETE", "/api/v1/admin/users/"+userToDelete.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 204, resp.StatusCode)

		// Verify user is deleted
		var user models.User
		result := db.Where("id = ?", userToDelete.ID).First(&user)
		assert.Error(t, result.Error, "User should be deleted")
	})

	t.Run("DeleteUser prevents deleting own account", func(t *testing.T) {
		// SECURITY: Admin cannot delete their own account to prevent lockout.
		// This test verifies the UUID comparison fix.

		req := httptest.NewRequest("DELETE", "/api/v1/admin/users/"+admin.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode, "Should prevent deleting own account")
	})

	t.Run("DeleteUser returns 404 for non-existent user", func(t *testing.T) {
		fakeID := "00000000-0000-0000-0000-000000000001"
		req := httptest.NewRequest("DELETE", "/api/v1/admin/users/"+fakeID, nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode)
	})
}
