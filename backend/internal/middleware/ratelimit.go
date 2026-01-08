package middleware

import (
	"log"
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

// RoleBasedLimiterConfig configures rate limits for different user roles
type RoleBasedLimiterConfig struct {
	// MaxRequestsPerMinute for regular users
	UserMax int
	// MaxRequestsPerMinute for admin users (much higher or 0 for unlimited)
	AdminMax int
	// Expiration duration for the rate limit window
	Expiration time.Duration
	// Trusted proxy CIDRs for IP extraction
	TrustedProxies []string
}

// DefaultRoleBasedConfig returns sensible defaults for role-based rate limiting
func DefaultRoleBasedConfig(trustedProxies []string) RoleBasedLimiterConfig {
	return RoleBasedLimiterConfig{
		UserMax:        60,  // 60 requests per minute for regular users
		AdminMax:       0,   // 0 = unlimited for admins
		Expiration:     1 * time.Minute,
		TrustedProxies: trustedProxies,
	}
}

// RoleBasedLimiter creates a rate limiter that checks user role from JWT claims.
// Admin users get much higher limits or bypass entirely (when AdminMax=0).
// Must be applied AFTER JWTValidator middleware.
// For unauthenticated requests, falls back to IP-based limiting.
func RoleBasedLimiter(config RoleBasedLimiterConfig) fiber.Handler {
	// User limiter (IP-based for authenticated users)
	userLimiter := limiter.New(limiter.Config{
		Max:        config.UserMax,
		Expiration: config.Expiration,
		KeyGenerator: func(c *fiber.Ctx) string {
			// For authenticated users, use user ID
			claims := c.Locals("user")
			if claims != nil {
				if claimMap, ok := claims.(map[string]interface{}); ok {
					if sub, ok := claimMap["sub"].(string); ok {
						return "user:" + sub
					}
				}
			}
			// Fallback to IP for unauthenticated
			return "ip:" + getClientIP(c, config.TrustedProxies)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded. Please try again later.",
				"code":  "RATE_LIMITED",
			})
		},
	})

	return func(c *fiber.Ctx) error {
		// Check if user is admin
		claims := c.Locals("user")
		if claims != nil {
			if claimMap, ok := claims.(map[string]interface{}); ok {
				if role, ok := claimMap["role"].(string); ok && role == "admin" {
					// Admin: check if unlimited or use high limit
					if config.AdminMax == 0 {
						// Unlimited - just log and continue
						log.Printf("[RateLimit] Admin bypass for user=%s", claimMap["sub"])
						return c.Next()
					}
					// Apply admin limit (not implemented in this version, would need separate limiter)
				}
			}
		}

		// Apply user limiter (IP-based for unauthenticated, user-based for authenticated)
		return userLimiter(c)
	}
}

// AdminLimiters returns rate limiters with higher limits for admin-only endpoints.
// Use these for admin-specific routes where you want to allow burst operations.
func AdminLimiters(trustedProxies []string) struct {
	Management fiber.Handler
} {
	// Admin management endpoints: 300 requests per minute (for bulk operations)
	managementLimiter := limiter.New(limiter.Config{
		Max:        300,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			claims := c.Locals("user")
			if claims != nil {
				if claimMap, ok := claims.(map[string]interface{}); ok {
					if sub, ok := claimMap["sub"].(string); ok {
						return "admin:" + sub
					}
				}
			}
			return "admin-ip:" + getClientIP(c, trustedProxies)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Admin rate limit exceeded. Please slow down.",
				"code":  "RATE_LIMITED",
			})
		},
	})

	return struct {
		Management fiber.Handler
	}{
		Management: managementLimiter,
	}
}
