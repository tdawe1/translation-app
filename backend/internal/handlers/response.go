package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	apperrors "github.com/tdawe1/translation-app/internal/errors"
	"github.com/tdawe1/translation-app/internal/models"
)

const (
	// CookieName is the name of the session cookie
	CookieName = "session_token"
	// CookieDomain for production (empty for localhost)
	CookieDomain = ""
	// DefaultCookieExpiration is the default session token expiration
	DefaultCookieExpiration = 7 * 24 * time.Hour
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    apperrors.ErrorCode    `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// UserResponse represents user data in API responses
type UserResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	IsActive      bool   `json:"is_active"`
}

// AuthResponse represents a successful authentication response
type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	User        UserResponse `json:"user"`
}

// RespondWithError sends an error response with the given error code and message
func RespondWithError(c *fiber.Ctx, status int, code apperrors.ErrorCode, message string) error {
	return c.Status(status).JSON(ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// RespondWithAPIError sends an error response from an APIError
func RespondWithAPIError(c *fiber.Ctx, status int, apiErr *apperrors.APIError) error {
	return c.Status(status).JSON(ErrorResponse{
		Error:   apiErr.Message,
		Code:    apiErr.Code,
		Details: apiErr.Details,
	})
}

// SetSessionCookie sets the httpOnly session cookie
func SetSessionCookie(c *fiber.Ctx, token string, secure bool) {
	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    token,
		Domain:   CookieDomain,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: "Lax",
		Expires:  time.Now().Add(DefaultCookieExpiration),
	})
}

// ClearSessionCookie clears the session cookie
func ClearSessionCookie(c *fiber.Ctx) {
	c.ClearCookie(CookieName)
}

// UserToResponse converts a User model to UserResponse
func UserToResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		IsActive:      user.IsActive,
	}
}

// ParseUserID parses a UUID string and returns an error response if invalid
func ParseUserID(userIDStr string) (uuid.UUID, error) {
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, err
	}
	return userUUID, nil
}
