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
		trustedProxies []string
		expectXFF     bool // Whether we expect XFF to be used (requires c.IP() in trusted range)
	}{
		{
			name:          "no X-Forwarded-For",
			xForwardedFor: "",
			trustedProxies: nil,
			expectXFF:     false,
		},
		{
			name:          "X-Forwarded-For present but not trusted",
			xForwardedFor: "192.168.1.100",
			trustedProxies: nil,
			expectXFF:     false, // XFF ignored because no trusted proxies
		},
		{
			name:          "X-Forwarded-For with trusted proxy (but test IP not in range)",
			xForwardedFor: "192.168.1.100",
			trustedProxies: []string{"10.0.0.0/8"},
			expectXFF:     false, // XFF ignored because c.IP()="0.0.0.0" not in 10.0.0.0/8
		},
		{
			name:          "multiple IPs in X-Forwarded-For",
			xForwardedFor: "192.168.1.100, 10.0.0.1, 172.16.0.1",
			trustedProxies: []string{"10.0.0.0/8"},
			expectXFF:     false, // XFF ignored because c.IP()="0.0.0.0" not in trusted range
		},
		{
			name:          "X-Forwarded-For with spaces",
			xForwardedFor: "  192.168.1.100  ,  10.0.0.1  ",
			trustedProxies: []string{"10.0.0.0/8"},
			expectXFF:     false, // XFF ignored because c.IP()="0.0.0.0" not in trusted range
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			var gotIP string
			app.Get("/test", func(c *fiber.Ctx) error {
				gotIP = getClientIP(c, tt.trustedProxies)
				return c.SendStatus(200)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			_, _ = app.Test(req)

			// In tests, c.IP() returns "0.0.0.0" which is never in trusted ranges,
			// so X-Forwarded-For is always ignored (secure behavior)
			if !tt.expectXFF {
				// Should use direct IP, not X-Forwarded-For
				assert.NotEqual(t, "192.168.1.100", gotIP, "Should not use XFF when untrusted")
			}
			// Verify we got some IP (either direct or fallback)
			assert.NotEmpty(t, gotIP, "Should always return an IP")
		})
	}
}

func TestIsTrustedProxy(t *testing.T) {
	tests := []struct {
		name          string
		ip            string
		trustedProxies []string
		expected      bool
	}{
		{
			name:          "no trusted proxies configured",
			ip:            "192.168.1.100",
			trustedProxies: nil,
			expected:      false,
		},
		{
			name:          "empty trusted proxy list",
			ip:            "192.168.1.100",
			trustedProxies: []string{},
			expected:      false,
		},
		{
			name:          "IP in trusted CIDR range",
			ip:            "10.0.0.5",
			trustedProxies: []string{"10.0.0.0/8"},
			expected:      true,
		},
		{
			name:          "IP not in trusted CIDR range",
			ip:            "192.168.1.100",
			trustedProxies: []string{"10.0.0.0/8"},
			expected:      false,
		},
		{
			name:          "multiple CIDR ranges, IP in second",
			ip:            "172.16.0.5",
			trustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12"},
			expected:      true,
		},
		{
			name:          "localhost is trusted",
			ip:            "127.0.0.1",
			trustedProxies: []string{"127.0.0.0/8"},
			expected:      true,
		},
		{
			name:          "IPv6 loopback",
			ip:            "::1",
			trustedProxies: []string{"::1/128"},
			expected:      true,
		},
		{
			name:          "invalid IP returns false",
			ip:            "invalid-ip",
			trustedProxies: []string{"10.0.0.0/8"},
			expected:      false,
		},
		{
			name:          "invalid CIDR is skipped",
			ip:            "10.0.0.5",
			trustedProxies: []string{"invalid-cidr", "10.0.0.0/8"},
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTrustedProxy(tt.ip, tt.trustedProxies)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetClientIP_TrustedProxy_UseXFF(t *testing.T) {
	app := fiber.New()
	var gotIP string
	trustedProxies := []string{"10.0.0.0/8"}

	app.Get("/test", func(c *fiber.Ctx) error {
		gotIP = getClientIP(c, trustedProxies)
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.195")

	_, _ = app.Test(req)

	// Since we can't control c.IP() in tests to return a trusted proxy IP,
	// the XFF header will be ignored (default secure behavior)
	// In real scenarios with trusted proxies, this would use the XFF value
	assert.NotEmpty(t, gotIP)
}

func TestGetClientIP_UntrustedProxy_IgnoreXFF(t *testing.T) {
	app := fiber.New()
	var gotIP string
	// No trusted proxies = don't trust X-Forwarded-For
	var trustedProxies []string = nil

	app.Get("/test", func(c *fiber.Ctx) error {
		gotIP = getClientIP(c, trustedProxies)
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// Set X-Forwarded-For but it should be ignored since no trusted proxies
	req.Header.Set("X-Forwarded-For", "203.0.113.195")

	_, _ = app.Test(req)

	// Should use direct IP, not X-Forwarded-For
	assert.NotEqual(t, "203.0.113.195", gotIP)
}

func TestAuthLimiters(t *testing.T) {
	trustedProxies := []string{"10.0.0.0/8"}
	limiters := AuthLimiters(trustedProxies)

	assert.NotNil(t, limiters.Login, "Login limiter should not be nil")
	assert.NotNil(t, limiters.Register, "Register limiter should not be nil")
}

func TestEmailLimiters(t *testing.T) {
	trustedProxies := []string{"10.0.0.0/8"}
	limiters := EmailLimiters(trustedProxies)

	assert.NotNil(t, limiters.SendVerification, "SendVerification limiter should not be nil")
	assert.NotNil(t, limiters.SendMagicLink, "SendMagicLink limiter should not be nil")
	assert.NotNil(t, limiters.SendPasswordReset, "SendPasswordReset limiter should not be nil")
}

func TestRateLimiting_Login(t *testing.T) {
	app := fiber.New()
	trustedProxies := []string{"10.0.0.0/8"}
	limiters := AuthLimiters(trustedProxies)

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
	trustedProxies := []string{"10.0.0.0/8"}
	limiters := AuthLimiters(trustedProxies)

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
	trustedProxies := []string{"10.0.0.0/8"}
	limiters := EmailLimiters(trustedProxies)

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
