package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/password"
	"github.com/tdawe1/translation-app/internal/service"
	"github.com/tdawe1/translation-app/internal/validation"
	"gorm.io/gorm"
)

// PasswordResetHandler handles password reset requests
type PasswordResetHandler struct {
	db           database.Database
	emailService *email.Service
	tokenService *service.TokenService
}

// NewPasswordResetHandler creates a new password reset handler
func NewPasswordResetHandler(db database.Database, emailService *email.Service, tokenSvc *service.TokenService) *PasswordResetHandler {
	return &PasswordResetHandler{
		db:           db,
		emailService: emailService,
		tokenService: tokenSvc,
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
			// Don't reveal if user exists (security best practice)
			return c.JSON(fiber.Map{
				"message": "If an account exists, a password reset link has been sent",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
			"code":  "DATABASE_ERROR",
		})
	}

	// Create password reset token using TokenService
	tokenStr, err := h.tokenService.CreatePasswordResetToken(req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create reset token",
			"code":  "TOKEN_CREATION_FAILED",
		})
	}

	// Send email
	if err := h.emailService.SendPasswordResetEmail(req.Email, tokenStr); err != nil {
		fmt.Printf("Failed to send reset email: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send password reset email. Please try again.",
			"code":  "EMAIL_SEND_FAILED",
		})
	}

	// Return success message (same whether user exists or not, for security)
	return c.JSON(fiber.Map{
		"message": "If an account exists, a password reset link has been sent",
	})
}

// ResetPasswordWithTokenRequest represents the request to reset password with token
type ResetPasswordWithTokenRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

// ResetPassword resets a user's password using a token
func (h *PasswordResetHandler) ResetPassword(c *fiber.Ctx) error {
	var req ResetPasswordWithTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate password strength
	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 8 characters long",
			"code":  "WEAK_PASSWORD",
		})
	}

	// Validate token using TokenService
	tokenResult, err := h.tokenService.ValidatePasswordResetToken(req.Token)
	if err != nil {
		code := "INVALID_TOKEN"
		if err.Error() == "TOKEN_EXPIRED" {
			code = "TOKEN_EXPIRED"
		} else if err.Error() == "TOKEN_ALREADY_USED" {
			code = "TOKEN_ALREADY_USED"
		}
		statusCode := fiber.StatusBadRequest
		if code == "INVALID_TOKEN" {
			statusCode = fiber.StatusNotFound
		}
		return c.Status(statusCode).JSON(fiber.Map{
			"error": "Invalid or expired token",
			"code":  code,
		})
	}

	// Find user
	var user models.User
	err = h.db.Where("email = ?", tokenResult.Email).First(&user).Error
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
	if err := h.tokenService.MarkPasswordResetTokenUsed(req.Token); err != nil {
		fmt.Printf("Failed to mark token as used: %v\n", err)
	}

	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}
