package watcher

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
)

// StateManager handles database operations for watcher state
type StateManager struct {
	db database.Database
}

// NewStateManager creates a new state manager
func NewStateManager(db database.Database) *StateManager {
	return &StateManager{db: db}
}

// LoadConfig retrieves the watcher config for a user
func (m *StateManager) LoadConfig(userID uuid.UUID) (*models.WatcherConfig, error) {
	var config models.WatcherConfig
	err := m.db.Where("user_id = ?", userID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// LoadState retrieves the watcher state for a user
func (m *StateManager) LoadState(userID uuid.UUID) (*models.WatcherState, error) {
	var state models.WatcherState
	err := m.db.Where("user_id = ?", userID).First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// UpdateStatus updates the watcher status in the database
func (m *StateManager) UpdateStatus(userID uuid.UUID, status string) error {
	return m.db.Model(&models.WatcherState{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"watcher_status": status,
			"last_activity":  time.Now(),
		}).Error
}

// UpdateConfig updates the watcher config in the database
func (m *StateManager) UpdateConfig(config *models.WatcherConfig) error {
	return m.db.Save(config).Error
}

// GetStatus retrieves the current status for a user
func (m *StateManager) GetStatus(userID uuid.UUID) (string, error) {
	var state models.WatcherState
	err := m.db.Where("user_id = ?", userID).First(&state).Error
	if err != nil {
		return "", err
	}
	return state.WatcherStatus, nil
}

// IncrementJobCount increments the job counter for a user
func (m *StateManager) IncrementJobCount(userID uuid.UUID) error {
	return m.db.Model(&models.WatcherState{}).
		Where("user_id = ?", userID).
		UpdateColumn("total_jobs_found", gorm.Expr("total_jobs_found + 1")).
		Update("last_activity", gorm.Expr("NOW()")).Error
}
