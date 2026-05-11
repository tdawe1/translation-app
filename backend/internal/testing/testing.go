// Package testing provides test utilities for the application
package testing

import (
	"os"

	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/logger"
)

// SetupTestEnvironment initializes environment variables for testing.
// Call this in TestMain or at the start of tests that require config.
func SetupTestEnvironment() {
	logger.Init("test")

	// Set required environment variables with safe defaults for testing
	envVars := map[string]string{
		"JWT_SECRET":                  "test-secret-key-32-characters-long-for-hs256!",
		"DB_HOST":                     "localhost",
		"DB_PORT":                     "5432",
		"DB_USER":                     "test",
		"DB_PASSWORD":                 "test",
		"DB_NAME":                     "testdb",
		"DB_SSLMODE":                  "disable",
		"RESEND_API_KEY":              "test-key",
		"FROM_EMAIL":                  "test@example.com",
		"FROM_NAME":                   "TestApp",
		"GOOGLE_OAUTH_CLIENT_ID":      "test-client-id",
		"GOOGLE_OAUTH_CLIENT_SECRET":  "test-client-secret",
		"GITHUB_OAUTH_CLIENT_ID":      "test-client-id",
		"GITHUB_OAUTH_CLIENT_SECRET":  "test-client-secret",
		"OAUTH_REDIRECT_URL":          "http://localhost:37181",
		"FRONTEND_URL":                "http://localhost:37180",
		"LEMONSQUEEZY_WEBHOOK_SECRET": "test-webhook-secret",
		"ALLOWED_ORIGINS":             "http://localhost:37180",
		"ENV":                         "test",
	}

	for key, value := range envVars {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

// TestConfig returns a config.Config instance for testing
func TestConfig() *config.Config {
	SetupTestEnvironment()
	return config.Load()
}

// CleanupTestEnvironment clears test environment variables.
// Useful for tests that need to verify config loading behavior.
func CleanupTestEnvironment() {
	envVars := []string{
		"JWT_SECRET",
		"DB_HOST",
		"DB_PORT",
		"DB_USER",
		"DB_PASSWORD",
		"DB_NAME",
		"DB_SSLMODE",
		"RESEND_API_KEY",
		"FROM_EMAIL",
		"FROM_NAME",
		"GOOGLE_OAUTH_CLIENT_ID",
		"GOOGLE_OAUTH_CLIENT_SECRET",
		"GITHUB_OAUTH_CLIENT_ID",
		"GITHUB_OAUTH_CLIENT_SECRET",
		"OAUTH_REDIRECT_URL",
		"FRONTEND_URL",
		"LEMONSQUEEZY_WEBHOOK_SECRET",
		"ALLOWED_ORIGINS",
		"ENV",
	}

	for _, key := range envVars {
		os.Unsetenv(key)
	}
}
