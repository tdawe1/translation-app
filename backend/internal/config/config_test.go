package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Development(t *testing.T) {
	// Clear environment
	os.Clearenv()
	os.Setenv("TEST_ENV", "true")
	os.Setenv("ENV", "development")

	cfg := Load()

	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, "8000", cfg.Port)
	assert.False(t, cfg.CookieSecure)
	assert.NotEmpty(t, cfg.JWTSecret, "Should generate default JWT secret in dev")
}

func TestLoad_Production_RequiresJWTSecret(t *testing.T) {
	os.Clearenv()
	os.Setenv("ENV", "production")

	assert.Panics(t, func() {
		Load()
	}, "Should panic without JWT_SECRET in production")
}

func TestLoad_Production_WithJWTSecret(t *testing.T) {
	os.Clearenv()
	os.Setenv("TEST_ENV", "true")
	os.Setenv("ENV", "production")
	os.Setenv("JWT_SECRET", "my-super-secret-jwt-key-for-prod")

	cfg := Load()

	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, "my-super-secret-jwt-key-for-prod", cfg.JWTSecret)
	assert.True(t, cfg.CookieSecure)
}

func TestIsDevelopment(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"development", true},
		{"production", false},
		{"staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env}
			assert.Equal(t, tt.want, cfg.IsDevelopment())
		})
	}
}

func TestIsProduction(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"production", true},
		{"development", false},
		{"staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := &Config{Env: tt.env}
			assert.Equal(t, tt.want, cfg.IsProduction())
		})
	}
}

func TestAllowedOriginList(t *testing.T) {
	tests := []struct {
		name    string
		origins string
		want    []string
	}{
		{
			name:    "single origin",
			origins: "http://localhost:3000",
			want:    []string{"http://localhost:3000"},
		},
		{
			name:    "multiple origins",
			origins: "http://localhost:3000,https://app.example.com",
			want:    []string{"http://localhost:3000", "https://app.example.com"},
		},
		{
			name:    "with spaces",
			origins: " http://localhost:3000 , https://app.example.com ",
			want:    []string{"http://localhost:3000", "https://app.example.com"},
		},
		{
			name:    "empty string",
			origins: "",
			want:    []string{},
		},
		{
			name:    "only commas",
			origins: ",,",
			want:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{AllowedOrigins: tt.origins}
			got := cfg.AllowedOriginList()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetEnv(t *testing.T) {
	// Test with value set
	os.Setenv("TEST_VAR", "test_value")
	assert.Equal(t, "test_value", getEnv("TEST_VAR", "default"))

	// Test with value not set
	os.Unsetenv("TEST_VAR_MISSING")
	assert.Equal(t, "default", getEnv("TEST_VAR_MISSING", "default"))

	// Test empty string is considered unset
	os.Setenv("TEST_VAR_EMPTY", "")
	assert.Equal(t, "default", getEnv("TEST_VAR_EMPTY", "default"))
}
