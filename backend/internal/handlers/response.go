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

// OAuthAccountResponse represents a linked OAuth account
type OAuthAccountResponse struct {
	Provider  string `json:"provider"`           // 'google', 'github'
	CreatedAt string `json:"created_at"`         // When the account was linked
}

// UserResponse represents user data in API responses
type UserResponse struct {
	ID            string                 `json:"id"`
	Email         string                 `json:"email"`
	EmailVerified bool                   `json:"email_verified"`
	IsActive      bool                   `json:"is_active"`
	Provider      string                 `json:"provider,omitempty"`      // 'google', 'github', or empty
	OAuthAccounts []OAuthAccountResponse `json:"oauth_accounts,omitempty"` // Linked OAuth accounts
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
	// Convert OAuth accounts to response format
	oauthAccounts := make([]OAuthAccountResponse, len(user.OAuthAccounts))
	for i, oa := range user.OAuthAccounts {
		oauthAccounts[i] = OAuthAccountResponse{
			Provider:  oa.Provider,
			CreatedAt: oa.CreatedAt.Format(time.RFC3339),
		}
	}

	return UserResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		IsActive:      user.IsActive,
		Provider:      user.Provider,
		OAuthAccounts: oauthAccounts,
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
