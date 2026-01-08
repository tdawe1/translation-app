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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"database/sql"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/service"
)

// databaseWrapper wraps gorm.DB to implement database.Database for testing
type databaseWrapper struct {
	db *gorm.DB
}

func (w *databaseWrapper) Create(value interface{}) *gorm.DB {
	return w.db.Create(value)
}

func (w *databaseWrapper) First(dest interface{}, conds ...interface{}) *gorm.DB {
	return w.db.First(dest, conds...)
}

func (w *databaseWrapper) Where(query interface{}, args ...interface{}) *gorm.DB {
	return w.db.Where(query, args...)
}

func (w *databaseWrapper) Model(value interface{}) *gorm.DB {
	return w.db.Model(value)
}

func (w *databaseWrapper) Begin(opts ...*sql.TxOptions) *gorm.DB {
	return w.db.Begin(opts...)
}

func (w *databaseWrapper) Exec(sql string, values ...interface{}) *gorm.DB {
	return w.db.Exec(sql, values...)
}

func (w *databaseWrapper) Save(value interface{}) *gorm.DB {
	return w.db.Save(value)
}

func (w *databaseWrapper) Updates(values interface{}) *gorm.DB {
	return w.db.Updates(values)
}

func (w *databaseWrapper) UpdateColumn(column string, value interface{}) *gorm.DB {
	return w.db.UpdateColumn(column, value)
}

func (w *databaseWrapper) Update(column string, value interface{}) *gorm.DB {
	return w.db.Update(column, value)
}

func (w *databaseWrapper) Delete(value interface{}, conds ...interface{}) *gorm.DB {
	return w.db.Delete(value, conds...)
}

func (w *databaseWrapper) Offset(offset int) *gorm.DB {
	return w.db.Offset(offset)
}

func (w *databaseWrapper) Limit(limit int) *gorm.DB {
	return w.db.Limit(limit)
}

func (w *databaseWrapper) Order(value interface{}) *gorm.DB {
	return w.db.Order(value)
}

func (w *databaseWrapper) Count(count *int64) *gorm.DB {
	return w.db.Count(count)
}

// setupTestDB creates a test database connection for handler tests
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "host=localhost port=5433 user=gengo password=devpass dbname=gengowatcher_test sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to test database")

	// Clean up before test
	db.Exec("DELETE FROM password_reset_tokens WHERE 1=1")
	db.Exec("DELETE FROM magic_link_tokens WHERE 1=1")
	db.Exec("DELETE FROM email_verification_tokens WHERE 1=1")
	db.Exec("DELETE FROM users WHERE 1=1")

	t.Cleanup(func() {
		// Clean up after test
		db.Exec("DELETE FROM password_reset_tokens WHERE 1=1")
		db.Exec("DELETE FROM magic_link_tokens WHERE 1=1")
		db.Exec("DELETE FROM email_verification_tokens WHERE 1=1")
		db.Exec("DELETE FROM users WHERE 1=1")

		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return db
}

func setupEmailVerificationTest(t *testing.T) (*fiber.App, *service.TokenService, *gorm.DB) {
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

	handler := NewEmailVerificationHandler(wrappedDB, tokenAuthSvc, emailSvc, tokenSvc)

	app := fiber.New(fiber.Config{
		AppName:               "EmailVerification Test",
		DisableStartupMessage: true,
	})

	app.Post("/api/v1/auth/verify-email/send", handler.SendVerificationEmail)
	app.Post("/api/v1/auth/verify-email", handler.VerifyEmail)

	return app, tokenSvc, db
}

func TestEmailVerificationHandler_SendVerificationEmail_Success(t *testing.T) {
	app, _, db := setupEmailVerificationTest(t)

	// Create test user
	user := &models.User{
		Email:         "verify-test@example.com",
		EmailVerified: false,
		IsActive:      true,
	}
	require.NoError(t, db.Create(user).Error)

	reqBody := bytes.NewBufferString(`{"email":"verify-test@example.com"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/verify-email/send", reqBody)
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

func TestEmailVerificationHandler_SendVerificationEmail_UserNotFound(t *testing.T) {
	app, _, _ := setupEmailVerificationTest(t)

	reqBody := bytes.NewBufferString(`{"email":"nonexistent@example.com"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/verify-email/send", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	// Should return 200 to prevent account enumeration
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	// Generic message that doesn't reveal if account exists
	assert.Contains(t, result["message"], "If an account exists")
}

func TestEmailVerificationHandler_SendVerificationEmail_AlreadyVerified(t *testing.T) {
	app, _, db := setupEmailVerificationTest(t)

	// Create already verified user
	user := &models.User{
		Email:         "already-verified@example.com",
		EmailVerified: true,
		IsActive:      true,
	}
	require.NoError(t, db.Create(user).Error)

	reqBody := bytes.NewBufferString(`{"email":"already-verified@example.com"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/verify-email/send", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"], "already verified")
}

func TestEmailVerificationHandler_VerifyEmail_Success(t *testing.T) {
	app, tokenSvc, db := setupEmailVerificationTest(t)

	// Create unverified user
	user := &models.User{
		Email:         "verify-success@example.com",
		EmailVerified: false,
		IsActive:      true,
	}
	require.NoError(t, db.Create(user).Error)

	//# Create verification token using TokenService
	
	token, err := tokenSvc.CreateVerificationToken(user.Email)
	require.NoError(t, err)

	reqBody := bytes.NewBufferString(`{"token":"` + token + `"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/verify-email", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Contains(t, result["message"], "verified successfully")

	// Verify user is now verified
	var updatedUser models.User
	db.Where("email = ?", user.Email).First(&updatedUser)
	assert.True(t, updatedUser.EmailVerified)
}

func TestEmailVerificationHandler_VerifyEmail_InvalidToken(t *testing.T) {
	app, _, _ := setupEmailVerificationTest(t)

	reqBody := bytes.NewBufferString(`{"token":"invalid-token"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/verify-email", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"], "Invalid")
}

func TestEmailVerificationHandler_VerifyEmail_ExpiredToken(t *testing.T) {
	app, _, db := setupEmailVerificationTest(t)

	// Create expired token directly in DB
	expiredToken := models.EmailVerificationToken{
		Email:     "expired@example.com",
		Token:     "expired-token-test",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, db.Create(&expiredToken).Error)

	reqBody := bytes.NewBufferString(`{"token":"expired-token-test"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/verify-email", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"], "expired")
}

func TestEmailVerificationHandler_VerifyEmail_AlreadyUsedToken(t *testing.T) {
	app, _, db := setupEmailVerificationTest(t)

	// Create used token
	now := time.Now()
	usedToken := models.EmailVerificationToken{
		Email:     "used-token@example.com",
		Token:     "already-used-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		UsedAt:    &now,
	}
	require.NoError(t, db.Create(&usedToken).Error)

	reqBody := bytes.NewBufferString(`{"token":"already-used-token"}`)
	req := httptest.NewRequest("POST", "/api/v1/auth/verify-email", reqBody)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var errorResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["code"], "TOKEN_ALREADY_USED")
}

func TestEmailVerificationHandler_ConcurrentTokenUse(t *testing.T) {
	app, tokenSvc, db := setupEmailVerificationTest(t)

	// Create user
	user := &models.User{
		Email:         "concurrent@example.com",
		EmailVerified: false,
		IsActive:      true,
	}
	require.NoError(t, db.Create(user).Error)

	//# Create verification token
	
	token, err := tokenSvc.CreateVerificationToken(user.Email)
	require.NoError(t, err)

	// First request should succeed
	reqBody := bytes.NewBufferString(`{"token":"` + token + `"}`)
	req1 := httptest.NewRequest("POST", "/api/v1/auth/verify-email", reqBody)
	req1.Header.Set("Content-Type", "application/json")

	resp1, err := app.Test(req1)
	require.NoError(t, err)
	assert.Equal(t, 200, resp1.StatusCode)

	// Second request with same token should fail (token already used)
	reqBody2 := bytes.NewBufferString(`{"token":"` + token + `"}`)
	req2 := httptest.NewRequest("POST", "/api/v1/auth/verify-email", reqBody2)
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := app.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, 400, resp2.StatusCode)
}
