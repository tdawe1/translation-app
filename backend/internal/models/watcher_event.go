package models

import (
	"time"

	"github.com/google/uuid"
)

// WatcherEvent is the append-only runtime timeline for a user's watcher worker.
// In the current recovery implementation, the worker identity is the user's
// single watcher instance, so WorkerID is stable per user.
type WatcherEvent struct {
	Base
	WorkerID   uuid.UUID `gorm:"type:uuid;not null;index" json:"worker_id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Level      string    `gorm:"size:20;not null" json:"level"`
	Source     string    `gorm:"size:32;not null;index" json:"source"`
	Type       string    `gorm:"size:120;not null;index" json:"type"`
	JobID      string    `gorm:"size:120" json:"job_id,omitempty"`
	Message    string    `gorm:"type:text;not null" json:"message"`
	Data       string    `gorm:"type:jsonb" json:"data"`
	OccurredAt time.Time `gorm:"not null;index" json:"occurred_at"`
}
