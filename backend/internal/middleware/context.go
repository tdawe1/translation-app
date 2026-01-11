package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// GetUserID extracts the authenticated user ID from the Fiber context.
// Returns the user ID and true if found, empty string and false otherwise.
// Handles both jwt.MapClaims and map[string]interface{} claim types.
func GetUserID(c *fiber.Ctx) (string, bool) {
	claims := c.Locals("user")
	if claims == nil {
		return "", false
	}

	// Handle both jwt.MapClaims and map[string]interface{}
	var sub string
	var ok bool

	if mapClaims, typeOK := claims.(map[string]interface{}); typeOK {
		sub, ok = mapClaims["sub"].(string)
	} else if jwtClaims, typeOK := claims.(jwt.MapClaims); typeOK {
		sub, ok = jwtClaims["sub"].(string)
	}

	if ok {
		return sub, true
	}
	return "", false
}

// GetUserRole extracts the user role from JWT claims in the Fiber context.
// Returns the role string and true if found, empty string and false otherwise.
// Handles both jwt.MapClaims and map[string]interface{} claim types.
func GetUserRole(c *fiber.Ctx) (string, bool) {
	claims := c.Locals("user")
	if claims == nil {
		return "", false
	}

	// Handle both jwt.MapClaims and map[string]interface{}
	var role string
	var ok bool

	if mapClaims, typeOK := claims.(map[string]interface{}); typeOK {
		role, ok = mapClaims["role"].(string)
	} else if jwtClaims, typeOK := claims.(jwt.MapClaims); typeOK {
		role, ok = jwtClaims["role"].(string)
	}

	if ok {
		return role, true
	}
	return "", false
}

// IsAdmin checks if the current user has admin role.
// Returns false if not authenticated or role is not admin.
// This provides a convenient helper for authorization checks.
func IsAdmin(c *fiber.Ctx) bool {
	role, ok := GetUserRole(c)
	return ok && role == "admin"
}
