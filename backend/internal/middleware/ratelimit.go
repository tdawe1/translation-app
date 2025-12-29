package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// getClientIP extracts the real client IP from X-Forwarded-For if present,
// otherwise falls back to c.IP(). Handles comma-separated lists by
// taking the leftmost IP (the original client).
func getClientIP(c *fiber.Ctx) string {
	// Check X-Forwarded-For header (set by reverse proxies)
	xff := c.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2"
		// The leftmost IP is the original client
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Fall back to direct IP
	return c.IP()
}

// AuthLimiters returns rate limiters for auth endpoints
func AuthLimiters() struct {
	Login    fiber.Handler
	Register fiber.Handler
} {
	// Login: 10 requests per minute per IP
	loginLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return getClientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests",
				"code":  "RATE_LIMITED",
			})
		},
	})

	// Register: 3 requests per minute per IP
	registerLimiter := limiter.New(limiter.Config{
		Max:        3,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return getClientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many requests",
				"code":  "RATE_LIMITED",
			})
		},
	})

	return struct {
		Login    fiber.Handler
		Register fiber.Handler
	}{
		Login:    loginLimiter,
		Register: registerLimiter,
	}
}
