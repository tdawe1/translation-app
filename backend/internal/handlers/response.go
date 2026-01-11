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
	// DefaultCookieExpiration is the default session token expiration
	DefaultCookieExpiration = 7 * 24 * time.Hour
)

// SessionConfig holds cookie configuration for session management.
// This ensures that SetSessionCookie and ClearSessionCookie use matching
// cookie attributes (domain, secure, sameSite), which is critical for
// proper cookie clearing in production environments.
type SessionConfig struct {
	Domain   string        // Cookie domain (empty for localhost, ".example.com" for prod)
	Secure   bool          // Whether to set the Secure flag (HTTPS only)
	SameSite string        // SameSite policy: "Lax", "Strict", or "None"
	Expires  time.Duration // Cookie expiration duration (for Set only, not Clear)
}

// DefaultSessionConfig returns a SessionConfig with development-friendly defaults.
// For production, use config-based values with proper domain and Secure=true.
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		Domain:   "",            // Current host only (localhost)
		Secure:   false,         // HTTP in development
		SameSite: "Lax",
		Expires:  DefaultCookieExpiration,
	}
}

// SessionConfigFromEnv creates a SessionConfig from environment-based values.
// Use this in production to ensure proper domain matching and security flags.
func SessionConfigFromEnv(domain string, secure bool, sameSite string) SessionConfig {
	return SessionConfig{
		Domain:   domain,
		Secure:   secure,
		SameSite: sameSite,
		Expires:  DefaultCookieExpiration,
	}
}

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

// SetSessionCookie sets the httpOnly session cookie with the given configuration.
// The config parameter ensures all cookie attributes (domain, secure, sameSite)
// are properly set and can be matched when clearing the cookie.
func SetSessionCookie(c *fiber.Ctx, token string, config SessionConfig) {
	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    token,
		Domain:   config.Domain,
		HTTPOnly: true,
		Secure:   config.Secure,
		SameSite: config.SameSite,
		Expires:  time.Now().Add(config.Expires),
	})
}

// SetSessionCookieWithDefaults is a convenience function that uses default session config.
// For development use only. In production, use SetSessionCookie with config from env.
func SetSessionCookieWithDefaults(c *fiber.Ctx, token string, secure bool) {
	config := DefaultSessionConfig()
	config.Secure = secure
	SetSessionCookie(c, token, config)
}

// ClearSessionCookie clears the session cookie.
// IMPORTANT: Must use the same Domain/Secure/SameSite values as when the cookie was set.
// For httpOnly cookies, we must set it with an expiration in the past.
// The config parameter ensures the cookie attributes match the original SetSessionCookie call.
func ClearSessionCookie(c *fiber.Ctx, config SessionConfig) {
	c.Cookie(&fiber.Cookie{
		Name:     CookieName,
		Value:    "",
		Domain:   config.Domain,
		HTTPOnly: true,
		Secure:   config.Secure,
		SameSite: config.SameSite,
		Expires:  time.Now().Add(-1 * time.Hour), // Set to past to ensure deletion
	})
}

// ClearSessionCookieWithDefaults is a convenience function that uses default session config.
// For development use only. In production, use ClearSessionCookie with config from env.
func ClearSessionCookieWithDefaults(c *fiber.Ctx, secure bool) {
	config := DefaultSessionConfig()
	config.Secure = secure
	ClearSessionCookie(c, config)
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
