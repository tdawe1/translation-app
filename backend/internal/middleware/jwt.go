package middleware

import (
	"errors"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrMissingToken is returned when no token is provided
	ErrMissingToken = errors.New("missing authorization token")

	// ErrInvalidToken is returned when the token is invalid
	ErrInvalidToken = errors.New("invalid authorization token")
)

// JWTConfig holds JWT middleware configuration
type JWTConfig struct {
	Secret         string
	Lookup        string
	AuthScheme    string
	ContextKey     string
	 ErrorHandler  fiber.ErrorHandler
}

// jwtConfig is the default configuration
var jwtConfig = &JWTConfig{
	Secret:      os.Getenv("JWT_SECRET"),
	Lookup:      "header:Authorization",
	AuthScheme:  "Bearer",
	ContextKey:  "user",
}

// NewJWTConfig creates a new JWT config with options
func NewJWTConfig(options ...func(*JWTConfig)) *JWTConfig {
	config := &JWTConfig{
		Secret:     os.Getenv("JWT_SECRET"),
		Lookup:     "header:Authorization",
		AuthScheme: "Bearer",
		ContextKey:  "user",
	}

	for _, option := range options {
		option(config)
	}

	// Set default secret if empty
	if config.Secret == "" {
		config.Secret = "change-this-secret-in-production"
	}

	return config
}

// JWTValidator returns a Fiber middleware that validates JWT tokens
func JWTValidator(config *JWTConfig) fiber.Handler {
	if config == nil {
		config = jwtConfig
	}

	// Extract parts from lookup string
	parts := strings.Split(config.Lookup, ":")
	extractor := extractTokenFromHeader

	switch parts[0] {
	case "header":
		if len(parts) < 2 {
			break
		}
		switch parts[1] {
		case "Authorization":
			extractor = extractTokenFromHeader
		case "Cookie":
			extractor = extractTokenFromCookie
		}
	case "query":
		extractor = extractTokenFromQuery
	}

	return func(c *fiber.Ctx) error {
		token, err := extractor(c, config)
		if err != nil {
			if config.ErrorHandler != nil {
				return config.ErrorHandler(c, err)
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
				"code":  "MISSING_TOKEN",
			})
		}

		// Validate token
		claims, err := validateToken(token, config.Secret)
		if err != nil {
			if config.ErrorHandler != nil {
				return config.ErrorHandler(c, err)
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
				"code":  "INVALID_TOKEN",
			})
		}

		// Store user info in context
		c.Locals(config.ContextKey, claims)
		return c.Next()
	}
}

// extractTokenFromHeader extracts token from Authorization header
func extractTokenFromHeader(c *fiber.Ctx, config *JWTConfig) (string, error) {
	auth := c.Get("Authorization")
	if auth == "" {
		return "", ErrMissingToken
	}

	// Parse "Bearer <token>"
	parts := strings.Split(auth, " ")
	if len(parts) != 2 || parts[0] != config.AuthScheme {
		return "", ErrInvalidToken
	}

	return parts[1], nil
}

// extractTokenFromCookie extracts token from cookie
func extractTokenFromCookie(c *fiber.Ctx, config *JWTConfig) (string, error) {
	token := c.Cookies("session_token")
	if token == "" {
		return "", ErrMissingToken
	}
	return token, nil
}

// extractTokenFromQuery extracts token from query parameter
func extractTokenFromQuery(c *fiber.Ctx, config *JWTConfig) (string, error) {
	token := c.Query("token", "")
	if token == "" {
		return "", ErrMissingToken
	}
	// Remove "Bearer " prefix if present
	if strings.HasPrefix(token, config.AuthScheme+" ") {
		token = strings.TrimPrefix(token, config.AuthScheme+" ")
	}
	return token, nil
}

// validateToken validates a JWT token and returns the claims
func validateToken(tokenString, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// GetUserID extracts user ID from context (after JWT middleware)
func GetUserID(c *fiber.Ctx) (string, bool) {
	claims := c.Locals("user")
	if claims == nil {
		return "", false
	}

	// BetterAuth uses "sub" for user ID
	userClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}

	if sub, ok := userClaims["sub"].(string); ok {
		return sub, true
	}

	return "", false
}

// WebSocketAuth returns a middleware that validates JWT from query parameter
// For WebSocket connections that can't use standard headers
func WebSocketAuth(config *JWTConfig) fiber.Handler {
	if config == nil {
		config = jwtConfig
	}

	// Override context key to store user_id as string
	cfg := &JWTConfig{
		Secret:     config.Secret,
		Lookup:     "query:token",
		AuthScheme: "Bearer",
		ContextKey:  "user_id",
	}

	return JWTValidator(cfg)
}
