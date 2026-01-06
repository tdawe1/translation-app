package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/testing" as app_testing
)

func init() {
	// Set up test environment using the centralized test helpers
	// This ensures all required environment variables are set for testing
	app_testing.SetupTestEnvironment()
}

// TestDB returns a test database connection
// Uses PostgreSQL test database - runs migrations for realistic testing
func TestDB(t *testing.T) *gorm.DB {
	// Construct DSN from individual env vars or use defaults
	dbHost := getEnv("TEST_DB_HOST", "localhost")
	dbPort := getEnv("TEST_DB_PORT", "5433")
	dbUser := getEnv("TEST_DB_USER", "gengo")
	dbPass := getEnv("TEST_DB_PASSWORD", "devpass")
	dbName := getEnv("TEST_DB_NAME", "gengowatcher_test")
	dbSSL := getEnv("TEST_DB_SSLMODE", "disable")

	dsn := "host=" + dbHost + " port=" + dbPort + " user=" + dbUser + " password=" + dbPass + " dbname=" + dbName + " sslmode=" + dbSSL

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Failed to connect to test database")

	// Clean up any existing data
	db.Exec("DELETE FROM audit_logs WHERE 1=1")
	db.Exec("DELETE FROM billing_events WHERE 1=1")
	db.Exec("DELETE FROM refresh_tokens WHERE 1=1")
	db.Exec("DELETE FROM api_keys WHERE 1=1")
	db.Exec("DELETE FROM oauth_accounts WHERE 1=1")
	db.Exec("DELETE FROM watcher_states WHERE 1=1")
	db.Exec("DELETE FROM watcher_configs WHERE 1=1")
	db.Exec("DELETE FROM subscriptions WHERE 1=1")
	db.Exec("DELETE FROM users WHERE 1=1")

	t.Cleanup(func() {
		// Clean up after test
		db.Exec("DELETE FROM audit_logs WHERE 1=1")
		db.Exec("DELETE FROM billing_events WHERE 1=1")
		db.Exec("DELETE FROM refresh_tokens WHERE 1=1")
		db.Exec("DELETE FROM api_keys WHERE 1=1")
		db.Exec("DELETE FROM oauth_accounts WHERE 1=1")
		db.Exec("DELETE FROM watcher_states WHERE 1=1")
		db.Exec("DELETE FROM watcher_configs WHERE 1=1")
		db.Exec("DELETE FROM subscriptions WHERE 1=1")
		db.Exec("DELETE FROM users WHERE 1=1")

		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return db
}

// TestRedis returns a test Redis client using database 15
// Returns nil if Redis is not available (tests can skip)
func TestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       15, // Use test database
	})

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.Ping(ctx).Err()
	if err != nil {
		t.Skip("Skipping test: Redis not available")
		return nil
	}

	t.Cleanup(func() {
		// Clean up test keys
		client.FlushDB(ctx)
		client.Close()
	})

	return client
}

// CreateTestUser creates a test user with hashed password
func CreateTestUser(t *testing.T, db *gorm.DB, email string) *models.User {
	t.Helper()

	hashedPassword := "$2a$10$uvmy6V0Jm.l3g5jK1TeLoeCAldIB0Q6NW6tnii7tI2z.WwIcIe3m2" // "password123"

	user := &models.User{
		Email:        email,
		PasswordHash: hashedPassword,
		EmailVerified: true,
		IsActive:     true,
	}
	require.NoError(t, db.Create(user).Error, "Failed to create test user")

	// Create default watcher config
	config := &models.WatcherConfig{
		UserID:                 user.ID,
		RSSFeedURL:             "https://gengo.com/jobs/rss",
		WebSocketEnabled:       true,
		MinReward:              1.0,
		MaxReward:              50.0,
		IncludedLanguagePairs:  "[]", // JSON array required for jsonb column
	}
	require.NoError(t, db.Create(config).Error, "Failed to create watcher config")

	// Create default watcher state
	state := &models.WatcherState{
		UserID:           user.ID,
		WatcherStatus:    "stopped",
		TotalJobsFound:   0,
		LastSeenJobIDs:   "[]", // JSON array required for jsonb column
		RecentJobHistory: "[]", // JSON array required for jsonb column
	}
	require.NoError(t, db.Create(state).Error, "Failed to create watcher state")

	return user
}

// GenerateTestToken generates a test JWT token for a user
// Uses the same JWT library and signing method as the middleware
func GenerateTestToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()

	// Get the secret from environment or use test default
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "test-secret-for-testing-only-32-chars-min"
	}

	// Create claims with "sub" field (standard JWT subject claim)
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	require.NoError(t, err, "Failed to sign test token")

	return tokenString
}

// RequireRedis is a test helper that skips the test if Redis is not available
func RequireRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available")
		return nil
	}

	t.Cleanup(func() {
		client.FlushDB(ctx)
		client.Close()
	})

	return client
}

// RequireDB is a test helper that skips the test if PostgreSQL is not available
func RequireDB(t *testing.T) *gorm.DB {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=gengo password=devpass dbname=gengowatcher_test sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skip("PostgreSQL not available")
		return nil
	}

	// Verify connection
	sqlDB, _ := db.DB()
	if err := sqlDB.Ping(); err != nil {
		t.Skip("PostgreSQL not accessible")
		return nil
	}

	// Clean up any existing data
	db.Exec("DELETE FROM audit_logs WHERE 1=1")
	db.Exec("DELETE FROM billing_events WHERE 1=1")
	db.Exec("DELETE FROM refresh_tokens WHERE 1=1")
	db.Exec("DELETE FROM api_keys WHERE 1=1")
	db.Exec("DELETE FROM oauth_accounts WHERE 1=1")
	db.Exec("DELETE FROM watcher_states WHERE 1=1")
	db.Exec("DELETE FROM watcher_configs WHERE 1=1")
	db.Exec("DELETE FROM subscriptions WHERE 1=1")
	db.Exec("DELETE FROM users WHERE 1=1")

	t.Cleanup(func() {
		db.Exec("DELETE FROM audit_logs WHERE 1=1")
		db.Exec("DELETE FROM billing_events WHERE 1=1")
		db.Exec("DELETE FROM refresh_tokens WHERE 1=1")
		db.Exec("DELETE FROM api_keys WHERE 1=1")
		db.Exec("DELETE FROM oauth_accounts WHERE 1=1")
		db.Exec("DELETE FROM watcher_states WHERE 1=1")
		db.Exec("DELETE FROM watcher_configs WHERE 1=1")
		db.Exec("DELETE FROM subscriptions WHERE 1=1")
		db.Exec("DELETE FROM users WHERE 1=1")
		sqlDB.Close()
	})

	return db
}

// AssertRedisHasKey is a test helper that asserts a Redis key exists
func AssertRedisHasKey(t *testing.T, client *redis.Client, key string) {
	ctx := context.Background()
	exists, err := client.Exists(ctx, key).Result()
	assert.NoError(t, err)
	assert.True(t, exists > 0, "Expected key %s to exist", key)
}

// AssertRedisHasValue is a test helper that asserts a Redis key has a specific value
func AssertRedisHasValue(t *testing.T, client *redis.Client, key string, expected string) {
	ctx := context.Background()
	val, err := client.Get(ctx, key).Result()
	assert.NoError(t, err)
	assert.Equal(t, expected, val)
}

// getEnv retrieves an environment variable or returns the default value
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
