package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"

	"github.com/tdawe1/translation-app/internal/auth"
	apperrors "github.com/tdawe1/translation-app/internal/errors"
	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/password"
)

// getAPIError safely converts an error to *apperrors.APIError.
// Returns nil if the error is not of the correct type.
func getAPIError(err error) *apperrors.APIError {
	if err == nil {
		return nil
	}
	apiErr, ok := err.(*apperrors.APIError)
	if !ok {
		return nil
	}
	return apiErr
}

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	userService  *auth.UserService
	tokenService *auth.TokenService
	emailService *email.Service
	redis        *redis.Client
	secureCookie bool
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userService *auth.UserService, tokenService *auth.TokenService, emailService *email.Service, redis *redis.Client, secureCookie bool) *AuthHandler {
	return &AuthHandler{
		userService:  userService,
		tokenService: tokenService,
		emailService: emailService,
		redis:        redis,
		secureCookie: secureCookie,
	}
}

// RegisterRequest represents registration input
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents login input
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// MagicLinkRequest represents magic link input
type MagicLinkRequest struct {
	Email string `json:"email"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid request body")
	}

	// Validate password strength (P2 fix - enforces 12+ chars, upper, lower, digit, special)
	if err := password.ValidateStrength(req.Password); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrWeakPassword, err.Error())
	}

	result, apiErr := h.userService.Register(auth.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	})

	if apiErr != nil {
		errObj := getAPIError(apiErr)
		if errObj == nil {
			return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Internal error")
		}
		status := h.statusCodeForError(errObj.Code)
		return RespondWithAPIError(c, status, errObj)
	}

	// Set httpOnly cookie
	SetSessionCookie(c, result.AccessToken, h.secureCookie)

	return c.Status(fiber.StatusCreated).JSON(AuthResponse{
		AccessToken: result.AccessToken,
		User:        UserToResponse(result.User),
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid request body")
	}

	result, apiErr := h.userService.Login(auth.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})

	if apiErr != nil {
		errObj := getAPIError(apiErr)
		if errObj == nil {
			return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Internal error")
		}
		status := h.statusCodeForError(errObj.Code)
		return RespondWithAPIError(c, status, errObj)
	}

	// Set httpOnly cookie
	SetSessionCookie(c, result.AccessToken, h.secureCookie)

	return c.JSON(AuthResponse{
		AccessToken: result.AccessToken,
		User:        UserToResponse(result.User),
	})
}

// GetMe returns current user info
// NOTE: This route is already wrapped with JWTValidator middleware in main.go,
// so we directly extract the user ID from context rather than using RequireAuth.
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	// Extract user ID from JWT claims (set by JWTValidator middleware)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return RespondWithError(c, fiber.StatusUnauthorized,
			apperrors.ErrNotAuthenticated, "Not authenticated")
	}

	// Parse UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return RespondWithError(c, fiber.StatusBadRequest,
			apperrors.ErrInvalidUserID, "Invalid user ID")
	}

	// Original getMeLogic logic
	user, apiErr := h.userService.GetUserByID(userUUID)
	if apiErr != nil {
		errObj := getAPIError(apiErr)
		if errObj == nil {
			return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Internal error")
		}
		status := h.statusCodeForError(errObj.Code)
		return RespondWithAPIError(c, status, errObj)
	}

	return c.JSON(UserToResponse(user))
}

// getMeLogic contains the actual GetMe logic after auth is verified
// DEPRECATED: Logic moved into GetMe since JWTValidator is applied at route level
func (h *AuthHandler) getMeLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	user, apiErr := h.userService.GetUserByID(userUUID)
	if apiErr != nil {
		errObj := getAPIError(apiErr)
		if errObj == nil {
			return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Internal error")
		}
		status := h.statusCodeForError(errObj.Code)
		return RespondWithAPIError(c, status, errObj)
	}

	return c.JSON(UserToResponse(user))
}

// Logout handles logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	ClearSessionCookie(c)
	return c.SendStatus(fiber.StatusNoContent)
}

// statusCodeForError maps error codes to HTTP status codes
func (h *AuthHandler) statusCodeForError(code apperrors.ErrorCode) int {
	switch code {
	case apperrors.ErrInvalidRequest, apperrors.ErrWeakPassword, apperrors.ErrInvalidUserID:
		return fiber.StatusBadRequest
	case apperrors.ErrUserExists:
		return fiber.StatusConflict
	case apperrors.ErrInvalidCredentials:
		return fiber.StatusUnauthorized
	case apperrors.ErrInactiveUser:
		return fiber.StatusForbidden
	case apperrors.ErrUserNotFound:
		return fiber.StatusNotFound
	default:
		return fiber.StatusInternalServerError
	}
}

// RequestMagicLink initiates magic link authentication
// POST /api/v1/auth/magic-link
func (h *AuthHandler) RequestMagicLink(c *fiber.Ctx) error {
	var req MagicLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid request body")
	}

	// Validate email
	if req.Email == "" {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Email is required")
	}

	// Check if email service is configured
	if !h.emailService.IsEnabled() {
		return RespondWithError(c, fiber.StatusServiceUnavailable, apperrors.ErrInternal, "Email service not available")
	}

	// Check if user exists (don't reveal if user doesn't exist for security)
	_, apiErr := h.userService.GetUserByEmail(req.Email)
	if apiErr != nil {
		// Don't reveal whether user exists - always return success
		log.Printf("Magic link requested for non-existent email: %s", req.Email)
		return c.JSON(fiber.Map{"message": "If an account exists, a magic link has been sent"})
	}

	// Generate secure token
	token := generateSecureToken()

	// Store token in Redis with 15-minute expiry
	ctx := context.Background()
	key := fmt.Sprintf("magiclink:%s", token)
	if err := h.redis.Set(ctx, key, req.Email, 15*time.Minute).Err(); err != nil {
		log.Printf("Failed to store magic link token: %v", err)
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Failed to generate magic link")
	}

	// Send email
	if err := h.emailService.SendMagicLink(req.Email, token); err != nil {
		log.Printf("Failed to send magic link email: %v", err)
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Failed to send email")
	}

	return c.JSON(fiber.Map{"message": "If an account exists, a magic link has been sent"})
}

// VerifyMagicLink verifies a magic link token and creates a session
// GET /api/v1/auth/verify?token=xxx
func (h *AuthHandler) VerifyMagicLink(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Token is required")
	}

	ctx := context.Background()
	key := fmt.Sprintf("magiclink:%s", token)

	// Atomic validate and consume token (GETDEL prevents reuse)
	email, err := h.redis.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidCredentials, "Invalid or expired token")
	} else if err != nil {
		log.Printf("Failed to verify magic link: %v", err)
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Failed to verify token")
	}

	// Get user by email
	user, apiErr := h.userService.GetUserByEmail(email)
	if apiErr != nil {
		errObj := getAPIError(apiErr)
		if errObj == nil {
			return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Internal error")
		}
		status := h.statusCodeForError(errObj.Code)
		return RespondWithAPIError(c, status, errObj)
	}

	// Generate session token
	accessToken, err := h.tokenService.GenerateAccessToken(user.ID)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrTokenError, "Failed to create session")
	}

	// Set session cookie
	SetSessionCookie(c, accessToken, h.secureCookie)

	// Redirect to dashboard
	return c.Redirect("/dashboard", fiber.StatusTemporaryRedirect)
}

// ChangePasswordRequest represents password change input
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangePassword handles password changes for authenticated users
// PUT /api/v1/me/password
// NOTE: This route is already wrapped with JWTValidator middleware in main.go,
// so we directly extract the user ID from context rather than using RequireAuth.
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	// Extract user ID from JWT claims (set by JWTValidator middleware)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return RespondWithError(c, fiber.StatusUnauthorized,
			apperrors.ErrNotAuthenticated, "Not authenticated")
	}

	// Parse UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return RespondWithError(c, fiber.StatusBadRequest,
			apperrors.ErrInvalidUserID, "Invalid user ID")
	}

	// Parse request body
	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid request body")
	}

	// Validate password strength (P2 fix - enforces 12+ chars, upper, lower, digit, special)
	if err := password.ValidateStrength(req.NewPassword); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrWeakPassword, err.Error())
	}

	// Change password via service
	apiErr := h.userService.ChangePassword(auth.ChangePasswordRequest{
		UserID:      userUUID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})

	if apiErr != nil {
		errObj := getAPIError(apiErr)
		if errObj == nil {
			return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Internal error")
		}
		status := h.statusCodeForError(errObj.Code)
		return RespondWithAPIError(c, status, errObj)
	}

	return c.JSON(fiber.Map{"message": "Password updated successfully"})
}

// generateSecureToken creates a cryptographically random token
func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to UUID if crypto rand fails
		return uuid.New().String()
	}
	return base64.URLEncoding.EncodeToString(b)
}

