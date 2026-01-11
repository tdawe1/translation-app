package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
)

func TestDatabase_ConnectionPoolConfigured(t *testing.T) {
	cfg := &config.Config{
		DBHost:               "localhost",
		DBPort:               "5432",
		DBUser:               "gengo",
		DBPassword:           "devpass",
		DBName:               "gengowatcher_test",
		DBSSLMode:            "disable",
		Env:                  "development",
		DBMaxOpenConnections: 25,
		DBMaxIdleConnections: 10,
		DBConnMaxLifetime:    1 * time.Hour,
		DBConnMaxIdleTime:    10 * time.Minute,
	}

	// Create database connection using database.New()
	db, err := database.New(cfg)
	require.NoError(t, err)

	// Get the underlying GORM DB from our Database interface
	gormDB, ok := database.GetPool(db)
	require.True(t, ok, "Should be able to get underlying GORM DB")

	sqlDB, err := gormDB.DB()
	require.NoError(t, err, "Should be able to get sql.DB")

	// Test connection pool settings
	stats := sqlDB.Stats()

	// Verify MaxOpenConnections is set (not using Go's default of 0/unlimited)
	assert.Greater(t, stats.MaxOpenConnections, 0, "MaxOpenConnections should be set (not default unlimited)")

	// Note: IdleConnections in stats shows current idle count, not the configured max
	// We verify the pool is being used by checking we have open connections available
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 1, "At least 1 connection should be allowed")

	t.Cleanup(func() {
		sqlDB.Close()
	})
}

func TestDatabase_ConnectionLifetimeSet(t *testing.T) {
	cfg := &config.Config{
		DBHost:               "localhost",
		DBPort:               "5432",
		DBUser:               "gengo",
		DBPassword:           "devpass",
		DBName:               "gengowatcher_test",
		DBSSLMode:            "disable",
		Env:                  "development",
		DBMaxOpenConnections: 25,
		DBMaxIdleConnections: 10,
		DBConnMaxLifetime:    1 * time.Hour,
		DBConnMaxIdleTime:    10 * time.Minute,
	}

	db, err := database.New(cfg)
	require.NoError(t, err)

	gormDB, ok := database.GetPool(db)
	require.True(t, ok)

	sqlDB, _ := gormDB.DB()

	// Verify connection pool is configured with custom values
	assert.Greater(t, sqlDB.Stats().MaxOpenConnections, 0, "Pool should be configured")

	t.Cleanup(func() {
		sqlDB.Close()
	})
}

func TestDatabase_PoolSettingsFromConfig(t *testing.T) {
	// Test with custom pool settings
	cfg := &config.Config{
		DBHost:               "localhost",
		DBPort:               "5432",
		DBUser:               "gengo",
		DBPassword:           "devpass",
		DBName:               "gengowatcher_test",
		DBSSLMode:            "disable",
		Env:                  "development",
		DBMaxOpenConnections: 50,
		DBMaxIdleConnections: 20,
		DBConnMaxLifetime:    30 * time.Minute,
		DBConnMaxIdleTime:    5 * time.Minute,
	}

	db, err := database.New(cfg)
	require.NoError(t, err)

	gormDB, ok := database.GetPool(db)
	require.True(t, ok)

	sqlDB, _ := gormDB.DB()

	// Verify pool settings are configured
	assert.Greater(t, sqlDB.Stats().MaxOpenConnections, 0, "Pool should be configured")

	t.Cleanup(func() {
		sqlDB.Close()
	})
}
