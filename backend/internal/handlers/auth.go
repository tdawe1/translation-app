package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/tdawe1/translation-app/internal/auth"
	apperrors "github.com/tdawe1/translation-app/internal/errors"
	"github.com/tdawe1/translation-app/internal/middleware"
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
	userService *auth.UserService
	secureCookie bool
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userService *auth.UserService, secureCookie bool) *AuthHandler {
	return &AuthHandler{
		userService:  userService,
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

// Register handles user registration
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid request body")
	}

	// Validate input
	if len(req.Password) < 8 {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrWeakPassword, "Password must be at least 8 characters")
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
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return RespondWithError(c, fiber.StatusUnauthorized, apperrors.ErrNotAuthenticated, "Not authenticated")
	}

	userUUID, err := ParseUserID(userID)
	if err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidUserID, "Invalid user ID")
	}

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
