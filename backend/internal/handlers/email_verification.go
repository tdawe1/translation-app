package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/service"
	"github.com/tdawe1/translation-app/internal/validation"
	"gorm.io/gorm"
)

// EmailVerificationHandler handles email verification requests
type EmailVerificationHandler struct {
	db               database.Database
	tokenAuthService *auth.TokenService
	tokenService     *service.TokenService
	emailService     *email.Service
}

// NewEmailVerificationHandler creates a new email verification handler
func NewEmailVerificationHandler(db database.Database, tokenAuthService *auth.TokenService, emailService *email.Service, tokenSvc *service.TokenService) *EmailVerificationHandler {
	return &EmailVerificationHandler{
		db:               db,
		tokenAuthService: tokenAuthService,
		tokenService:     tokenSvc,
		emailService:     emailService,
	}
}

// SendVerificationEmailRequest represents the request to send a verification email
type SendVerificationEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// SendVerificationEmail sends a verification email to the user
func (h *EmailVerificationHandler) SendVerificationEmail(c *fiber.Ctx) error {
	var req SendVerificationEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate email format before any database operations (M-3 fix)
	if !validation.ValidateEmail(req.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid email format",
			"code":  "INVALID_EMAIL",
		})
	}

	// Check if user exists
	var user models.User
	err := h.db.Where("email = ?", req.Email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Don't reveal if user exists (security best practice - prevents account enumeration)
			return c.JSON(fiber.Map{
				"message": "If an account exists with this email, a verification link has been sent",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
			"code":  "DATABASE_ERROR",
		})
	}

	// Check if already verified
	if user.EmailVerified {
		return c.JSON(fiber.Map{
			"message": "If an account exists with this email, a verification link has been sent",
		})
	}

	// Create verification token using TokenService
	tokenStr, err := h.tokenService.CreateVerificationToken(req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create verification token",
			"code":  "TOKEN_CREATION_FAILED",
		})
	}

	// Get token expiry time for response
	tokenResult, err := h.tokenService.ValidateEmailVerificationToken(tokenStr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to validate token",
			"code":  "TOKEN_VALIDATION_FAILED",
		})
	}

	// Send email
	if err := h.emailService.SendVerificationEmail(req.Email, tokenStr); err != nil {
		fmt.Printf("Failed to send verification email: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send verification email. Please try again.",
			"code":  "EMAIL_SEND_FAILED",
		})
	}

	// Return expiration info
	return c.JSON(fiber.Map{
		"message":            "Verification email sent",
		"expires_at":         tokenResult.ExpiresAt.Format(time.RFC3339),
		"expires_in_minutes": int(time.Until(tokenResult.ExpiresAt).Minutes()),
	})
}

// VerifyEmailTokenRequest represents the request to verify an email
type VerifyEmailTokenRequest struct {
	Token string `json:"token" validate:"required"`
}

// VerifyEmail verifies an email using a token
func (h *EmailVerificationHandler) VerifyEmail(c *fiber.Ctx) error {
	var req VerifyEmailTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate token using TokenService
	tokenResult, err := h.tokenService.ValidateEmailVerificationToken(req.Token)
	if err != nil {
		code := "INVALID_TOKEN"
		if err.Error() == "TOKEN_EXPIRED" {
			code = "TOKEN_EXPIRED"
		} else if err.Error() == "TOKEN_ALREADY_USED" {
			code = "TOKEN_ALREADY_USED"
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired token",
			"code":  code,
		})
	}

	// Find user and update
	var user models.User
	err = h.db.Where("email = ?", tokenResult.Email).First(&user).Error
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
		fmt.Printf("Failed to mark token as used: %v\n", err)
	}

	return c.JSON(fiber.Map{
		"message": "Email verified successfully",
	})
}
