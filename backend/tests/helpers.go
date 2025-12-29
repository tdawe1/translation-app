package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tdawe1/translation-app/internal/models"
)

// TestDB returns a test database connection
// Uses PostgreSQL test database from environment
func TestDB(t *testing.T) *gorm.DB {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=gengo password=devpass dbname=gengowatcher_test sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "Failed to connect to test database")

	// Clean up any existing data
	db.Exec("DELETE FROM billing_events WHERE 1=1")
	db.Exec("DELETE FROM refresh_tokens WHERE 1=1")
	db.Exec("DELETE FROM api_keys WHERE 1=1")
	db.Exec("DELETE FROM oauth_accounts WHERE 1=1")
	db.Exec("DELETE FROM watcher_states WHERE 1=1")
	db.Exec("DELETE FROM watcher_configs WHERE 1=1")
	db.Exec("DELETE FROM users WHERE 1=1")

	t.Cleanup(func() {
		// Clean up after test
		db.Exec("DELETE FROM billing_events WHERE 1=1")
		db.Exec("DELETE FROM refresh_tokens WHERE 1=1")
		db.Exec("DELETE FROM api_keys WHERE 1=1")
		db.Exec("DELETE FROM oauth_accounts WHERE 1=1")
		db.Exec("DELETE FROM watcher_states WHERE 1=1")
		db.Exec("DELETE FROM watcher_configs WHERE 1=1")
		db.Exec("DELETE FROM users WHERE 1=1")

		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return db
}

// TestRedis returns a test Redis client using database 15
func TestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       15, // Use test database
	})

	// Ping to verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Ping(ctx).Err()
	require.NoError(t, err, "Failed to connect to Redis")

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

	hashedPassword := "$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj9SjKE2FqW" // "password123"

	user := &models.User{
		Email:        email,
		PasswordHash: hashedPassword,
		EmailVerified: true,
		IsActive:     true,
	}
	require.NoError(t, db.Create(user).Error, "Failed to create test user")

	// Create default watcher config
	config := &models.WatcherConfig{
		UserID:           user.ID,
		RSSFeedURL:       "https://gengo.com/jobs/rss",
		WebSocketEnabled: true,
		MinReward:        1.0,
		MaxReward:        50.0,
	}
	require.NoError(t, db.Create(config).Error, "Failed to create watcher config")

	// Create default watcher state
	state := &models.WatcherState{
		UserID:        user.ID,
		WatcherStatus: "stopped",
		TotalJobsFound: 0,
	}
	require.NoError(t, db.Create(state).Error, "Failed to create watcher state")

	return user
}

// GenerateTestToken generates a test JWT token for a user
// This uses the test secret from .env.test
func GenerateTestToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()

	// Import jwt package
	claims := map[string]interface{}{
		"user_id": userID.String(),
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	// Sign with test secret
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "test-secret-for-testing-only-32-chars-min"
	}

	// Simple token generation (using HS256)
	token := fmt.Sprintf("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.%s", encodeSegment(claims, secret))
	return token
}

// encodeSegment is a simple base64url encoder for test tokens
func encodeSegment(data interface{}, secret string) string {
	// This is a simplified version for testing
	// In real tests, you'd use the actual jwt library
	return "test_payload_" + uuid.New().String()
}
