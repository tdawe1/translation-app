package watcher

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
)

// EnsureUserResources creates default watcher resources for a user when they
// are missing, which keeps older accounts compatible with the current watcher
// pipeline.
func EnsureUserResources(db database.Database, userID uuid.UUID) error {
	if err := ensureWatcherConfig(db, userID); err != nil {
		return err
	}
	return ensureWatcherState(db, userID)
}

// BackfillUserResources eagerly prepares watcher resources for all existing
// users. This is intended to run during server startup as a best-effort
// reconciliation task so request handlers are not responsible for bootstrap.
func BackfillUserResources(db database.Database) error {
	gormDB, ok := database.GetPool(db)
	if !ok || gormDB == nil {
		return nil
	}

	var userIDs []uuid.UUID
	if err := gormDB.Model(&models.User{}).Pluck("id", &userIDs).Error; err != nil {
		return err
	}

	for _, userID := range userIDs {
		if err := EnsureUserResources(db, userID); err != nil {
			return fmt.Errorf("user %s: %w", userID, err)
		}
	}

	return nil
}

func ensureWatcherConfig(db database.Database, userID uuid.UUID) error {
	var config models.WatcherConfig
	if err := db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		config = models.WatcherConfig{
			Base:                  models.Base{ID: uuid.New()},
			UserID:                userID,
			IncludedLanguagePairs: "[]",
		}
		if createErr := db.Create(&config).Error; createErr != nil {
			return createErr
		}
	}

	return nil
}

func ensureWatcherState(db database.Database, userID uuid.UUID) error {
	var state models.WatcherState
	if err := db.Where("user_id = ?", userID).First(&state).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		state = models.WatcherState{
			UserID:           userID,
			WatcherStatus:    StatusStopped,
			OverallStatus:    OverallStatusStopped,
			FeedStatus:       FeedStatusStopped,
			BrowserStatus:    BrowserStatusUnconfigured,
			ActionStatus:     ActionStatusIdle,
			AlertStatus:      AlertStatusNone,
			ProfileStatus:    ProfileStatusUnseeded,
			LoggedInState:    "unknown",
			LastSeenJobIDs:   "[]",
			RecentJobHistory: "[]",
		}
		if createErr := db.Create(&state).Error; createErr != nil {
			return createErr
		}
		return nil
	}

	updates := map[string]interface{}{}
	if state.OverallStatus == "" {
		updates["overall_status"] = OverallStatusStopped
	}
	if state.FeedStatus == "" {
		updates["feed_status"] = FeedStatusStopped
	}
	if state.BrowserStatus == "" {
		updates["browser_status"] = BrowserStatusUnconfigured
	}
	if state.ActionStatus == "" {
		updates["action_status"] = ActionStatusIdle
	}
	if state.AlertStatus == "" {
		updates["alert_status"] = AlertStatusNone
	}
	if state.ProfileStatus == "" {
		updates["profile_status"] = ProfileStatusUnseeded
	}
	if state.LoggedInState == "" {
		updates["logged_in_state"] = "unknown"
	}
	if len(updates) > 0 {
		if err := db.Model(&models.WatcherState{}).Where("user_id = ?", userID).Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}
