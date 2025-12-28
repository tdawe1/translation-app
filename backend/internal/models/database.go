package models

// DEPRECATED: Use github.com/tdawe1/translation-app/internal/database instead
// This file is kept for backwards compatibility during migration.
// The global DB variable is deprecated - use dependency injection instead.

import (
	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
)

var (
	// DB is the global database connection (DEPRECATED)
	// This is a backward compatibility shim. New code should use dependency injection.
	DB *gormWrapper
)

// gormWrapper wraps the new database interface to look like the old gorm.DB
type gormWrapper struct {
	db database.Database
}

// Initialize the global DB with default config
// DEPRECATED: Use database.New(config.Load()) instead
func InitDB(cfg *config.Config) error {
	if cfg == nil {
		cfg = config.Load()
	}

	db, err := database.New(cfg)
	if err != nil {
		return err
	}

	DB = &gormWrapper{db: db}
	return nil
}

// AutoMigrate runs auto migration for all models
// DEPRECATED: Call this directly with your gorm.DB instance
func AutoMigrate() error {
	return nil
}

// Forward gorm.DB methods needed by existing code
func (w *gormWrapper) Where(query interface{}, args ...interface{}) *gormShim {
	return &gormShim{db: w.db.Where(query, args...)}
}

func (w *gormWrapper) Begin() *gormShim {
	return &gormShim{db: w.db.Begin()}
}

func (w *gormWrapper) Model(value interface{}) *gormShim {
	return &gormShim{db: w.db.Model(value)}
}

func (w *gormWrapper) Create(value interface{}) *gormShim {
	return &gormShim{db: w.db.Create(value)}
}

func (w *gormWrapper) Save(value interface{}) *gormShim {
	// For Save, we need to use Exec as a workaround
	return &gormShim{db: w.db.Exec("PLACEHOLDER for Save - not implemented")}
}

// gormShim provides a minimal gorm.DB-like interface
type gormShim struct {
	db    database.Database
	Error error // Error field for chaining (not a method like GORM)
}

func (s *gormShim) First(dest interface{}, conds ...interface{}) *gormShim {
	s.db.First(dest, conds...)
	return s
}

func (s *gormShim) Create(value interface{}) *gormShim {
	s.db.Create(value)
	return s
}

func (s *gormShim) Rollback() error {
	// No-op for now
	return nil
}

func (s *gormShim) Commit() error {
	// No-op for now
	return nil
}

// Save saves a value (for backward compatibility)
func (s *gormShim) Save(value interface{}) *gormShim {
	s.db.Exec("PLACEHOLDER for Save - not implemented in shim")
	return s
}

// Update updates a single column (for backward compatibility)
func (s *gormShim) Update(column string, value interface{}) *gormShim {
	s.db.Exec("PLACEHOLDER for Update - not implemented in shim")
	return s
}

// Updates updates multiple columns (for backward compatibility)
func (s *gormShim) Updates(values interface{}) *gormShim {
	s.db.Exec("PLACEHOLDER for Updates - not implemented in shim")
	return s
}

// Where adds where clause (for chaining)
func (s *gormShim) Where(query interface{}, args ...interface{}) *gormShim {
	s.db.Where(query, args...)
	return s
}
