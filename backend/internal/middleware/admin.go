package middleware

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// RequireAdmin is a middleware that ensures the user has admin role.
// Must be used after JWTValidator middleware.
// Returns 403 Forbidden if user is not an admin.
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get claims from context (set by JWTValidator)
		claims := c.Locals("user")
		if claims == nil {
			log.Printf("[Admin] Access denied: no claims in context")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
				"code":  "NOT_AUTHENTICATED",
			})
		}

		// Extract user role from claims
		// Handle multiple claim types that JWTValidator might store:
		// - jwt.MapClaims (from jwt-go library)
		// - map[string]interface{} (generic map)
		// - map[string]any (Go 1.18+ any type)
		var claimMap map[string]interface{}

		// Try jwt.MapClaims first (what JWTValidator actually stores)
		if jwtClaims, ok := claims.(jwt.MapClaims); ok {
			// Convert jwt.MapClaims to map[string]interface{}
			claimMap = make(map[string]interface{})
			for k, v := range jwtClaims {
				claimMap[k] = v
			}
		} else if stdMap, ok := claims.(map[string]interface{}); ok {
			claimMap = stdMap
		} else {
			// Try map[string]any for Go 1.18+ compatibility
			claimMapAny, okAny := claims.(map[string]any)
			if !okAny {
				log.Printf("[Admin] Access denied: invalid claims type (got %T)", claims)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid authentication token",
					"code":  "INVALID_TOKEN",
				})
			}
			// Convert map[string]any to map[string]interface{}
			claimMap = make(map[string]interface{})
			for k, v := range claimMapAny {
				claimMap[k] = v
			}
		}

		role, ok := claimMap["role"].(string)
		if !ok || role != "admin" {
			userID := "unknown"
			if sub, ok := claimMap["sub"].(string); ok {
				userID = sub
			}
			log.Printf("[Admin] Access denied: user %s is not admin (role=%s)", userID, role)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Admin access required",
				"code":  "FORBIDDEN",
			})
		}

		log.Printf("[Admin] Access granted for admin user")
		return c.Next()
	}
}
