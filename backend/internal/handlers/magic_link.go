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
	"gorm.io/gorm"
)

// MagicLinkHandler handles magic link authentication requests
type MagicLinkHandler struct {
	db              database.Database
	tokenAuthService *auth.TokenService
	tokenService    *service.TokenService
	emailService    *email.Service
	cookieSecure    bool
	frontendURL     string // Frontend URL for redirects after successful auth
}

// NewMagicLinkHandler creates a new magic link handler
func NewMagicLinkHandler(db database.Database, tokenAuthService *auth.TokenService, emailService *email.Service, tokenSvc *service.TokenService, cookieSecure bool, frontendURL string) *MagicLinkHandler {
	return &MagicLinkHandler{
		db:              db,
		tokenAuthService: tokenAuthService,
		tokenService:    tokenSvc,
		emailService:    emailService,
		cookieSecure:    cookieSecure,
		frontendURL:     frontendURL,
	}
}

// SendMagicLinkAuthRequest represents the request to send a magic link
type SendMagicLinkAuthRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// SendMagicLink sends a magic link for passwordless authentication
func (h *MagicLinkHandler) SendMagicLink(c *fiber.Ctx) error {
	var req SendMagicLinkAuthRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Create magic link token using TokenService
	tokenStr, err := h.tokenService.CreateMagicLinkToken(req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create magic link token",
			"code":  "TOKEN_CREATION_FAILED",
		})
	}

	// Get token expiry time for response
	tokenResult, err := h.tokenService.ValidateMagicLinkToken(tokenStr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to validate token",
			"code":  "TOKEN_VALIDATION_FAILED",
		})
	}

	// Send email
	if err := h.emailService.SendMagicLinkEmail(req.Email, tokenStr); err != nil {
		fmt.Printf("Failed to send magic link email: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send magic link email. Please try again.",
			"code":  "EMAIL_SEND_FAILED",
		})
	}

	// Return expiration info
	return c.JSON(fiber.Map{
		"message":            "Magic link sent to your email",
		"expires_at":         tokenResult.ExpiresAt.Format(time.RFC3339),
		"expires_in_minutes": int(time.Until(tokenResult.ExpiresAt).Minutes()),
	})
}

// VerifyMagicLinkRequest represents the request to verify a magic link
type VerifyMagicLinkRequest struct {
	Token string `json:"token" validate:"required"`
}

// VerifyMagicLink verifies a magic link and creates a session
func (h *MagicLinkHandler) VerifyMagicLink(c *fiber.Ctx) error {
	// Support both POST body and query parameter
	token := c.Query("token")
	if token == "" {
		var req VerifyMagicLinkRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
				"code":  "INVALID_REQUEST",
			})
		}
		token = req.Token
	}

	// Validate token using TokenService
	tokenResult, err := h.tokenService.ValidateMagicLinkToken(token)
	if err != nil {
		code := "INVALID_TOKEN"
		if err.Error() == "TOKEN_EXPIRED" {
			code = "TOKEN_EXPIRED"
		} else if err.Error() == "TOKEN_ALREADY_USED" {
			code = "TOKEN_ALREADY_USED"
		}
		// For GET requests (redirect flow), return error response
		// For POST requests, also return error response
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired link",
			"code":  code,
		})
	}

	// Find or create user
	var user models.User
	err = h.db.Where("email = ?", tokenResult.Email).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		// Create new user with email verified (magic link implies verified email)
		user = models.User{
			Email:         tokenResult.Email,
			EmailVerified: true,
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

	// Mark user's email as verified if it wasn't already
	// This handles the case where a user with an unverified email uses magic link
	if !user.EmailVerified {
		user.EmailVerified = true
		if err := h.db.Save(&user).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update user",
				"code":  "USER_UPDATE_FAILED",
			})
		}
	}

	// Mark token as used
	if err := h.tokenService.MarkMagicLinkTokenUsed(token); err != nil {
		fmt.Printf("Failed to mark token as used: %v\n", err)
	}

	// Generate JWT token
	accessToken, err := h.tokenAuthService.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate access token",
			"code":  "TOKEN_GENERATION_FAILED",
		})
	}

	// Check if this is a GET request (redirect flow from email link)
	if c.Method() == "GET" {
		// Set httpOnly cookie and redirect to frontend
		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    accessToken,
			HTTPOnly: true,
			Secure:   h.cookieSecure,
			SameSite: "lax",
			MaxAge:   7 * 24 * 60 * 60, // 7 days
		})

		// Redirect to frontend with success indicator
		frontendURL := h.frontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:3001/dashboard?auth=success"
	} else {
		frontendURL = frontendURL + "/dashboard?auth=success"
	}
	return c.Redirect(frontendURL)
	}

	// POST flow: return JSON response with token
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    accessToken,
		HTTPOnly: true,
		Secure:   h.cookieSecure,
		SameSite: "lax",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})

	return c.JSON(fiber.Map{
		"access_token": accessToken,
		"user":         user,
	})
}
