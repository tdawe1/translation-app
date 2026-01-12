package tests

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/handlers"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/models"
)

// TestGetLinkedAccounts_Success returns the user's linked OAuth accounts
func TestGetLinkedAccounts_Success(t *testing.T) {
	db := RequireDB(t)
	require.NotNil(t, db)

	wrappedDB := database.Wrap(db)

	// Create test config
	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
	}

	// Create OAuth handler
	oauthHandler := handlers.NewOAuthHandler(wrappedDB, nil, cfg, nil)

	// Create test app
	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	jwtCfg := middleware.NewJWTConfig()
	app.Get("/api/v1/oauth/accounts", middleware.JWTValidator(jwtCfg), oauthHandler.GetLinkedAccounts)

	// Create test user with OAuth provider
	user := &models.User{
		Email:         "oauth-linked@example.com",
		EmailVerified: true,
		IsActive:      true,
		Provider:      "google",
		ProviderID:    "google-12345",
	}
	require.NoError(t, db.Create(user).Error, "Failed to create user")

	// Add additional OAuth accounts
	oauthAccount1 := &models.OAuthAccount{
		UserID:         user.ID,
		Provider:       "github",
		ProviderUserID: "github-67890",
	}
	require.NoError(t, db.Create(oauthAccount1).Error, "Failed to create GitHub OAuth account")

	oauthAccount2 := &models.OAuthAccount{
		UserID:         user.ID,
		Provider:       "google",
		ProviderUserID: "google-secondary@example.com",
	}
	require.NoError(t, db.Create(oauthAccount2).Error, "Failed to create secondary Google OAuth account")

	// Generate test token
	authHeader := "Bearer " + GenerateTestToken(t, user.ID)

	// Request linked accounts
	req := httptest.NewRequest("GET", "/api/v1/oauth/accounts", nil)
	req.Header.Set("Authorization", authHeader)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Verify response structure
	assert.Contains(t, result, "linked_accounts")

	// Check that we have linked accounts
	accounts, ok := result["linked_accounts"].([]interface{})
	assert.True(t, ok, "linked_accounts should be an array")
	assert.Equal(t, 2, len(accounts), "Should have 2 linked OAuth accounts (GitHub + secondary Google)")

	// Note: Primary provider stored on User model is NOT included in linked_accounts
	// linked_accounts only contains additional OAuth accounts from oauth_accounts table
}

// TestGetLinkedAccounts_EmptyList returns empty list when user has no additional linked accounts
func TestGetLinkedAccounts_EmptyList(t *testing.T) {
	db := RequireDB(t)
	require.NotNil(t, db)

	wrappedDB := database.Wrap(db)

	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
	}

	oauthHandler := handlers.NewOAuthHandler(wrappedDB, nil, cfg, nil)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	jwtCfg := middleware.NewJWTConfig()
	app.Get("/api/v1/oauth/accounts", middleware.JWTValidator(jwtCfg), oauthHandler.GetLinkedAccounts)

	// Create test user WITHOUT any additional OAuth accounts
	user := CreateTestUser(t, db, "no-oauth@example.com")

	// Generate test token
	authHeader := "Bearer " + GenerateTestToken(t, user.ID)

	req := httptest.NewRequest("GET", "/api/v1/oauth/accounts", nil)
	req.Header.Set("Authorization", authHeader)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Verify empty array
	accounts, ok := result["linked_accounts"].([]interface{})
	assert.True(t, ok, "linked_accounts should be an array")
	assert.Empty(t, accounts, "Should have no linked OAuth accounts")
}

// TestGetLinkedAccounts_Unauthorized returns 401 when not authenticated
func TestGetLinkedAccounts_Unauthorized(t *testing.T) {
	db := RequireDB(t)
	wrappedDB := database.Wrap(db)

	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
	}

	oauthHandler := handlers.NewOAuthHandler(wrappedDB, nil, cfg, nil)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	jwtCfg := middleware.NewJWTConfig()
	app.Get("/api/v1/oauth/accounts", middleware.JWTValidator(jwtCfg), oauthHandler.GetLinkedAccounts)

	// Request without auth token
	req := httptest.NewRequest("GET", "/api/v1/oauth/accounts", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

// TestUnlinkAccount_Success successfully unlinks an OAuth account
func TestUnlinkAccount_Success(t *testing.T) {
	db := RequireDB(t)
	require.NotNil(t, db)

	wrappedDB := database.Wrap(db)

	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
	}

	oauthHandler := handlers.NewOAuthHandler(wrappedDB, nil, cfg, nil)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	jwtCfg := middleware.NewJWTConfig()
	app.Delete("/api/v1/oauth/accounts/:provider", middleware.JWTValidator(jwtCfg), oauthHandler.UnlinkAccount)

	// Create test user with OAuth provider
	user := &models.User{
		Email:         "oauth-unlink@example.com",
		EmailVerified: true,
		IsActive:      true,
	}
	require.NoError(t, db.Create(user).Error, "Failed to create user")

	// Add an OAuth account to unlink
	oauthAccount := &models.OAuthAccount{
		UserID:         user.ID,
		Provider:       "github",
		ProviderUserID: "github-to-unlink",
	}
	require.NoError(t, db.Create(oauthAccount).Error, "Failed to create OAuth account")

	// Verify account exists
	var count int64
	db.Model(&models.OAuthAccount{}).Where("user_id = ? AND provider = ?", user.ID, "github").Count(&count)
	assert.Equal(t, int64(1), count, "OAuth account should exist before unlink")

	// Generate test token
	authHeader := "Bearer " + GenerateTestToken(t, user.ID)

	// Unlink the account
	req := httptest.NewRequest("DELETE", "/api/v1/oauth/accounts/github", nil)
	req.Header.Set("Authorization", authHeader)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode, "Should return 204 No Content on success")

	// Verify account was deleted
	db.Model(&models.OAuthAccount{}).Where("user_id = ? AND provider = ?", user.ID, "github").Count(&count)
	assert.Equal(t, int64(0), count, "OAuth account should be deleted after unlink")
}

// TestUnlinkAccount_InvalidProvider returns 400 for invalid provider
func TestUnlinkAccount_InvalidProvider(t *testing.T) {
	db := RequireDB(t)
	wrappedDB := database.Wrap(db)

	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
	}

	oauthHandler := handlers.NewOAuthHandler(wrappedDB, nil, cfg, nil)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	jwtCfg := middleware.NewJWTConfig()
	app.Delete("/api/v1/oauth/accounts/:provider", middleware.JWTValidator(jwtCfg), oauthHandler.UnlinkAccount)

	user := CreateTestUser(t, db, "invalid-provider-test@example.com")
	authHeader := "Bearer " + GenerateTestToken(t, user.ID)

	// Try to unlink invalid provider
	req := httptest.NewRequest("DELETE", "/api/v1/oauth/accounts/facebook", nil)
	req.Header.Set("Authorization", authHeader)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "invalid provider", result["error"])
	assert.Equal(t, "INVALID_PROVIDER", result["code"])
}

// TestUnlinkAccount_Unauthorized returns 401 when not authenticated
func TestUnlinkAccount_Unauthorized(t *testing.T) {
	db := RequireDB(t)
	wrappedDB := database.Wrap(db)

	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
	}

	oauthHandler := handlers.NewOAuthHandler(wrappedDB, nil, cfg, nil)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	jwtCfg := middleware.NewJWTConfig()
	app.Delete("/api/v1/oauth/accounts/:provider", middleware.JWTValidator(jwtCfg), oauthHandler.UnlinkAccount)

	// Request without auth token
	req := httptest.NewRequest("DELETE", "/api/v1/oauth/accounts/github", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

// TestUnlinkAccount_NonExistentProvider returns 204 even when provider not linked
// (idempotent operation - unlinking an already unlinked account should succeed)
func TestUnlinkAccount_NonExistentProvider(t *testing.T) {
	db := RequireDB(t)
	wrappedDB := database.Wrap(db)

	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
	}

	oauthHandler := handlers.NewOAuthHandler(wrappedDB, nil, cfg, nil)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	jwtCfg := middleware.NewJWTConfig()
	app.Delete("/api/v1/oauth/accounts/:provider", middleware.JWTValidator(jwtCfg), oauthHandler.UnlinkAccount)

	user := CreateTestUser(t, db, "no-github@example.com")
	authHeader := "Bearer " + GenerateTestToken(t, user.ID)

	// Try to unlink GitHub when user doesn't have GitHub linked
	req := httptest.NewRequest("DELETE", "/api/v1/oauth/accounts/github", nil)
	req.Header.Set("Authorization", authHeader)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 204, resp.StatusCode, "Should return 204 even when provider not linked (idempotent)")
}
