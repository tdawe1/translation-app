package models

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Config holds database configuration
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DefaultConfig returns database config from environment variables
func DefaultConfig() *Config {
	return &Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5433"),
		User:     getEnv("DB_USER", "gengo"),
		Password: getEnv("DB_PASSWORD", "devpass"),
		DBName:   getEnv("DB_NAME", "gengowatcher"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

// DSN builds the PostgreSQL connection string
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// InitDB initializes the database connection
func InitDB(config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}

	// Configure GORM logger
	logLevel := logger.Silent
	if os.Getenv("ENV") == "development" {
		logLevel = logger.Info
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	var err error
	DB, err = gorm.Open(postgres.Open(config.DSN()), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now()
		},
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	return nil
}

// AutoMigrate runs auto migration for all models
func AutoMigrate() error {
	return DB.AutoMigrate(
		&User{},
		&OAuthAccount{},
		&APIKey{},
		&RefreshToken{},
		&WatcherConfig{},
		&WatcherState{},
		&SubscriptionPlan{},
		&Subscription{},
		&BillingEvent{},
		&AuditLog{},
	)
}

// getEnv retrieves environment variable or returns default
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
