package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/password"
	"github.com/tdawe1/translation-app/internal/service"
)

// setupPasswordResetTest creates test app with password reset handler
func setupPasswordResetTest(t *testing.T) (*fiber.App, *service.TokenService, *gorm.DB) {
	t.Helper()

	db := setupTestDB(t)
	wrappedDB := &databaseWrapper{db: db}

	tokenSvc := service.NewTokenService(db)

	emailSvc := email.NewTestService(&email.Config{
		FromEmail: "test@example.com",
		FromName:  "Test",
		BaseURL:   "http://localhost:3000",
	})

	handler := NewPasswordResetHandler(wrappedDB, emailSvc, tokenSvc)

	app := fiber.New(fiber.Config{
		AppName:               "PasswordReset Test",
		DisableStartupMessage: true,
	})

	app.Post("/api/v1/auth/password-reset", handler.SendPasswordReset)
	app.Post("/api/v1/auth/password-reset/verify", handler.ResetPassword)

	return app, tokenSvc, db
}

func TestPasswordResetHandler_SendPasswordReset_Success(t *testing.T) {
	app, _, db := setupPasswordResetTest(t)

	// Create test user with password
	hashedPassword, err := password.HashPassword("oldpassword123")
	require.NoError(t, err)
	user := &models.User{
		Email:        "reset-test@example.com",
		PasswordHash: hashedPassword,
		IsActive:     true,
	}
	require.NoError(t, db.Create(user).Error)

	reqBody := bytes.NewBufferString(`{"email":"reset-test@example.com"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/password-reset", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result["message"], "sent")
}

func TestPasswordResetHandler_SendPasswordReset_UserNotFound(t *testing.T) {
	app, _, _ := setupPasswordResetTest(t)

	// Non-existent user
	reqBody := bytes.NewBufferString(`{"email":"nonexistent@example.com"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/password-reset", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode) // Still returns 200 to prevent enumeration

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result["message"], "sent")
}

func TestPasswordResetHandler_ResetPassword_Success(t *testing.T) {
	app, tokenSvc, db := setupPasswordResetTest(t)

	// Create test user
	hashedPassword, err := password.HashPassword("oldpassword123")
	require.NoError(t, err)
	user := &models.User{
		Email:        "reset-success@example.com",
		PasswordHash: hashedPassword,
		IsActive:     true,
	}
	require.NoError(t, db.Create(user).Error)

	// Create password reset token
	token, err := tokenSvc.CreatePasswordResetToken(user.Email)
	require.NoError(t, err)

	reqBody := bytes.NewBufferString(`{"token":"` + token + `","password":"newPassword456"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/password-reset/verify", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result["message"], "successfully")

	// Verify password was changed
	var updatedUser models.User
	db.Where("email = ?", user.Email).First(&updatedUser)
	assert.True(t, password.VerifyPassword("newPassword456", updatedUser.PasswordHash))
}

func TestPasswordResetHandler_ResetPassword_InvalidToken(t *testing.T) {
	app, _, _ := setupPasswordResetTest(t)

	reqBody := bytes.NewBufferString(`{"token":"invalid-token","password":"newPassword456"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/password-reset/verify", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var errorResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["code"], "INVALID_TOKEN")
}

func TestPasswordResetHandler_ResetPassword_ExpiredToken(t *testing.T) {
	app, _, db := setupPasswordResetTest(t)

	// Create expired token directly in DB
	expiredToken := models.PasswordResetToken{
		Email:     "expired-reset@example.com",
		Token:     "expired-reset-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, db.Create(&expiredToken).Error)

	reqBody := bytes.NewBufferString(`{"token":"expired-reset-token","password":"newPassword456"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/password-reset/verify", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["code"], "TOKEN_EXPIRED")
}

func TestPasswordResetHandler_ResetPassword_AlreadyUsedToken(t *testing.T) {
	app, _, db := setupPasswordResetTest(t)

	// Create used token
	now := time.Now()
	usedToken := models.PasswordResetToken{
		Email:     "used-reset@example.com",
		Token:     "already-used-reset-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		UsedAt:    &now,
	}
	require.NoError(t, db.Create(&usedToken).Error)

	reqBody := bytes.NewBufferString(`{"token":"already-used-reset-token","password":"newPassword456"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/password-reset/verify", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["code"], "TOKEN_ALREADY_USED")
}

func TestPasswordResetHandler_ResetPassword_WeakPassword(t *testing.T) {
	app, tokenSvc, db := setupPasswordResetTest(t)

	// Create test user
	hashedPassword, err := password.HashPassword("oldpassword123")
	require.NoError(t, err)
	user := &models.User{
		Email:        "weak-password@example.com",
		PasswordHash: hashedPassword,
		IsActive:     true,
	}
	require.NoError(t, db.Create(user).Error)

	// Create password reset token
	token, err := tokenSvc.CreatePasswordResetToken(user.Email)
	require.NoError(t, err)

	reqBody := bytes.NewBufferString(`{"token":"` + token + `","password":"short"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/password-reset/verify", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"], "8 characters")
}

func TestPasswordResetHandler_ConcurrentTokenUse(t *testing.T) {
	app, tokenSvc, db := setupPasswordResetTest(t)

	// Create test user
	hashedPassword, err := password.HashPassword("oldpassword123")
	require.NoError(t, err)
	user := &models.User{
		Email:        "concurrent-reset@example.com",
		PasswordHash: hashedPassword,
		IsActive:     true,
	}
	require.NoError(t, db.Create(user).Error)

	// Create password reset token
	token, err := tokenSvc.CreatePasswordResetToken(user.Email)
	require.NoError(t, err)

	// First request should succeed
	reqBody := bytes.NewBufferString(`{"token":"` + token + `","password":"newPassword123"}`)
	req1 := httptest.NewRequest("POST", "/api/v1/auth/password-reset/verify", reqBody)
	req1.Header.Set("Content-Type", "application/json")

	resp1, err := app.Test(req1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp1.StatusCode)

	// Second request with same token should fail
	reqBody2 := bytes.NewBufferString(`{"token":"` + token + `","password":"anotherPassword"}`)
	req2 := httptest.NewRequest("POST", "/api/v1/auth/password-reset/verify", reqBody2)
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := app.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 400, resp2.StatusCode)
}
