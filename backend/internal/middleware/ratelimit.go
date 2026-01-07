package middleware

import (
	"net"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// getClientIP extracts the real client IP with trusted proxy validation (P2 fix).
// Only trusts X-Forwarded-For from configured trusted proxy CIDR ranges.
// Falls back to direct connection IP if header is untrusted or missing.
func getClientIP(c *fiber.Ctx, trustedProxies []string) string {
	// Check X-Forwarded-For header (set by reverse proxies)
	xff := c.Get("X-Forwarded-For")
	if xff != "" {
		// Only trust X-Forwarded-For if the direct connection is from a trusted proxy
		remoteAddr := c.IP()
		if isTrustedProxy(remoteAddr, trustedProxies) {
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
		// If proxy is not trusted, ignore X-Forwarded-For entirely
	}

	// Fall back to direct IP
	return c.IP()
}

// isTrustedProxy checks if an IP address is within the trusted proxy CIDR ranges.
// Returns false if no trusted proxies are configured (secure by default).
func isTrustedProxy(ip string, trustedProxies []string) bool {
	if len(trustedProxies) == 0 {
		return false // Default: don't trust any proxy
	}

	netIP := net.ParseIP(ip)
	if netIP == nil {
		return false
	}

	for _, cidr := range trustedProxies {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue // Skip invalid CIDR ranges
		}
		if ipNet != nil && ipNet.Contains(netIP) {
			return true
		}
	}
	return false
}

// AuthLimiters returns rate limiters for auth endpoints
func AuthLimiters(trustedProxies []string) struct {
	Login    fiber.Handler
	Register fiber.Handler
} {
	// Login: 10 requests per minute per IP
	loginLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return getClientIP(c, trustedProxies)
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
			return getClientIP(c, trustedProxies)
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

// EmailLimiters returns rate limiters for email endpoints (#009 fix)
func EmailLimiters(trustedProxies []string) struct {
	SendVerification  fiber.Handler
	SendMagicLink     fiber.Handler
	SendPasswordReset fiber.Handler
} {
	// 3 emails per hour per IP address
	sendVerificationLimiter := limiter.New(limiter.Config{
		Max:        3,
		Expiration: 1 * time.Hour,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "verify:" + getClientIP(c, trustedProxies)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many verification emails requested. Please try again later.",
				"code":  "RATE_LIMITED",
			})
		},
	})

	sendMagicLinkLimiter := limiter.New(limiter.Config{
		Max:        3,
		Expiration: 1 * time.Hour,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "magic:" + getClientIP(c, trustedProxies)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many magic link requests. Please try again later.",
				"code":  "RATE_LIMITED",
			})
		},
	})

	sendPasswordResetLimiter := limiter.New(limiter.Config{
		Max:        3,
		Expiration: 1 * time.Hour,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "reset:" + getClientIP(c, trustedProxies)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(429).JSON(fiber.Map{
				"error": "Too many password reset requests. Please try again later.",
				"code":  "RATE_LIMITED",
			})
		},
	})

	return struct {
		SendVerification  fiber.Handler
		SendMagicLink     fiber.Handler
		SendPasswordReset fiber.Handler
	}{
		SendVerification:  sendVerificationLimiter,
		SendMagicLink:     sendMagicLinkLimiter,
		SendPasswordReset: sendPasswordResetLimiter,
	}
}
