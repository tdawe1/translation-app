package watcher

import (
	"encoding/json"
	"fmt"
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

// UpdateRuntime updates the extended runtime snapshot fields for a user.
func (m *StateManager) UpdateRuntime(userID uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now().UTC()
	return m.db.Model(&models.WatcherState{}).
		Where("user_id = ?", userID).
		Updates(updates).Error
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

// WatcherEventInput is the persisted and streamed representation of a watcher event.
type WatcherEventInput struct {
	Level   string
	Source  string
	Type    string
	JobID   string
	Message string
	Data    map[string]interface{}
}

// AppendEvent persists a watcher event for the user's timeline.
func (m *StateManager) AppendEvent(userID uuid.UUID, input WatcherEventInput) (*models.WatcherEvent, error) {
	data := "{}"
	if len(input.Data) > 0 {
		raw, err := json.Marshal(input.Data)
		if err != nil {
			return nil, fmt.Errorf("marshal watcher event data: %w", err)
		}
		data = string(raw)
	}

	event := &models.WatcherEvent{
		Base:       models.Base{ID: uuid.New()},
		WorkerID:   userID,
		UserID:     userID,
		Level:      input.Level,
		Source:     input.Source,
		Type:       input.Type,
		JobID:      input.JobID,
		Message:    input.Message,
		Data:       data,
		OccurredAt: time.Now().UTC(),
	}

	if err := m.db.Create(event).Error; err != nil {
		return nil, err
	}
	return event, nil
}

// ListEvents returns recent watcher events for a user in reverse chronological order.
func (m *StateManager) ListEvents(userID uuid.UUID, limit int) ([]models.WatcherEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	gormDB, ok := database.GetPool(m.db)
	if !ok || gormDB == nil {
		return nil, fmt.Errorf("database pool unavailable")
	}

	var events []models.WatcherEvent
	if err := gormDB.
		Where("user_id = ?", userID).
		Order("occurred_at DESC").
		Limit(limit).
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
