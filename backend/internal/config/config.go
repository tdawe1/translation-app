// Package config provides centralized configuration management
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration
type Config struct {
	// Server
	Port string
	Env  string // "development" or "production"

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Security
	JWTSecret                string
	LemonSqueezyWebhookSecret string

	// Email (Resend)
	ResendAPIKey string
	FromEmail    string
	FromName     string

	// OAuth
	GoogleOAuthClientID     string
	GoogleOAuthClientSecret string
	GitHubOAuthClientID     string
	GitHubOAuthClientSecret string
	OAuthRedirectURL       string // Base URL for OAuth callbacks

	// CORS
	AllowedOrigins string // Comma-separated list

	// Cookies
	CookieSecure bool // Set Secure flag on cookies
}

// Load reads configuration from environment variables
// It will panic if required values are missing in production
func Load() *Config {
	cfg := &Config{
		Port:                      getEnv("PORT", "8000"),
		Env:                       getEnv("ENV", "development"),
		DBHost:                    getEnv("DB_HOST", "localhost"),
		DBPort:                    getEnv("DB_PORT", "5433"),
		DBUser:                    getEnv("DB_USER", "gengo"),
		DBPassword:                getEnv("DB_PASSWORD", "devpass"),
		DBName:                    getEnv("DB_NAME", "gengowatcher"),
		DBSSLMode:                 getEnv("DB_SSLMODE", "disable"),
		JWTSecret:                 getEnv("JWT_SECRET", ""),
		LemonSqueezyWebhookSecret: getEnv("LEMONSQUEZY_WEBHOOK_SECRET", ""),
		ResendAPIKey:              getEnv("RESEND_API_KEY", ""),
		FromEmail:                 getEnv("FROM_EMAIL", "noreply@gengowatcher.example"),
		FromName:                  getEnv("FROM_NAME", "GengoWatcher"),
		GoogleOAuthClientID:       getEnv("GOOGLE_OAUTH_CLIENT_ID", ""),
		GoogleOAuthClientSecret:   getEnv("GOOGLE_OAUTH_CLIENT_SECRET", ""),
		GitHubOAuthClientID:       getEnv("GITHUB_OAUTH_CLIENT_ID", ""),
		GitHubOAuthClientSecret:   getEnv("GITHUB_OAUTH_CLIENT_SECRET", ""),
		OAuthRedirectURL:         getEnv("OAUTH_REDIRECT_URL", "http://localhost:3000"),
		AllowedOrigins:            getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:3001"),
		CookieSecure:              getEnv("ENV", "development") == "production",
	}

	// Validate required secrets in production
	if cfg.Env == "production" {
		if cfg.JWTSecret == "" {
			panic("JWT_SECRET must be set in production")
		}
	}

	// Generate a warning if using default JWT secret in development
	if cfg.Env == "development" && cfg.JWTSecret == "" {
		cfg.JWTSecret = "dev-secret-change-in-production"
		fmt.Println("⚠️  WARNING: Using default JWT secret. Set JWT_SECRET in production!")
	}

	return cfg
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// AllowedOriginList returns the allowed origins as a slice
func (c *Config) AllowedOriginList() []string {
	origins := strings.Split(c.AllowedOrigins, ",")
	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// getEnv retrieves an environment variable or returns the default value
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
