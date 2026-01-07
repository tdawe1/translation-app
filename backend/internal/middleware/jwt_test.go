package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Set test environment before running tests
func init() {
	os.Setenv("TEST_ENV", "true")
}

func TestJWTConfig_InTestEnvironment(t *testing.T) {
	// This test should pass even without JWT_SECRET set
	app := fiber.New()

	// Add JWT middleware - it should not panic in tests
	app.Use(JWTValidator(nil))

	// Create a simple route
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// The middleware should not panic during test setup
	req, _ := http.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Expect unauthorized since we're not sending a token, but shouldn't panic
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Logf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestJWTConfig_WithTestSecret(t *testing.T) {
	app := fiber.New()

	// Use test secret to avoid env var requirement
	cfg := NewJWTConfig(WithSecret("test-secret-for-testing-only-32-chars-long!!"))
	app.Use(JWTValidator(cfg))

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Expect unauthorized since we're not sending a token
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestJWTConfig_WithValidToken(t *testing.T) {
	app := fiber.New()

	// Use test secret
	secret := "test-secret-for-testing-only-32-chars-long!!"
	cfg := NewJWTConfig(WithSecret(secret))
	app.Use(JWTValidator(cfg))

	// Create a protected route that returns user info
	app.Get("/protected", func(c *fiber.Ctx) error {
		user := c.Locals("user")
		if user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No user context",
			})
		}
		return c.JSON(fiber.Map{
			"user": user,
		})
	})

	// Generate a valid token for testing
	token := generateTestToken(secret, "user123")

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Should get 200 with valid token
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200 with valid token, got %d", resp.StatusCode)
	}
}

func TestJWTConfig_WithInvalidToken(t *testing.T) {
	app := fiber.New()

	cfg := NewJWTConfig(WithSecret("test-secret-for-testing-only-32-chars-long!!"))
	app.Use(JWTValidator(cfg))

	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("Protected")
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Should get 401 with invalid token
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401 with invalid token, got %d", resp.StatusCode)
	}
}

// Helper function to generate a test JWT token
func generateTestToken(secret, userID string) string {
	// Create a real JWT token for testing
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   "test@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		// This shouldn't happen in tests
		return "error-generating-token"
	}
	return tokenString
}

func TestJWTValidator_ExpiredToken(t *testing.T) {
	// Setup
	secret := "test-secret-for-testing-only-32-chars-long!!"
	cfg := NewJWTConfig(WithSecret(secret))

	// Create an expired token (exp set to past)
	expiredClaims := jwt.MapClaims{
		"user_id": "123e4567-e89b-12d3-a456-426614174000",
		"exp":     time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
		"iat":     time.Now().Add(-25 * time.Hour).Unix(),
	}

	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, err := expiredToken.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to generate expired token: %v", err)
	}

	// Create request with expired token
	app := fiber.New()
	app.Use(JWTValidator(cfg))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("protected")
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredTokenString)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	// Should reject expired token
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401 for expired token, got %d", resp.StatusCode)
	}

	// Parse and verify response body
	var result map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &result)

	// Verify error code
	if result["code"] != "INVALID_TOKEN" {
		t.Errorf("Expected INVALID_TOKEN code, got %v", result["code"])
	}
}
