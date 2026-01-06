package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TokenResult represents the result of token validation
type TokenResult struct {
	Email     string
	IsExpired bool
	IsUsed    bool
	ExpiresAt time.Time
}

// TokenService handles token lifecycle operations
type TokenService struct {
	db *gorm.DB
}

// NewTokenService creates a new token service
func NewTokenService(db *gorm.DB) *TokenService {
	return &TokenService{db: db}
}

// CreateVerificationToken creates and stores an email verification token
func (s *TokenService) CreateVerificationToken(email string) (string, error) {
	return s.createToken(email, 24*time.Hour, "email_verification_tokens", &EmailVerificationTokenModel{})
}

// CreateMagicLinkToken creates and stores a magic link token
func (s *TokenService) CreateMagicLinkToken(email string) (string, error) {
	return s.createToken(email, 15*time.Minute, "magic_link_tokens", &MagicLinkTokenModel{})
}

// CreatePasswordResetToken creates and stores a password reset token
func (s *TokenService) CreatePasswordResetToken(email string) (string, error) {
	return s.createToken(email, 1*time.Hour, "password_reset_tokens", &PasswordResetTokenModel{})
}

// createToken is a generic token creator
func (s *TokenService) createToken(email string, ttl time.Duration, tableName string, _ interface{}) (string, error) {
	tokenStr, err := generateEmailSecureToken()
	if err != nil {
		return "", fmt.Errorf("TOKEN_GENERATION_FAILED: %w", err)
	}

	// Delete existing unused tokens for this email
	s.db.Table(tableName).Where("email = ? AND used_at IS NULL", email).Delete(nil)

	// Create new token
	expiresAt := time.Now().Add(ttl)
	err = s.db.Table(tableName).Create(map[string]interface{}{
		"email":      email,
		"token":      tokenStr,
		"expires_at": expiresAt,
		"created_at": time.Now(),
	}).Error

	if err != nil {
		return "", fmt.Errorf("TOKEN_CREATION_FAILED: %w", err)
	}

	return tokenStr, nil
}

// ValidateEmailVerificationToken validates an email verification token
func (s *TokenService) ValidateEmailVerificationToken(token string) (*TokenResult, error) {
	var dbToken EmailVerificationTokenModel
	err := s.db.Table("email_verification_tokens").Where("token = ?", token).First(&dbToken).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("INVALID_TOKEN")
		}
		return nil, fmt.Errorf("DATABASE_ERROR: %w", err)
	}

	return s.validateTokenResult(dbToken.Email, dbToken.ExpiresAt, dbToken.UsedAt)
}

// ValidateMagicLinkToken validates a magic link token
func (s *TokenService) ValidateMagicLinkToken(token string) (*TokenResult, error) {
	var dbToken MagicLinkTokenModel
	err := s.db.Table("magic_link_tokens").Where("token = ?", token).First(&dbToken).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("INVALID_TOKEN")
		}
		return nil, fmt.Errorf("DATABASE_ERROR: %w", err)
	}

	return s.validateTokenResult(dbToken.Email, dbToken.ExpiresAt, dbToken.UsedAt)
}

// ValidatePasswordResetToken validates a password reset token
func (s *TokenService) ValidatePasswordResetToken(token string) (*TokenResult, error) {
	var dbToken PasswordResetTokenModel
	err := s.db.Table("password_reset_tokens").Where("token = ?", token).First(&dbToken).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("INVALID_TOKEN")
		}
		return nil, fmt.Errorf("DATABASE_ERROR: %w", err)
	}

	return s.validateTokenResult(dbToken.Email, dbToken.ExpiresAt, dbToken.UsedAt)
}

// validateTokenResult is shared validation logic
func (s *TokenService) validateTokenResult(email string, expiresAt time.Time, usedAt *time.Time) (*TokenResult, error) {
	result := &TokenResult{
		Email:     email,
		ExpiresAt: expiresAt,
	}

	// Check expiration
	if time.Now().After(expiresAt) {
		result.IsExpired = true
		return nil, fmt.Errorf("TOKEN_EXPIRED")
	}

	// Check if already used
	if usedAt != nil {
		result.IsUsed = true
		return nil, fmt.Errorf("TOKEN_ALREADY_USED")
	}

	return result, nil
}

// MarkEmailVerificationTokenUsed marks an email verification token as used
func (s *TokenService) MarkEmailVerificationTokenUsed(token string) error {
	return s.markTokenUsed("email_verification_tokens", token)
}

// MarkMagicLinkTokenUsed marks a magic link token as used
func (s *TokenService) MarkMagicLinkTokenUsed(token string) error {
	return s.markTokenUsed("magic_link_tokens", token)
}

// MarkPasswordResetTokenUsed marks a password reset token as used
func (s *TokenService) MarkPasswordResetTokenUsed(token string) error {
	return s.markTokenUsed("password_reset_tokens", token)
}

// markTokenUsed is shared token marking logic
func (s *TokenService) markTokenUsed(tableName, token string) error {
	now := time.Now()
	return s.db.Table(tableName).Where("token = ?", token).Update("used_at", now).Error
}

// generateEmailSecureToken generates a secure random token
func generateEmailSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Model types for token table access

type EmailVerificationTokenModel struct {
	Email     string
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type MagicLinkTokenModel struct {
	Email     string
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type PasswordResetTokenModel struct {
	Email     string
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
