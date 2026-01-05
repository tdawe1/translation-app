package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		xForwardedFor string
		expectedIP    string
	}{
		{
			name:          "no X-Forwarded-For",
			xForwardedFor: "",
			expectedIP:    "0.0.0.0", // Fiber default for test requests
		},
		{
			name:          "single IP in X-Forwarded-For",
			xForwardedFor: "192.168.1.100",
			expectedIP:    "192.168.1.100",
		},
		{
			name:          "multiple IPs in X-Forwarded-For",
			xForwardedFor: "192.168.1.100, 10.0.0.1, 172.16.0.1",
			expectedIP:    "192.168.1.100",
		},
		{
			name:          "X-Forwarded-For with spaces",
			xForwardedFor: "  192.168.1.100  ,  10.0.0.1  ",
			expectedIP:    "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			var gotIP string
			app.Get("/test", func(c *fiber.Ctx) error {
				gotIP = getClientIP(c)
				return c.SendStatus(200)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			_, _ = app.Test(req)

			// For the no header case, we expect the fallback IP
			if tt.xForwardedFor == "" {
				assert.Contains(t, []string{"0.0.0.0", ""}, gotIP)
			} else {
				assert.Equal(t, tt.expectedIP, gotIP)
			}
		})
	}
}

func TestAuthLimiters(t *testing.T) {
	limiters := AuthLimiters()

	assert.NotNil(t, limiters.Login, "Login limiter should not be nil")
	assert.NotNil(t, limiters.Register, "Register limiter should not be nil")
}

func TestEmailLimiters(t *testing.T) {
	limiters := EmailLimiters()

	assert.NotNil(t, limiters.SendVerification, "SendVerification limiter should not be nil")
	assert.NotNil(t, limiters.SendMagicLink, "SendMagicLink limiter should not be nil")
	assert.NotNil(t, limiters.SendPasswordReset, "SendPasswordReset limiter should not be nil")
}

func TestRateLimiting_Login(t *testing.T) {
	app := fiber.New()
	limiters := AuthLimiters()

	app.Post("/login", limiters.Login, func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	// First 10 requests should succeed
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, 200, resp.StatusCode, "Request %d should succeed", i+1)
	}

	// 11th request should be rate limited
	req := httptest.NewRequest("POST", "/login", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 429, resp.StatusCode, "Request should be rate limited")
}

func TestRateLimiting_Register(t *testing.T) {
	app := fiber.New()
	limiters := AuthLimiters()

	app.Post("/register", limiters.Register, func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/register", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, 200, resp.StatusCode, "Request %d should succeed", i+1)
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("POST", "/register", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 429, resp.StatusCode, "Request should be rate limited")
}

func TestRateLimiting_Email(t *testing.T) {
	app := fiber.New()
	limiters := EmailLimiters()

	app.Post("/send-email", limiters.SendVerification, func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/send-email", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, 200, resp.StatusCode, "Request %d should succeed", i+1)
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("POST", "/send-email", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, 429, resp.StatusCode, "Request should be rate limited")
}
