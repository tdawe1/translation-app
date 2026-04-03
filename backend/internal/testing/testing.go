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
		"JWT_SECRET":                  "test-secret-for-testing-only-32-chars-min",
		"DB_HOST":                     "localhost",
		"DB_PORT":                     "5433",
		"DB_USER":                     "gengo",
		"DB_PASSWORD":                 "devpass",
		"DB_NAME":                     "gengowatcher_test",
		"TEST_DB_HOST":                "localhost",
		"TEST_DB_PORT":                "5433",
		"TEST_DB_USER":                "gengo",
		"TEST_DB_PASSWORD":            "devpass",
		"TEST_DB_NAME":                "gengowatcher_test",
		"TEST_DB_SSLMODE":             "disable",
		"TEST_DATABASE_URL":           "host=localhost port=5433 user=gengo password=devpass dbname=gengowatcher_test sslmode=disable",
		"DB_SSLMODE":                  "disable",
		"RESEND_API_KEY":              "test-key",
		"FROM_EMAIL":                  "test@example.com",
		"FROM_NAME":                   "TestApp",
		"GOOGLE_OAUTH_CLIENT_ID":      "test-client-id",
		"GOOGLE_OAUTH_CLIENT_SECRET":  "test-client-secret",
		"GITHUB_OAUTH_CLIENT_ID":      "test-client-id",
		"GITHUB_OAUTH_CLIENT_SECRET":  "test-client-secret",
		"OAUTH_REDIRECT_URL":          "http://localhost:8000",
		"FRONTEND_URL":                "http://localhost:3001",
		"LEMONSQUEEZY_WEBHOOK_SECRET": "test-webhook-secret",
		"ALLOWED_ORIGINS":             "http://localhost:3000,http://localhost:3001",
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
		"TEST_DB_HOST",
		"TEST_DB_PORT",
		"TEST_DB_USER",
		"TEST_DB_PASSWORD",
		"TEST_DB_NAME",
		"TEST_DB_SSLMODE",
		"TEST_DATABASE_URL",
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
