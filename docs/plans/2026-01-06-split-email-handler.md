# Split Email Handler Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Refactor 504-line `internal/handlers/email.go` into focused, single-responsibility handlers

**Architecture:** Extract 4 distinct components:
1. Token utilities - shared token validation logic
2. EmailVerificationHandler - email verification flow
3. MagicLinkHandler - passwordless authentication
4. PasswordResetHandler - password reset flow

**Tech Stack:** Go 1.25, Fiber 3.x, GORM

---

## Analysis

**Current file:** `internal/handlers/email.go` (504 lines)

**Code Duplication Detected:**
- Token validation logic repeated 3 times (lines 140-170, 269-298, 437-466)
- Token creation pattern repeated 3 times
- User lookup pattern repeated 3 times

**Extraction Plan:**

| Component | Lines | Responsibility |
|-----------|-------|----------------|
| TokenService | ~80 | Shared token validation, creation, marking used |
| EmailVerificationHandler | ~73 | Send/verify email verification tokens |
| MagicLinkHandler | ~146 | Send/verify magic links for passwordless auth |
| PasswordResetHandler | ~149 | Send password reset, reset password |

---

## Task 1: Extract TokenService for Shared Logic

**Why:** Token validation logic is duplicated 3 times with 90% similarity. Extracting reduces duplication and centralizes token lifecycle management.

**Files:**
- Create: `internal/service/token_service.go`
- Modify: `internal/handlers/email.go` (use new service)

**Step 1: Write failing test for TokenService**

Create `internal/service/token_service_test.go`:

```go
package service

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
)

func TestTokenService_CreateVerificationToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	email := "test@example.com"
	token, err := service.CreateVerificationToken(email)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotEqual(t, 32, len(token)) // Base64 encoded, not raw bytes
}

func TestTokenService_ValidateToken_Success(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	// Create a valid token
	email := "test@example.com"
	token, _ := service.CreateVerificationToken(email)

	// Validate it
	result, err := service.ValidateToken(token)

	assert.NoError(t, err)
	assert.Equal(t, email, result.Email)
	assert.False(t, result.IsExpired)
	assert.False(t, result.IsUsed)
}

func TestTokenService_ValidateToken_Expired(t *testing.T) {
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

	_, err := service.ValidateToken("expired-token")

	assert.Error(t, err)
	assert.Equal(t, "TOKEN_EXPIRED", err.Error())
}

func TestTokenService_ValidateToken_AlreadyUsed(t *testing.T) {
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

	_, err := service.ValidateToken("used-token")

	assert.Error(t, err)
	assert.Equal(t, "TOKEN_ALREADY_USED", err.Error())
}

func TestTokenService_MarkTokenUsed(t *testing.T) {
	db := setupTestDB(t)
	service := NewTokenService(db)

	email := "test@example.com"
	token, _ := service.CreateVerificationToken(email)

	err := service.MarkTokenUsed(token, "email_verification")

	assert.NoError(t, err)

	// Verify it's marked
	var verificationToken models.EmailVerificationToken
	db.Where("token = ?", token).First(&verificationToken)
	assert.NotNil(t, verificationToken.UsedAt)
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/service/... -v -run TestTokenService
```

Expected: `FAIL` with "undefined: NewTokenService"

**Step 3: Implement TokenService**

Create `internal/service/token_service.go`:

```go
package service

import (
	"fmt"
	"time"

	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
	"gorm.io/gorm"
)

// TokenResult represents the result of token validation
type TokenResult struct {
	Email      string
	IsExpired  bool
	IsUsed     bool
	ExpiresAt  time.Time
}

// TokenService handles token lifecycle operations
type TokenService struct {
	db database.Database
}

// NewTokenService creates a new token service
func NewTokenService(db database.Database) *TokenService {
	return &TokenService{db: db}
}

// CreateVerificationToken creates and stores an email verification token
func (s *TokenService) CreateVerificationToken(email string) (string, error) {
	return s.createToken(email, 24*time.Hour, &models.EmailVerificationToken{})
}

// CreateMagicLinkToken creates and stores a magic link token
func (s *TokenService) CreateMagicLinkToken(email string) (string, error) {
	return s.createToken(email, 15*time.Minute, &models.MagicLinkToken{})
}

// CreatePasswordResetToken creates and stored a password reset token
func (s *TokenService) CreatePasswordResetToken(email string) (string, error) {
	return s.createToken(email, 1*time.Hour, &models.PasswordResetToken{})
}

// createToken is a generic token creator using interface{}
func (s *TokenService) createToken(email string, ttl time.Duration, tokenModel interface{}) (string, error) {
	tokenStr, err := generateSecureToken()
	if err != nil {
		return "", fmt.Errorf("TOKEN_GENERATION_FAILED: %w", err)
	}

	// Delete existing unused tokens for this email
	switch tokenModel.(type) {
	case *models.EmailVerificationToken:
		s.db.Where("email = ? AND used_at IS NULL", email).Delete(&models.EmailVerificationToken{})
	case *models.MagicLinkToken:
		s.db.Where("email = ? AND used_at IS NULL", email).Delete(&models.MagicLinkToken{})
	case *models.PasswordResetToken:
		s.db.Where("email = ? AND used_at IS NULL", email).Delete(&models.PasswordResetToken{})
	}

	// Create new token using reflection-like pattern with type switch
	err = s.db.Create(map[string]interface{}{
		"email":      email,
		"token":      tokenStr,
		"expires_at": time.Now().Add(ttl),
	}).Error

	if err != nil {
		return "", fmt.Errorf("TOKEN_CREATION_FAILED: %w", err)
	}

	return tokenStr, nil
}

// ValidateEmailVerificationToken validates an email verification token
func (s *TokenService) ValidateEmailVerificationToken(token string) (*TokenResult, error) {
	var dbToken models.EmailVerificationToken
	err := s.db.Where("token = ?", token).First(&dbToken).Error
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
	var dbToken models.MagicLinkToken
	err := s.db.Where("token = ?", token).First(&dbToken).Error
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
	var dbToken models.PasswordResetToken
	err := s.db.Where("token = ?", token).First(&dbToken).Error
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
	return s.markTokenUsed(token, &models.EmailVerificationToken{})
}

// MarkMagicLinkTokenUsed marks a magic link token as used
func (s *TokenService) MarkMagicLinkTokenUsed(token string) error {
	return s.markTokenUsed(token, &models.MagicLinkToken{})
}

// MarkPasswordResetTokenUsed marks a password reset token as used
func (s *TokenService) MarkPasswordResetTokenUsed(token string) error {
	return s.markTokenUsed(token, &models.PasswordResetToken{})
}

// markTokenUsed is shared token marking logic
func (s *TokenService) markTokenUsed(token string, model interface{}) error {
	now := time.Now()
	return s.db.Model(model).Where("token = ?", token).Update("used_at", now).Error
}
```

**Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/service/... -v -run TestTokenService
```

Expected: All PASS

**Step 5: Commit**

```bash
git add internal/service/token_service.go internal/service/token_service_test.go
git commit -m "feat(service): add TokenService for shared token logic"
```

---

## Task 2: Extract EmailVerificationHandler

**Why:** Isolate email verification responsibility into its own handler. Currently mixed with other email-related flows in the 504-line file.

**Files:**
- Create: `internal/handlers/email_verification.go`
- Modify: `internal/handlers/email.go` (remove extracted methods)

**Step 1: Create EmailVerificationHandler**

Create `internal/handlers/email_verification.go`:

```go
package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/models"
	tokenSvc "github.com/tdawe1/translation-app/internal/service"
	"gorm.io/gorm"
)

// EmailVerificationHandler handles email verification
type EmailVerificationHandler struct {
	db           database.Database
	emailService  *email.Service
	tokenService  *tokenSvc.TokenService
}

// NewEmailVerificationHandler creates a new email verification handler
func NewEmailVerificationHandler(db database.Database, emailService *email.Service, tokenService *tokenSvc.TokenService) *EmailVerificationHandler {
	return &EmailVerificationHandler{
		db:          db,
		emailService: emailService,
		tokenService: tokenService,
	}
}

// SendVerificationRequest represents the request to send a verification email
type SendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// SendVerificationEmail sends a verification email to the user
func (h *EmailVerificationHandler) SendVerificationEmail(c *fiber.Ctx) error {
	var req SendVerificationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Check if user exists
	var user models.User
	err := h.db.Where("email = ?", req.Email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
				"code":  "USER_NOT_FOUND",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
			"code":  "DATABASE_ERROR",
		})
	}

	// Check if already verified
	if user.EmailVerified {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email already verified",
			"code":  "ALREADY_VERIFIED",
		})
	}

	// Create token
	token, err := h.tokenService.CreateVerificationToken(req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create token",
			"code":  "TOKEN_CREATION_FAILED",
		})
	}

	// Send email
	if err := h.emailService.SendVerificationEmail(req.Email, token); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send verification email. Please try again.",
			"code":  "EMAIL_SEND_FAILED",
		})
	}

	// Get expires_at for response
	var verificationToken models.EmailVerificationToken
	h.db.Where("token = ?", token).First(&verificationToken)

	return c.JSON(fiber.Map{
		"message":            "Verification email sent",
		"expires_at":         verificationToken.ExpiresAt.Format(time.RFC3339),
		"expires_in_minutes": int(time.Until(verificationToken.ExpiresAt).Minutes()),
	})
}

// VerifyEmailRequest represents the request to verify an email
type VerifyEmailRequest struct {
	Token string `json:"token" validate:"required"`
}

// VerifyEmail verifies an email using a token
func (h *EmailVerificationHandler) VerifyEmail(c *fiber.Ctx) error {
	var req VerifyEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate token
	result, err := h.tokenService.ValidateEmailVerificationToken(req.Token)
	if err != nil {
		code := err.Error()
		if code == "INVALID_TOKEN" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Invalid or expired token",
				"code":  code,
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
			"code":  code,
		})
	}

	// Find user and update
	var user models.User
	err = h.db.Where("email = ?", result.Email).First(&user).Error
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
			"code":  "USER_NOT_FOUND",
		})
	}

	// Update user as verified
	user.EmailVerified = true
	if err := h.db.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
			"code":  "UPDATE_FAILED",
		})
	}

	// Mark token as used
	if err := h.tokenService.MarkEmailVerificationTokenUsed(req.Token); err != nil {
		// Log but don't fail - user is already verified
		fmt.Printf("Failed to mark token as used: %v\n", err)
	}

	return c.JSON(fiber.Map{
		"message": "Email verified successfully",
	})
}
```

**Step 2: Remove from email.go**

Delete lines 44-199 from `internal/handlers/email.go` (SendVerificationEmail and VerifyEmail methods and their request types).

**Step 3: Build verification**

```bash
cd backend && go build ./cmd/server
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/handlers/email_verification.go internal/handlers/email.go
git commit -m "refactor(handlers): extract EmailVerificationHandler"
```

---

## Task 3: Extract MagicLinkHandler

**Why:** Magic link authentication is a distinct flow from email verification and password reset. It deserves its own handler.

**Files:**
- Create: `internal/handlers/magic_link.go`
- Modify: `internal/handlers/email.go` (remove extracted methods)

**Step 1: Create MagicLinkHandler**

Create `internal/handlers/magic_link.go`:

```go
package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/models"
	tokenSvc "github.com/tdawe1/translation-app/internal/service"
	"gorm.io/gorm"
)

// MagicLinkHandler handles magic link authentication
type MagicLinkHandler struct {
	db           database.Database
	tokenService  *auth.TokenService
	emailService  *email.Service
	tokenStore    *tokenSvc.TokenService
}

// NewMagicLinkHandler creates a new magic link handler
func NewMagicLinkHandler(db database.Database, tokenService *auth.TokenService, emailService *email.Service, tokenStore *tokenSvc.TokenService) *MagicLinkHandler {
	return &MagicLinkHandler{
		db:           db,
		tokenService: tokenService,
		emailService: emailService,
		tokenStore:   tokenStore,
	}
}

// SendMagicLinkRequest represents the request to send a magic link
type SendMagicLinkRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// SendMagicLink sends a magic link for passwordless authentication
func (h *MagicLinkHandler) SendMagicLink(c *fiber.Ctx) error {
	var req SendMagicLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Create token
	token, err := h.tokenStore.CreateMagicLinkToken(req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create token",
			"code":  "TOKEN_CREATION_FAILED",
		})
	}

	// Send email
	if err := h.emailService.SendMagicLinkEmail(req.Email, token); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send magic link email. Please try again.",
			"code":  "EMAIL_SEND_FAILED",
		})
	}

	// Get expires_at for response
	var magicLinkToken models.MagicLinkToken
	h.db.Where("token = ?", token).First(&magicLinkToken)

	return c.JSON(fiber.Map{
		"message":            "Magic link sent to your email",
		"expires_at":         magicLinkToken.ExpiresAt.Format(time.RFC3339),
		"expires_in_minutes": int(time.Until(magicLinkToken.ExpiresAt).Minutes()),
	})
}

// VerifyMagicLinkRequest represents the request to verify a magic link
type VerifyMagicLinkRequest struct {
	Token string `json:"token" validate:"required"`
}

// VerifyMagicLink verifies a magic link and creates a session
func (h *MagicLinkHandler) VerifyMagicLink(c *fiber.Ctx) error {
	var req VerifyMagicLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate token
	result, err := h.tokenStore.ValidateMagicLinkToken(req.Token)
	if err != nil {
		code := err.Error()
		if code == "INVALID_TOKEN" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Invalid or expired link",
				"code":  code,
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
			"code":  code,
		})
	}

	// Find or create user
	var user models.User
	err = h.db.Where("email = ?", result.Email).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		// Create new user
		user = models.User{
			Email:         result.Email,
			EmailVerified: true, // Magic link implies verified email
			IsActive:      true,
		}
		if err := h.db.Create(&user).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create user",
				"code":  "USER_CREATION_FAILED",
			})
		}
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
			"code":  "DATABASE_ERROR",
		})
	}

	// Mark token as used
	if err := h.tokenStore.MarkMagicLinkTokenUsed(req.Token); err != nil {
		fmt.Printf("Failed to mark magic link token as used: %v\n", err)
	}

	// Generate JWT token
	accessToken, err := h.tokenService.GenerateAccessToken(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate access token",
			"code":  "TOKEN_GENERATION_FAILED",
		})
	}

	// Set httpOnly cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    accessToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "lax",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	return c.JSON(fiber.Map{
		"access_token": accessToken,
		"user":         user,
	})
}
```

**Step 2: Remove from email.go**

Delete lines 201-353 from `internal/handlers/email.go` (SendMagicLink and VerifyMagicLink methods).

**Step 3: Build verification**

```bash
cd backend && go build ./cmd/server
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/handlers/magic_link.go internal/handlers/email.go
git commit -m "refactor(handlers): extract MagicLinkHandler"
```

---

## Task 4: Extract PasswordResetHandler

**Why:** Password reset is a security-critical flow that should be isolated and easily auditable.

**Files:**
- Create: `internal/handlers/password_reset.go`
- Modify: `internal/handlers/email.go` (remove extracted methods)
- Delete: `internal/handlers/email.go` (should be empty or just utilities)

**Step 1: Create PasswordResetHandler**

Create `internal/handlers/password_reset.go`:

```go
package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/password"
	tokenSvc "github.com/tdawe1/translation-app/internal/service"
	"gorm.io/gorm"
)

// PasswordResetHandler handles password reset flow
type PasswordResetHandler struct {
	db          database.Database
	emailService *email.Service
	tokenStore   *tokenSvc.TokenService
}

// NewPasswordResetHandler creates a new password reset handler
func NewPasswordResetHandler(db database.Database, emailService *email.Service, tokenStore *tokenSvc.TokenService) *PasswordResetHandler {
	return &PasswordResetHandler{
		db:          db,
		emailService: emailService,
		tokenStore:   tokenStore,
	}
}

// SendPasswordResetRequest represents the request to send a password reset
type SendPasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// SendPasswordReset sends a password reset email
func (h *PasswordResetHandler) SendPasswordReset(c *fiber.Ctx) error {
	var req SendPasswordResetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Check if user exists (don't reveal if user doesn't exist)
	var user models.User
	err := h.db.Where("email = ?", req.Email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Don't reveal if user exists
			return c.JSON(fiber.Map{
				"message": "If an account exists, a password reset link has been sent",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
			"code":  "DATABASE_ERROR",
		})
	}

	// Create token
	token, err := h.tokenStore.CreatePasswordResetToken(req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create reset token",
			"code":  "TOKEN_CREATION_FAILED",
		})
	}

	// Send email
	if err := h.emailService.SendPasswordResetEmail(req.Email, token); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send password reset email. Please try again.",
			"code":  "EMAIL_SEND_FAILED",
		})
	}

	return c.JSON(fiber.Map{
		"message": "If an account exists, a password reset link has been sent",
	})
}

// ResetPasswordRequest represents the request to reset password
type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

// ResetPassword resets a user's password using a token
func (h *PasswordResetHandler) ResetPassword(c *fiber.Ctx) error {
	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate token
	result, err := h.tokenStore.ValidatePasswordResetToken(req.Token)
	if err != nil {
		code := err.Error()
		if code == "INVALID_TOKEN" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Invalid or expired token",
				"code":  code,
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
			"code":  code,
		})
	}

	// Find user
	var user models.User
	err = h.db.Where("email = ?", result.Email).First(&user).Error
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
			"code":  "USER_NOT_FOUND",
		})
	}

	// Hash new password
	hashedPassword, err := password.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
			"code":  "PASSWORD_HASH_FAILED",
		})
	}

	// Update password
	user.PasswordHash = hashedPassword
	if err := h.db.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update password",
			"code":  "UPDATE_FAILED",
		})
	}

	// Mark token as used
	if err := h.tokenStore.MarkPasswordResetTokenUsed(req.Token); err != nil {
		fmt.Printf("Failed to mark reset token as used: %v\n", err)
	}

	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}
```

**Step 2: Delete email.go or keep only utilities**

The file should now be empty or only contain `generateEmailSecureToken` which can be moved to `internal/service/token_service.go`.

**Step 3: Build verification**

```bash
cd backend && go build ./cmd/server
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/handlers/password_reset.go internal/handlers/email.go
git commit -m "refactor(handlers): extract PasswordResetHandler, remove email.go"
```

---

## Task 5: Update Route Registration

**Why:** Routes need to be updated to use the new handler instances.

**Files:**
- Modify: `internal/routes/routes.go` (or wherever routes are registered)

**Step 1: Find route registration**

```bash
cd backend && grep -r "EmailHandler" internal/ cmd/
```

**Step 2: Update route registration**

Old pattern:
```go
emailHandler := handlers.NewEmailHandler(db, tokenService, emailService)
api.Post("/verify/send", emailHandler.SendVerificationEmail)
api.Post("/verify/token", emailHandler.VerifyEmail)
api.Post("/magic-link", emailHandler.SendMagicLink)
api.Post("/magic-link/verify", emailHandler.VerifyMagicLink)
api.Post("/password-reset", emailHandler.SendPasswordReset)
api.Post("/password-reset/confirm", emailHandler.ResetPassword)
```

New pattern:
```go
// Create shared token service
tokenStore := service.NewTokenService(db)

// Create specialized handlers
emailVerificationHandler := handlers.NewEmailVerificationHandler(db, emailService, tokenStore)
magicLinkHandler := handlers.NewMagicLinkHandler(db, tokenService, emailService, tokenStore)
passwordResetHandler := handlers.NewPasswordResetHandler(db, emailService, tokenStore)

// Register routes
api.Post("/verify/send", emailVerificationHandler.SendVerificationEmail)
api.Post("/verify/token", emailVerificationHandler.VerifyEmail)
api.Post("/magic-link", magicLinkHandler.SendMagicLink)
api.Post("/magic-link/verify", magicLinkHandler.VerifyMagicLink)
api.Post("/password-reset", passwordResetHandler.SendPasswordReset)
api.Post("/password-reset/confirm", passwordResetHandler.ResetPassword)
```

**Step 3: Build verification**

```bash
cd backend && go build ./cmd/server
```

Expected: Build succeeds

**Step 4: Commit**

```bash
git add internal/routes/
git commit -m "refactor(routes): update routes to use specialized handlers"
```

---

## Task 6: Verify All Tests Pass

**Step 1: Run handler tests**

```bash
cd backend && go test ./internal/handlers/... -v
```

Expected: All PASS (or adjust tests as needed)

**Step 2: Run service tests**

```bash
cd backend && go test ./internal/service/... -v
```

Expected: All PASS

**Step 3: Full build**

```bash
cd backend && go build ./cmd/server
```

Expected: Build succeeds

**Step 4: Final commit**

```bash
git add -A
git commit -m "refactor(handlers): complete email handler split - 504 lines → focused handlers"
```

---

## Success Criteria

- [ ] `internal/handlers/email.go` removed or reduced to <50 lines
- [ ] Three new handler files created:
  - `internal/handlers/email_verification.go`
  - `internal/handlers/magic_link.go`
  - `internal/handlers/password_reset.go`
- [ ] `internal/service/token_service.go` created with shared token logic
- [ ] All routes updated and working
- [ ] All tests pass
- [ ] Build succeeds
