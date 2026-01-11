package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/tdawe1/translation-app/internal/models"
)

// setupTestDB creates a test database connection for token service tests
func setupTestDB(t *testing.T) *gorm.DB {
	// Use existing test infrastructure
	dsn := "host=localhost port=5432 user=gengo password=devpass dbname=gengowatcher_test sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to test database")

	// Run migrations for token models
	err = db.AutoMigrate(&models.EmailVerificationToken{}, &models.MagicLinkToken{}, &models.PasswordResetToken{})
	require.NoError(t, err, "Failed to run migrations")

	// Clean up before test
	db.Exec("DELETE FROM password_reset_tokens WHERE 1=1")
	db.Exec("DELETE FROM magic_link_tokens WHERE 1=1")
	db.Exec("DELETE FROM email_verification_tokens WHERE 1=1")

	t.Cleanup(func() {
		// Clean up after test
		db.Exec("DELETE FROM password_reset_tokens WHERE 1=1")
		db.Exec("DELETE FROM magic_link_tokens WHERE 1=1")
		db.Exec("DELETE FROM email_verification_tokens WHERE 1=1")

		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return db
}

func TestTokenService_CreateVerificationToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	email := "test@example.com"
	token, err := service.CreateVerificationToken(email)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Greater(t, len(token), 32) // Base64 encoded, longer than raw bytes

	// Verify token was stored
	var stored models.EmailVerificationToken
	err = db.Where("token = ?", token).First(&stored).Error
	assert.NoError(t, err)
	assert.Equal(t, email, stored.Email)
	assert.Nil(t, stored.UsedAt)
}

func TestTokenService_CreateMagicLinkToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	email := "test@example.com"
	token, err := service.CreateMagicLinkToken(email)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token was stored
	var stored models.MagicLinkToken
	err = db.Where("token = ?", token).First(&stored).Error
	assert.NoError(t, err)
	assert.Equal(t, email, stored.Email)
	assert.Nil(t, stored.UsedAt)
}

func TestTokenService_CreatePasswordResetToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	email := "test@example.com"
	token, err := service.CreatePasswordResetToken(email)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token was stored
	var stored models.PasswordResetToken
	err = db.Where("token = ?", token).First(&stored).Error
	assert.NoError(t, err)
	assert.Equal(t, email, stored.Email)
	assert.Nil(t, stored.UsedAt)
}

func TestTokenService_ValidateEmailVerificationToken_Success(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	// Create a valid token
	email := "test@example.com"
	token, _ := service.CreateVerificationToken(email)

	// Validate it
	result, err := service.ValidateEmailVerificationToken(token)

	assert.NoError(t, err)
	assert.Equal(t, email, result.Email)
	assert.False(t, result.IsExpired)
	assert.False(t, result.IsUsed)
}

func TestTokenService_ValidateEmailVerificationToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	_, err := service.ValidateEmailVerificationToken("nonexistent-token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_TOKEN")
}

func TestTokenService_ValidateEmailVerificationToken_Expired(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	// Create expired token
	email := "test@example.com"
	expiredToken := models.EmailVerificationToken{
		Email:     email,
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, db.Create(&expiredToken).Error)

	_, err := service.ValidateEmailVerificationToken("expired-token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TOKEN_EXPIRED")
}

func TestTokenService_ValidateEmailVerificationToken_AlreadyUsed(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	// Create used token
	email := "test@example.com"
	now := time.Now()
	usedToken := models.EmailVerificationToken{
		Email:     email,
		Token:     "used-token",
		ExpiresAt: time.Now().Add(1 * time.Hour),
		UsedAt:    &now,
	}
	require.NoError(t, db.Create(&usedToken).Error)

	_, err := service.ValidateEmailVerificationToken("used-token")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TOKEN_ALREADY_USED")
}

func TestTokenService_MarkEmailVerificationTokenUsed(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	email := "test@example.com"
	token, _ := service.CreateVerificationToken(email)

	err := service.MarkEmailVerificationTokenUsed(token)

	assert.NoError(t, err)

	// Verify it's marked
	var verificationToken models.EmailVerificationToken
	db.Where("token = ?", token).First(&verificationToken)
	assert.NotNil(t, verificationToken.UsedAt)
}

func TestTokenService_MarkMagicLinkTokenUsed(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	email := "test@example.com"
	token, _ := service.CreateMagicLinkToken(email)

	err := service.MarkMagicLinkTokenUsed(token)

	assert.NoError(t, err)

	// Verify it's marked
	var magicLinkToken models.MagicLinkToken
	db.Where("token = ?", token).First(&magicLinkToken)
	assert.NotNil(t, magicLinkToken.UsedAt)
}

func TestTokenService_MarkPasswordResetTokenUsed(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	email := "test@example.com"
	token, _ := service.CreatePasswordResetToken(email)

	err := service.MarkPasswordResetTokenUsed(token)

	assert.NoError(t, err)

	// Verify it's marked
	var resetToken models.PasswordResetToken
	db.Where("token = ?", token).First(&resetToken)
	assert.NotNil(t, resetToken.UsedAt)
}

func TestTokenService_ValidateMagicLinkToken_Success(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	email := "test@example.com"
	token, _ := service.CreateMagicLinkToken(email)

	result, err := service.ValidateMagicLinkToken(token)

	assert.NoError(t, err)
	assert.Equal(t, email, result.Email)
	assert.False(t, result.IsExpired)
	assert.False(t, result.IsUsed)
}

func TestTokenService_ValidatePasswordResetToken_Success(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	email := "test@example.com"
	token, _ := service.CreatePasswordResetToken(email)

	result, err := service.ValidatePasswordResetToken(token)

	assert.NoError(t, err)
	assert.Equal(t, email, result.Email)
	assert.False(t, result.IsExpired)
	assert.False(t, result.IsUsed)
}
