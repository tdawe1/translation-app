package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

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
			return c.IP() // TODO: Support X-Forwarded-For
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
			return c.IP()
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
