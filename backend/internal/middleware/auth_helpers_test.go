package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestRequireAuth_WithoutUser(t *testing.T) {
	app := fiber.New()

	// Handler that uses RequireAuth
	app.Get("/test", RequireAuth(func(c *fiber.Ctx, userUUID uuid.UUID) error {
		return c.SendString("Success: " + userUUID.String())
	}))

	// Make request without auth
	req, _ := http.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestRequireAuth_WithValidUser(t *testing.T) {
	// Mock authenticated user
	testUUID := uuid.New()

	// Create a test app with middleware that sets the user
	testApp := fiber.New()
	testApp.Use(func(c *fiber.Ctx) error {
		// Mock JWT middleware - set user claims
		c.Locals("user", map[string]interface{}{
			"sub": testUUID.String(),
		})
		return c.Next()
	})

	testApp.Get("/test", RequireAuth(func(c *fiber.Ctx, userUUID uuid.UUID) error {
		if userUUID != testUUID {
			t.Errorf("Expected %s, got %s", testUUID, userUUID)
		}
		return c.SendString("Success")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := testApp.Test(req)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestRequireAuth_WithInvalidUserID(t *testing.T) {
	app := fiber.New()

	// Middleware that sets invalid user ID
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", map[string]interface{}{
			"sub": "not-a-valid-uuid",
		})
		return c.Next()
	})

	app.Get("/test", RequireAuth(func(c *fiber.Ctx, userUUID uuid.UUID) error {
		return c.SendString("Success")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected 400, got %d", resp.StatusCode)
	}
}
