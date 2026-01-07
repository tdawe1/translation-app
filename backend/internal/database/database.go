// Package database provides database abstraction and connection management
package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tdawe1/translation-app/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database defines the interface for database operations
// This allows us to inject mock implementations for testing
type Database interface {
	// Create creates a new record
	Create(value interface{}) *gorm.DB
	// First finds the first record by conditions
	First(dest interface{}, conds ...interface{}) *gorm.DB
	// Where adds a where condition
	Where(query interface{}, args ...interface{}) *gorm.DB
	// Model specifies the model for operations
	Model(value interface{}) *gorm.DB
	// Begin starts a transaction
	Begin(opts ...*sql.TxOptions) *gorm.DB
	// Exec executes raw SQL
	Exec(sql string, values ...interface{}) *gorm.DB
	// Save saves a value (including associations)
	Save(value interface{}) *gorm.DB
	// Updates updates multiple columns
	Updates(values interface{}) *gorm.DB
	// UpdateColumn updates a single column with expression
	UpdateColumn(column string, value interface{}) *gorm.DB
	// Update updates a single column
	Update(column string, value interface{}) *gorm.DB
}

// gormDB wraps gorm.DB to implement our Database interface
type gormDB struct {
	*gorm.DB
}

// Ensure gormDB implements Database interface
var _ Database = (*gormDB)(nil)

// New creates a new database connection
func New(cfg *config.Config) (Database, error) {
	// Build DSN
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	// Configure logger
	logLevel := logger.Silent
	if cfg.IsDevelopment() {
		logLevel = logger.Info
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool from config
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Set maximum number of open connections
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConnections)

	// Set maximum number of idle connections
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConnections)

	// Set maximum connection lifetime (only if configured)
	if cfg.DBConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.DBConnMaxLifetime)
	}

	// Set maximum idle time for connections (only if configured)
	if cfg.DBConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.DBConnMaxIdleTime)
	}

	return &gormDB{DB: db}, nil
}

// MustNew creates a new database connection or panics
// Useful for main package initialization
func MustNew(cfg *config.Config) Database {
	db, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return db
}

// GetPool returns the underlying sql.DB for connection pool management
func GetPool(db Database) (*gorm.DB, bool) {
	if g, ok := db.(*gormDB); ok {
		return g.DB, true
	}
	return nil, false
}
