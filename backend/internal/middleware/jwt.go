package middleware

import (
	"errors"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// validateJWTSecretOnStartup validates JWT_SECRET at application startup.
// P0-4 FIX: Fails fast in ALL environments if secret is missing or too short (except tests).
// Tests can bypass this by setting ENV=test or TEST_ENV=true.
func ValidateJWTSecretOnStartup() {
	env := os.Getenv("ENV")
	secret := os.Getenv("JWT_SECRET")

	// P0-4 FIX: Check for test environment first
	isTestEnv := env == "test" || os.Getenv("TEST_ENV") == "true"

	// In test environment, just log a warning and continue
	if isTestEnv {
		if secret == "" {
			log.Printf("INFO: JWT_SECRET not set in test environment, using test default")
		} else if len(secret) < minSecretLength {
			log.Printf("WARNING: JWT_SECRET too short for production use (%d chars)", len(secret))
		}
		return
	}

	// In all non-test environments, require JWT_SECRET
	if secret == "" {
		log.Fatal("FATAL: JWT_SECRET must be set and >= 32 characters. " +
			"Generate one with: openssl rand -hex 32")
	}

	if len(secret) < minSecretLength {
		log.Fatalf("FATAL: JWT_SECRET must be at least %d characters (256 bits for HS256). "+
			"Current length: %d. Please generate a stronger secret.", minSecretLength, len(secret))
	}

	log.Printf("INFO: JWT_SECRET validated (length: %d chars)", len(secret))
}

const (
	// minSecretLength is the minimum required length for JWT secret (256 bits for HS256)
	minSecretLength = 32
)

var (
	// ErrMissingToken is returned when no token is provided
	ErrMissingToken = errors.New("missing authorization token")

	// ErrInvalidToken is returned when the token is invalid
	ErrInvalidToken = errors.New("invalid authorization token")

	// ErrMissingSecret is returned when JWT secret is not set
	ErrMissingSecret = errors.New("JWT_SECRET environment variable is not set")

	// ErrSecretTooShort is returned when JWT secret is too short
	ErrSecretTooShort = errors.New("JWT_SECRET must be at least 32 characters")

	// jwtConfig is the default configuration (lazily initialized)
	jwtConfig   *JWTConfig
	jwtConfigMu sync.Mutex
)

// getJWTConfig returns the default JWT config, initializing it lazily
func getJWTConfig() *JWTConfig {
	jwtConfigMu.Lock()
	defer jwtConfigMu.Unlock()

	if jwtConfig != nil {
		return jwtConfig
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// In tests, use a test secret instead of fatal
		if isTestEnv() {
			secret = "test-secret-for-jwt-testing-purposes-only-32chars"
		} else if os.Getenv("ENV") == "development" || os.Getenv("ENV") == "" {
			// Use development default if ENV is development
			secret = "dev-secret-change-in-production-do-not-use-in-deployment"
		} else {
			log.Fatal("FATAL: JWT_SECRET environment variable is not set. " +
				"Authentication cannot function without a secure secret. " +
				"Please set JWT_SECRET to a random string of at least 32 characters.")
		}
	}

	jwtConfig = &JWTConfig{
		Secret:      secret,
		Lookup:      "cookie:session_token",
		AuthScheme:  "Bearer",
		ContextKey:  "user",
	}
	return jwtConfig
}

// validateJWTSecret validates and returns the JWT secret.
// P0-4 FIX: Require JWT_SECRET in all environments (except tests)
// In test environment (detected by testing flag), it returns a test secret instead of fatal.
func validateJWTSecret(secret string) string {
	if secret == "" {
		// In tests, use a test secret instead of fatal
		if isTestEnv() {
			return "test-secret-for-jwt-testing-purposes-only-32chars"
		}
		// P0-4 FIX: No hardcoded fallback - fail fast if JWT_SECRET is not set
		log.Fatal("FATAL: JWT_SECRET environment variable is not set. " +
			"Authentication cannot function without a secure secret. " +
			"Please set JWT_SECRET to a random string of at least 32 characters.")
	}
	if len(secret) < minSecretLength {
		// In tests, still validate length with warning
		if isTestEnv() {
			log.Printf("WARNING: JWT_SECRET is too short for production use (length: %d)", len(secret))
			return secret // Allow in tests for convenience
		}
		log.Fatalf("FATAL: JWT_SECRET must be at least %d characters (256 bits for HS256). "+
			"Current length: %d. Please generate a stronger secret.", minSecretLength, len(secret))
	}
	return secret
}

// isTestEnv detects if we're running in a test environment
func isTestEnv() bool {
	// P0-4 FIX: Check both TEST_ENV and ENV environment variables
	// This is consistent with the config package approach
	return os.Getenv("TEST_ENV") == "true" || os.Getenv("ENV") == "test"
}

// JWTConfig holds JWT middleware configuration
type JWTConfig struct {
	Secret        string
	Lookup       string
	AuthScheme   string
	ContextKey    string
	ErrorHandler fiber.ErrorHandler
}

// NewJWTConfig creates a new JWT config with options
func NewJWTConfig(options ...func(*JWTConfig)) *JWTConfig {
	config := &JWTConfig{
		Secret:     validateJWTSecret(os.Getenv("JWT_SECRET")),
		Lookup:     "cookie:session_token",
		AuthScheme: "Bearer",
		ContextKey:  "user",
	}

	for _, option := range options {
		option(config)
	}

	// Re-validate after options have been applied (in case an option modified Secret)
	// In test mode, we're more lenient
	if !isTestEnv() && (config.Secret == "" || len(config.Secret) < minSecretLength) {
		log.Fatalf("FATAL: JWT_SECRET validation failed after applying options. " +
			"Secret must be at least %d characters.", minSecretLength)
	}

	return config
}

// WithSecret sets a custom JWT secret (useful for testing)
func WithSecret(secret string) func(*JWTConfig) {
	return func(cfg *JWTConfig) {
		cfg.Secret = secret
	}
}

// JWTValidator returns a Fiber middleware that validates JWT tokens
// Supports multiple token sources: cookie, header, and query parameter
// Tries cookie first (httpOnly), then Authorization header (for sessionStorage)
func JWTValidator(config *JWTConfig) fiber.Handler {
	if config == nil {
		config = getJWTConfig()
	}

	return func(c *fiber.Ctx) error {
		var token string
		var err error

		// Try each source in order: cookie -> header -> query
		// This supports both httpOnly cookies (secure) and Authorization header (convenience)

		// 1. Try cookie first (most secure - httpOnly)
		token = c.Cookies("session_token")
		if token != "" {
			log.Printf("[JWT] Token extracted from cookie")
		} else {
			// 2. Try Authorization header (for sessionStorage compatibility)
			auth := c.Get("Authorization")
			if auth != "" {
				parts := strings.Split(auth, " ")
				if len(parts) == 2 && parts[0] == config.AuthScheme {
					token = parts[1]
					log.Printf("[JWT] Token extracted from Authorization header")
				}
			}
		}

		// 3. No token found in any source
		if token == "" {
			err = ErrMissingToken
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
			log.Printf("[JWT] Token validation failed: %v", err)
			if config.ErrorHandler != nil {
				return config.ErrorHandler(c, err)
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
				"code":  "INVALID_TOKEN",
			})
		}

		log.Printf("[JWT] Token validated successfully for user=%v", claims["sub"])
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

// WebSocketAuth returns a middleware that validates JWT from query parameter
// For WebSocket connections that can't use standard headers
func WebSocketAuth(config *JWTConfig) fiber.Handler {
	if config == nil {
		config = getJWTConfig()
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
