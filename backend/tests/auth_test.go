package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/handlers"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/models"
)

// TestMagicLink_AtomicConsume verifies that magic link tokens are consumed atomically
// Using Redis GETDEL prevents race conditions and ensures single-use tokens
func TestMagicLink_AtomicConsume(t *testing.T) {
	db := RequireDB(t)
	redisClient := RequireRedis(t)
	require.NotNil(t, redisClient, "Redis required for magic link tests")

	wrappedDB := &databaseWrapper{db: db}

	// Clean up any leftover magic link tokens from previous test runs
	ctx := context.Background()
	redisClient.Del(ctx, redisClient.Keys(ctx, "magiclink:*").Val()...)

	// Create test config and services
	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
		ResendAPIKey: "test-key",
		FromEmail:    "test@example.com",
		FromName:     "Test",
	}

	tokenSvc := auth.NewTokenService(cfg.JWTSecret)
	userSvc := auth.NewUserService(wrappedDB, tokenSvc)

	// Create test email service (logs instead of sending real emails)
	emailSvc := email.NewTestService(&email.Config{
		FromEmail: cfg.FromEmail,
		FromName:  cfg.FromName,
		BaseURL:   "http://localhost:3000",
	})

	authHandler := handlers.NewAuthHandler(userSvc, tokenSvc, emailSvc, redisClient, cfg.CookieSecure)

	// Create test app
	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	// Register magic link routes
	app.Post("/api/v1/auth/magic-link", authHandler.RequestMagicLink)
	app.Get("/api/v1/auth/verify", authHandler.VerifyMagicLink)

	_ = CreateTestUser(t, db, "magiclink-test@example.com")

	t.Run("RequestMagicLink stores token in Redis with 15-minute expiry", func(t *testing.T) {
		// Clean up any leftover tokens from previous subtests
		ctx := context.Background()
		redisClient.Del(ctx, redisClient.Keys(ctx, "magiclink:*").Val()...)

		reqBody := bytes.NewBufferString(`{"email":"magiclink-test@example.com"}`)
		req := httptest.NewRequest("POST", "/api/v1/auth/magic-link", reqBody)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Contains(t, result["message"], "magic link has been sent")

		// Verify token was stored in Redis
		keys, err := redisClient.Keys(ctx, "magiclink:*").Result()
		require.NoError(t, err)
		assert.Equal(t, 1, len(keys), "Expected exactly one magic link token in Redis")

		// Verify expiry time (approximately 15 minutes)
		ttl, err := redisClient.TTL(ctx, keys[0]).Result()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, ttl.Seconds(), 14*time.Minute.Seconds(), "Token should have ~15 minute expiry")
		assert.LessOrEqual(t, ttl.Seconds(), 15*time.Minute.Seconds(), "Token should have ~15 minute expiry")
	})

	t.Run("VerifyMagicLink consumes token atomically (single-use)", func(t *testing.T) {
		// Clean up any leftover tokens from previous subtests
		ctx := context.Background()
		redisClient.Del(ctx, redisClient.Keys(ctx, "magiclink:*").Val()...)

		// First, create a magic link token
		reqBody := bytes.NewBufferString(`{"email":"magiclink-test@example.com"}`)
		req := httptest.NewRequest("POST", "/api/v1/auth/magic-link", reqBody)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Get the token from Redis
		keys, err := redisClient.Keys(ctx, "magiclink:*").Result()
		require.NoError(t, err)
		require.Equal(t, 1, len(keys), "Expected magic link token to be created")

		// Extract token from key
		token := keys[0][len("magiclink:"):]

		// Verify the token (first attempt - should succeed)
		verifyURL := "/api/v1/auth/verify?token=" + url.QueryEscape(token)
		req2 := httptest.NewRequest("GET", verifyURL, nil)
		resp2, err := app.Test(req2, -1) // -1 = don't follow redirects
		require.NoError(t, err)

		// Should redirect to dashboard with session cookie
		// Fiber v3 returns 307 Temporary Redirect instead of 302 Found
		assert.Contains(t, []int{302, 307}, resp2.StatusCode, "First verification should redirect")

		// Verify token was consumed (no longer in Redis)
		ttl, err := redisClient.TTL(ctx, keys[0]).Result()
		require.NoError(t, err)
		assert.Equal(t, time.Duration(-2), ttl, "Token should be deleted after consumption (Redis returns -2 for non-existent keys)")

		// Try to verify the same token again (should fail)
		req3 := httptest.NewRequest("GET", verifyURL, nil)
		resp3, err := app.Test(req3, -1) // -1 = don't follow redirects
		require.NoError(t, err)
		assert.Equal(t, 400, resp3.StatusCode, "Second verification should fail - token already consumed")

		var errorResp map[string]interface{}
		err = json.NewDecoder(resp3.Body).Decode(&errorResp)
		require.NoError(t, err)
		assert.Contains(t, errorResp["error"], "Invalid or expired")
	})

	t.Run("Concurrent verification attempts - only one should succeed", func(t *testing.T) {
		// Clean up any leftover tokens from previous subtests
		ctx := context.Background()
		redisClient.Del(ctx, redisClient.Keys(ctx, "magiclink:*").Val()...)

		// Create a new magic link token
		reqBody := bytes.NewBufferString(`{"email":"magiclink-test@example.com"}`)
		req := httptest.NewRequest("POST", "/api/v1/auth/magic-link", reqBody)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Get the token from Redis
		keys, err := redisClient.Keys(ctx, "magiclink:*").Result()
		require.NoError(t, err)
		require.Equal(t, 1, len(keys))

		token := keys[0][len("magiclink:"):]

		// Simulate concurrent verification attempts
		results := make(chan int, 3) // channel for status codes

		for i := 0; i < 3; i++ {
			go func() {
				verifyURL := "/api/v1/auth/verify?token=" + url.QueryEscape(token)
				req := httptest.NewRequest("GET", verifyURL, nil)
				resp, err := app.Test(req, -1) // -1 = don't follow redirects
				if err == nil {
					results <- resp.StatusCode
				} else {
					results <- 0
				}
			}()
		}

		// Collect results
		var statusCodes []int
		for i := 0; i < 3; i++ {
			statusCodes = append(statusCodes, <-results)
		}

		// Count successes (302 or 307 redirect) and failures (400 bad request)
		successCount := 0
		failCount := 0
		for _, code := range statusCodes {
			if code == 302 || code == 307 {
				successCount++
			} else if code == 400 {
				failCount++
			}
		}

		assert.Equal(t, 1, successCount, "Exactly one verification should succeed")
		assert.Equal(t, 2, failCount, "Two verifications should fail")
	})

	t.Run("Expired token is rejected", func(t *testing.T) {
		// Create a token with very short expiry (1 second)
		shortToken := "expired-token-test"
		ctx := context.Background()
		key := "magiclink:" + shortToken
		err := redisClient.Set(ctx, key, "magiclink-test@example.com", time.Second).Err()
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(2 * time.Second)

		// Try to verify expired token
		verifyURL := "/api/v1/auth/verify?token=" + shortToken
		req := httptest.NewRequest("GET", verifyURL, nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode, "Expired token should be rejected")

		var errorResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)
		assert.Contains(t, errorResp["error"], "Invalid or expired")
	})
}

// TestPasswordChange verifies the password change endpoint
func TestPasswordChange(t *testing.T) {
	db := RequireDB(t)
	wrappedDB := &databaseWrapper{db: db}

	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
	}

	tokenSvc := auth.NewTokenService(cfg.JWTSecret)
	userSvc := auth.NewUserService(wrappedDB, tokenSvc)

	authHandler := handlers.NewAuthHandler(userSvc, tokenSvc, nil, nil, cfg.CookieSecure)

	// Create test app
	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	jwtCfg := middleware.NewJWTConfig()
	app.Put("/api/v1/me/password", middleware.JWTValidator(jwtCfg), authHandler.ChangePassword)

	// Create test user
	user := CreateTestUser(t, db, "password-change-test@example.com")

	// Generate test token
	authHeader := "Bearer " + GenerateTestToken(t, user.ID)

	t.Run("PasswordChange succeeds with valid credentials", func(t *testing.T) {
		// Use strong password meeting new requirements: 12+ chars, upper, lower, digit, special
		newPassword := "NewPassword456!"
		reqBody := bytes.NewBufferString(`{"old_password":"password123","new_password":"` + newPassword + `"}`)
		req := httptest.NewRequest("PUT", "/api/v1/me/password", reqBody)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authHeader)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Contains(t, result["message"], "successfully")

		// Verify user can login with new password
		_, apiErr := userSvc.Login(auth.LoginRequest{
			Email:    "password-change-test@example.com",
			Password: newPassword,
		})
		assert.Nil(t, apiErr, "Should be able to login with new password")
	})

	t.Run("PasswordChange fails with wrong old password", func(t *testing.T) {
		// Use strong password for new_password
		reqBody := bytes.NewBufferString(`{"old_password":"wrongPassword","new_password":"NewPassword456!"}`)
		req := httptest.NewRequest("PUT", "/api/v1/me/password", reqBody)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authHeader)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)

		var errorResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)
		assert.Contains(t, errorResp["error"], "incorrect")
	})

	t.Run("PasswordChange fails with weak new password", func(t *testing.T) {
		// "short" doesn't meet the 12+ character requirement
		reqBody := bytes.NewBufferString(`{"old_password":"password123","new_password":"short"}`)
		req := httptest.NewRequest("PUT", "/api/v1/me/password", reqBody)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authHeader)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)

		var errorResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResp)
		require.NoError(t, err)
		// New validation requires 12+ characters
		assert.Contains(t, errorResp["error"], "12 characters")
	})

	t.Run("PasswordChange fails without authentication", func(t *testing.T) {
		// Use strong password for new_password
		reqBody := bytes.NewBufferString(`{"old_password":"password123","new_password":"NewPassword456!"}`)
		req := httptest.NewRequest("PUT", "/api/v1/me/password", reqBody)
		req.Header.Set("Content-Type", "application/json")
		// No Authorization header

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})
}

// TestUserResponse_IncludesOAuthAccounts verifies that user responses include OAuth account data
func TestUserResponse_IncludesOAuthAccounts(t *testing.T) {
	db := RequireDB(t)
	wrappedDB := &databaseWrapper{db: db}

	cfg := &config.Config{
		JWTSecret:    "test-secret-for-testing-only-32-chars-min",
		CookieSecure: false,
	}

	tokenSvc := auth.NewTokenService(cfg.JWTSecret)
	userSvc := auth.NewUserService(wrappedDB, tokenSvc)

	authHandler := handlers.NewAuthHandler(userSvc, tokenSvc, nil, nil, cfg.CookieSecure)

	// Create test app
	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	jwtCfg := middleware.NewJWTConfig()
	app.Get("/api/v1/me", middleware.JWTValidator(jwtCfg), authHandler.GetMe)

	// Create test user with OAuth provider
	user := &models.User{
		Email:         "oauth-user@example.com",
		EmailVerified: true,
		IsActive:      true,
		Provider:      "google",
		ProviderID:    "google-12345",
	}
	require.NoError(t, db.Create(user).Error, "Failed to create user")

	// Add an additional OAuth account
	oauthAccount := &models.OAuthAccount{
		UserID:         user.ID,
		Provider:       "github",
		ProviderUserID: "github-67890",
	}
	require.NoError(t, db.Create(oauthAccount).Error, "Failed to create OAuth account")

	// Generate test token
	authHeader := "Bearer " + GenerateTestToken(t, user.ID)

	// Request user data
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.Header.Set("Authorization", authHeader)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result handlers.UserResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	// Verify OAuth accounts are included
	assert.Equal(t, "google", result.Provider, "Primary provider should be included")
	assert.NotNil(t, result.OAuthAccounts, "OAuth accounts should not be nil")

	// Should have GitHub OAuth account
	hasGitHub := false
	for _, account := range result.OAuthAccounts {
		if account.Provider == "github" {
			hasGitHub = true
			assert.NotEmpty(t, account.CreatedAt, "OAuth account should have created_at timestamp")
		}
	}
	assert.True(t, hasGitHub, "GitHub OAuth account should be included")
}
