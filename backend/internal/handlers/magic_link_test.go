package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/service"
)

// setupMagicLinkTest creates test app with magic link handler
func setupMagicLinkTest(t *testing.T) (*fiber.App, *service.TokenService, *gorm.DB) {
	t.Helper()

	db := setupTestDB(t)
	wrappedDB := &databaseWrapper{db: db}

	tokenAuthSvc := auth.NewTokenService("test-secret-for-testing-only-32-chars-min")
	tokenSvc := service.NewTokenService(db)

	emailSvc := email.NewTestService(&email.Config{
		FromEmail: "test@example.com",
		FromName:  "Test",
		BaseURL:   "http://localhost:3000",
	})

	sessionConfig := DefaultSessionConfig()
	handler := NewMagicLinkHandler(wrappedDB, tokenAuthSvc, emailSvc, tokenSvc, sessionConfig, "http://localhost:3001")

	app := fiber.New(fiber.Config{
		AppName:               "MagicLink Test",
		DisableStartupMessage: true,
	})

	app.Post("/api/v1/auth/magic-link", handler.SendMagicLink)
	app.Post("/api/v1/auth/magic-link/verify", handler.VerifyMagicLink)
	app.Get("/api/v1/auth/magic-link/verify", handler.VerifyMagicLink)

	return app, tokenSvc, db
}

func TestMagicLinkHandler_SendMagicLink_Success(t *testing.T) {
	app, _, db := setupMagicLinkTest(t)

	// Create test user
	user := &models.User{
		Email:         "magiclink-test@example.com",
		EmailVerified: false,
		IsActive:      true,
	}
	require.NoError(t, db.Create(user).Error)

	reqBody := bytes.NewBufferString(`{"email":"magiclink-test@example.com"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/magic-link", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result["message"], "sent")
	assert.NotNil(t, result["expires_at"])
	assert.NotNil(t, result["expires_in_minutes"])
}

func TestMagicLinkHandler_SendMagicLink_CreatesUserIfNotExists(t *testing.T) {
	app, _, _ := setupMagicLinkTest(t)

	// No user pre-created
	email := "newuser-magiclink@example.com"
	reqBody := bytes.NewBufferString(`{"email":"` + email + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/magic-link", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// User should be created during verification
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.NotNil(t, result["expires_at"])
}

func TestMagicLinkHandler_VerifyMagicLink_Success(t *testing.T) {
	app, tokenSvc, db := setupMagicLinkTest(t)

	// Create test user
	user := &models.User{
		Email:         "magiclink-verify@example.com",
		EmailVerified: false,
		IsActive:      true,
	}
	require.NoError(t, db.Create(user).Error)

	// Create magic link token
	token, err := tokenSvc.CreateMagicLinkToken(user.Email)
	require.NoError(t, err)

	reqBody := bytes.NewBufferString(`{"token":"` + token + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/magic-link/verify", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.NotNil(t, result["access_token"])
	assert.NotNil(t, result["user"])

	// Verify user is now email verified
	var updatedUser models.User
	db.Where("email = ?", user.Email).First(&updatedUser)
	assert.True(t, updatedUser.EmailVerified)
}

func TestMagicLinkHandler_VerifyMagicLink_CreatesNewUser(t *testing.T) {
	app, tokenSvc, db := setupMagicLinkTest(t)

	email := "newuser-verify@example.com"
	// Create magic link token for non-existent user
	token, err := tokenSvc.CreateMagicLinkToken(email)
	require.NoError(t, err)

	reqBody := bytes.NewBufferString(`{"token":"` + token + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/magic-link/verify", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.NotNil(t, result["access_token"])
	assert.NotNil(t, result["user"])

	// Verify user was created with email verified
	var newUser models.User
	db.Where("email = ?", email).First(&newUser)
	assert.True(t, newUser.EmailVerified)
	assert.True(t, newUser.IsActive)
}

func TestMagicLinkHandler_VerifyMagicLink_InvalidToken(t *testing.T) {
	app, _, _ := setupMagicLinkTest(t)

	reqBody := bytes.NewBufferString(`{"token":"invalid-token"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/magic-link/verify", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["code"], "INVALID_TOKEN")
}

func TestMagicLinkHandler_VerifyMagicLink_ExpiredToken(t *testing.T) {
	app, _, db := setupMagicLinkTest(t)

	// Create expired token directly in DB
	expiredToken := models.MagicLinkToken{
		Email:     "expired-magic@example.com",
		Token:     "expired-magic-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, db.Create(&expiredToken).Error)

	reqBody := bytes.NewBufferString(`{"token":"expired-magic-token"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/magic-link/verify", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["code"], "TOKEN_EXPIRED")
}

func TestMagicLinkHandler_VerifyMagicLink_AlreadyUsedToken(t *testing.T) {
	app, _, db := setupMagicLinkTest(t)

	// Create used token
	now := time.Now()
	usedToken := models.MagicLinkToken{
		Email:     "used-magic@example.com",
		Token:     "already-used-magic-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		UsedAt:    &now,
	}
	require.NoError(t, db.Create(&usedToken).Error)

	reqBody := bytes.NewBufferString(`{"token":"already-used-magic-token"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/magic-link/verify", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["code"], "TOKEN_ALREADY_USED")
}

func TestMagicLinkHandler_ConcurrentTokenUse(t *testing.T) {
	app, tokenSvc, db := setupMagicLinkTest(t)

	// Create user
	user := &models.User{
		Email:         "concurrent-magic@example.com",
		EmailVerified: false,
		IsActive:      true,
	}
	require.NoError(t, db.Create(user).Error)

	// Create magic link token
	token, err := tokenSvc.CreateMagicLinkToken(user.Email)
	require.NoError(t, err)

	// First request should succeed
	reqBody := bytes.NewBufferString(`{"token":"` + token + `"}`)
	req1 := httptest.NewRequest("POST", "/api/v1/auth/magic-link/verify", reqBody)
	req1.Header.Set("Content-Type", "application/json")

	resp1, err := app.Test(req1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp1.StatusCode)

	// Second request with same token should fail
	reqBody2 := bytes.NewBufferString(`{"token":"` + token + `"}`)
	req2 := httptest.NewRequest("POST", "/api/v1/auth/magic-link/verify", reqBody2)
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := app.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 400, resp2.StatusCode)
}

func TestMagicLinkHandler_VerifyMagicLink_QueryParam(t *testing.T) {
	app, tokenSvc, db := setupMagicLinkTest(t)

	// Create test user
	user := &models.User{
		Email:         "magiclink-query@example.com",
		EmailVerified: false,
		IsActive:      true,
	}
	require.NoError(t, db.Create(user).Error)

	// Create magic link token
	token, err := tokenSvc.CreateMagicLinkToken(user.Email)
	require.NoError(t, err)

	// Test with query parameter (for GET redirect flow)
	verifyURL := "/api/v1/auth/magic-link/verify?token=" + url.QueryEscape(token)
	req := httptest.NewRequest("GET", verifyURL, nil)
	resp, err := app.Test(req, -1) // -1 = don't follow redirects
	require.NoError(t, err)

	// Should redirect to frontend with session
	assert.Contains(t, []int{302, 307}, resp.StatusCode, "Should redirect on successful verification")
}
