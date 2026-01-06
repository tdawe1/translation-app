package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	apperrors "github.com/tdawe1/translation-app/internal/errors"
)

// ErrorResponse represents a standardized error response (duplicate of handlers.ErrorResponse to avoid import cycle)
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    apperrors.ErrorCode    `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// respondWithError sends an error response with the given error code and message
// (duplicate of handlers.RespondWithError to avoid import cycle)
func respondWithError(c *fiber.Ctx, status int, code apperrors.ErrorCode, message string) error {
	return c.Status(status).JSON(ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// parseUserID parses a UUID string and returns an error if invalid
// (duplicate of handlers.ParseUserID to avoid import cycle)
func parseUserID(userIDStr string) (uuid.UUID, error) {
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, err
	}
	return userUUID, nil
}

// AuthenticatedHandler is a handler that requires authentication.
// The userUUID parameter is automatically provided from the JWT token.
type AuthenticatedHandler func(c *fiber.Ctx, userUUID uuid.UUID) error

// RequireAuth wraps an AuthenticatedHandler with authentication checks.
// It extracts the user ID from JWT, validates it, and calls the handler with the UUID.
// Returns 401 if not authenticated, 400 if user ID is invalid.
func RequireAuth(h AuthenticatedHandler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user ID from JWT (set by JWT middleware)
		userID, ok := GetUserID(c)
		if !ok {
			return respondWithError(c, fiber.StatusUnauthorized,
				apperrors.ErrNotAuthenticated, "Not authenticated")
		}

		// Parse UUID
		userUUID, err := parseUserID(userID)
		if err != nil {
			return respondWithError(c, fiber.StatusBadRequest,
				apperrors.ErrInvalidUserID, "Invalid user ID")
		}

		// Call wrapped handler with userUUID
		return h(c, userUUID)
	}
}
