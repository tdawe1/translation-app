package testing

import (
	"os"
	"testing"

	"github.com/tdawe1/translation-app/internal/config"
)

func TestSetupTestEnvironment(t *testing.T) {
	// Clear any existing env vars first
	CleanupTestEnvironment()

	SetupTestEnvironment()

	// Verify environment variables are set
	expectedVars := map[string]string{
		"JWT_SECRET":                 "test-secret-for-testing-only-32-chars-min",
		"DB_HOST":                    "localhost",
		"DB_PORT":                    "5433",
		"DB_USER":                    "gengo",
		"DB_PASSWORD":                "devpass",
		"DB_NAME":                    "gengowatcher_test",
		"TEST_DB_HOST":               "localhost",
		"TEST_DB_PORT":               "5433",
		"TEST_DB_USER":               "gengo",
		"TEST_DB_PASSWORD":           "devpass",
		"TEST_DB_NAME":               "gengowatcher_test",
		"RESEND_API_KEY":             "test-key",
		"FROM_EMAIL":                 "test@example.com",
		"GOOGLE_OAUTH_CLIENT_ID":     "test-client-id",
		"GOOGLE_OAUTH_CLIENT_SECRET": "test-client-secret",
		"GITHUB_OAUTH_CLIENT_ID":     "test-client-id",
		"GITHUB_OAUTH_CLIENT_SECRET": "test-client-secret",
		"ENV":                        "test",
	}

	for key, expected := range expectedVars {
		if got := os.Getenv(key); got != expected {
			t.Errorf("%s = %q, want %q", key, got, expected)
		}
	}

	// Load config to verify it works
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		t.Error("JWT_SECRET should be set in test environment")
	}

	if cfg.JWTSecret == "test-secret-for-testing-only-32-chars-min" {
		t.Log("Test environment properly configured")
	}
}

func TestTestConfig(t *testing.T) {
	CleanupTestEnvironment()
	cfg := TestConfig()

	if cfg == nil {
		t.Fatal("TestConfig returned nil")
	}

	if cfg.JWTSecret == "" {
		t.Error("JWT_SECRET not set in TestConfig")
	}

	// Verify it's using test values
	if cfg.Env != "test" {
		t.Errorf("Env = %q, want 'test'", cfg.Env)
	}

	if cfg.DBName != "gengowatcher_test" {
		t.Errorf("DBName = %q, want 'gengowatcher_test'", cfg.DBName)
	}
}

func TestCleanupTestEnvironment(t *testing.T) {
	SetupTestEnvironment()

	// Verify vars are set
	if os.Getenv("JWT_SECRET") == "" {
		t.Error("Expected JWT_SECRET to be set before cleanup")
	}

	CleanupTestEnvironment()

	// Verify vars are unset
	if os.Getenv("JWT_SECRET") != "" {
		t.Error("Expected JWT_SECRET to be unset after cleanup")
	}
}

func TestSetupIdempotent(t *testing.T) {
	CleanupTestEnvironment()

	SetupTestEnvironment()
	firstValue := os.Getenv("JWT_SECRET")

	// Calling again should not change values
	SetupTestEnvironment()
	secondValue := os.Getenv("JWT_SECRET")

	if firstValue != secondValue {
		t.Error("SetupTestEnvironment should be idempotent")
	}
}
