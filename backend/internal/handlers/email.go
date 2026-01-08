// Package handlers provides HTTP handlers for email verification
package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/password"
	"gorm.io/gorm"
)

// EmailHandler handles email verification requests
type EmailHandler struct {
	db           database.Database
	tokenService *auth.TokenService
	emailService *email.Service
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(db database.Database, tokenService *auth.TokenService, emailService *email.Service) *EmailHandler {
	return &EmailHandler{
		db:           db,
		tokenService: tokenService,
		emailService: emailService,
	}
}

// generateEmailSecureToken generates a secure random token for email verification
func generateEmailSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// SendVerificationRequest represents the request to send a verification email
type SendVerificationRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// SendVerificationEmail sends a verification email to the user
func (h *EmailHandler) SendVerificationEmail(c *fiber.Ctx) error {
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

	// Generate token
	token, err := generateEmailSecureToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
			"code":  "TOKEN_GENERATION_FAILED",
		})
	}

	// Delete any existing unused tokens for this email
	h.db.Where("email = ? AND used_at IS NULL", req.Email).Delete(&models.EmailVerificationToken{})

	// Create verification token
	verificationToken := models.EmailVerificationToken{
		Email:     req.Email,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := h.db.Create(&verificationToken).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create verification token",
			"code":  "TOKEN_CREATION_FAILED",
		})
	}

	// Send email (#014 fix - return error instead of silent failure)
	if err := h.emailService.SendVerificationEmail(req.Email, token); err != nil {
		fmt.Printf("Failed to send verification email: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send verification email. Please try again.",
			"code":  "EMAIL_SEND_FAILED",
		})
	}

	// #023 fix - Return expiration info
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
func (h *EmailHandler) VerifyEmail(c *fiber.Ctx) error {
	var req VerifyEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Find token
	var verificationToken models.EmailVerificationToken
	err := h.db.Where("token = ?", req.Token).First(&verificationToken).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Invalid or expired token",
				"code":  "INVALID_TOKEN",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
			"code":  "DATABASE_ERROR",
		})
	}

	// Check if expired
	if time.Now().After(verificationToken.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token has expired",
			"code":  "TOKEN_EXPIRED",
		})
	}

	// Check if already used
	if verificationToken.UsedAt != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token already used",
			"code":  "TOKEN_ALREADY_USED",
		})
	}

	// Find user and update
	var user models.User
	err = h.db.Where("email = ?", verificationToken.Email).First(&user).Error
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
	now := time.Now()
	verificationToken.UsedAt = &now
	h.db.Save(&verificationToken)

	return c.JSON(fiber.Map{
		"message": "Email verified successfully",
	})
}

// SendMagicLinkRequest represents the request to send a magic link
type SendMagicLinkRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// SendMagicLink sends a magic link for passwordless authentication
func (h *EmailHandler) SendMagicLink(c *fiber.Ctx) error {
	var req SendMagicLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Generate token
	token, err := generateEmailSecureToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
			"code":  "TOKEN_GENERATION_FAILED",
		})
	}

	// Delete any existing unused tokens for this email
	h.db.Where("email = ? AND used_at IS NULL", req.Email).Delete(&models.MagicLinkToken{})

	// Create magic link token
	magicLinkToken := models.MagicLinkToken{
		Email:     req.Email,
		Token:     token,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	if err := h.db.Create(&magicLinkToken).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create magic link token",
			"code":  "TOKEN_CREATION_FAILED",
		})
	}

	// Send email (#014 fix - return error instead of silent failure)
	if err := h.emailService.SendMagicLinkEmail(req.Email, token); err != nil {
		fmt.Printf("Failed to send magic link email: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send magic link email. Please try again.",
			"code":  "EMAIL_SEND_FAILED",
		})
	}

	// #023 fix - Return expiration info
	return c.JSON(fiber.Map{
		"message":            "Magic link sent to your email",
		"expires_at":         magicLinkToken.ExpiresAt.Format(time.RFC3339),
		"expires_in_minutes": int(time.Until(magicLinkToken.ExpiresAt).Minutes()),
	})
}

// VerifyMagicLink verifies a magic link and creates a session
func (h *EmailHandler) VerifyMagicLink(c *fiber.Ctx) error {
	var req VerifyEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Find token
	var magicLinkToken models.MagicLinkToken
	err := h.db.Where("token = ?", req.Token).First(&magicLinkToken).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Invalid or expired link",
				"code":  "INVALID_TOKEN",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
			"code":  "DATABASE_ERROR",
		})
	}

	// Check if expired
	if time.Now().After(magicLinkToken.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Link has expired",
			"code":  "TOKEN_EXPIRED",
		})
	}

	// Check if already used
	if magicLinkToken.UsedAt != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Link already used",
			"code":  "TOKEN_ALREADY_USED",
		})
	}

	// Find or create user
	var user models.User
	err = h.db.Where("email = ?", magicLinkToken.Email).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		// Create new user
		user = models.User{
			Email:         magicLinkToken.Email,
			EmailVerified: true, // Magic link implies verified email
			IsActive:      true,
		}
		if err := h.db.Create(&user).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create user",
				"code":  "USER_CREATION_FAILED",
			})
		}
		// #017 fix - Don't create watcher config in auth flow
		// Watcher resources will be created lazily on first access
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
			"code":  "DATABASE_ERROR",
		})
	}

	// Mark token as used
	now := time.Now()
	magicLinkToken.UsedAt = &now
	h.db.Save(&magicLinkToken)

	// Generate JWT token
	accessToken, err := h.tokenService.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate access token",
			"code":  "TOKEN_GENERATION_FAILED",
		})
	}

	// Set httpOnly cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    accessToken, // Using access token for simplicity (refresh tokens are stored in DB)
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

// SendPasswordReset sends a password reset email
func (h *EmailHandler) SendPasswordReset(c *fiber.Ctx) error {
	var req SendMagicLinkRequest
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

	// Generate token
	token, err := generateEmailSecureToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
			"code":  "TOKEN_GENERATION_FAILED",
		})
	}

	// Delete any existing unused tokens for this email
	h.db.Where("email = ? AND used_at IS NULL", req.Email).Delete(&models.PasswordResetToken{})

	// Create reset token
	resetToken := models.PasswordResetToken{
		Email:     req.Email,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := h.db.Create(&resetToken).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create reset token",
			"code":  "TOKEN_CREATION_FAILED",
		})
	}

	// Send email (#014 fix - return error instead of silent failure)
	if err := h.emailService.SendPasswordResetEmail(req.Email, token); err != nil {
		fmt.Printf("Failed to send reset email: %v\n", err)
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
func (h *EmailHandler) ResetPassword(c *fiber.Ctx) error {
	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Find token
	var resetToken models.PasswordResetToken
	err := h.db.Where("token = ?", req.Token).First(&resetToken).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Invalid or expired token",
				"code":  "INVALID_TOKEN",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
			"code":  "DATABASE_ERROR",
		})
	}

	// Check if expired
	if time.Now().After(resetToken.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token has expired",
			"code":  "TOKEN_EXPIRED",
		})
	}

	// Check if already used
	if resetToken.UsedAt != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token already used",
			"code":  "TOKEN_ALREADY_USED",
		})
	}

	// Find user
	var user models.User
	err = h.db.Where("email = ?", resetToken.Email).First(&user).Error
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
	now := time.Now()
	resetToken.UsedAt = &now
	h.db.Save(&resetToken)

	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}
