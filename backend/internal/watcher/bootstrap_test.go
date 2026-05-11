package watcher

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/tests"
)

func TestEnsureUserResources_CreatesMissingRecords(t *testing.T) {
	db := tests.RequireDB(t)
	wrappedDB := database.Wrap(db)

	user := &models.User{
		Email:         "bootstrap-missing@example.com",
		PasswordHash:  "$2a$10$uvmy6V0Jm.l3g5jK1TeLoeCAldIB0Q6NW6tnii7tI2z.WwIcIe3m2",
		EmailVerified: true,
		IsActive:      true,
	}
	require.NoError(t, db.Create(user).Error)

	require.NoError(t, EnsureUserResources(wrappedDB, user.ID))

	var config models.WatcherConfig
	require.NoError(t, db.Where("user_id = ?", user.ID).First(&config).Error)
	require.Equal(t, "[]", config.IncludedLanguagePairs)

	var state models.WatcherState
	require.NoError(t, db.Where("user_id = ?", user.ID).First(&state).Error)
	require.Equal(t, "stopped", state.WatcherStatus)
	require.Equal(t, "[]", state.LastSeenJobIDs)
	require.Equal(t, "[]", state.RecentJobHistory)
}

func TestEnsureUserResources_BackfillsBlankWatcherStatus(t *testing.T) {
	db := tests.RequireDB(t)
	wrappedDB := database.Wrap(db)

	user := &models.User{
		Email:         "bootstrap-backfill@example.com",
		PasswordHash:  "$2a$10$uvmy6V0Jm.l3g5jK1TeLoeCAldIB0Q6NW6tnii7tI2z.WwIcIe3m2",
		EmailVerified: true,
		IsActive:      true,
	}
	require.NoError(t, db.Create(user).Error)
	require.NoError(t, EnsureUserResources(wrappedDB, user.ID))

	require.NoError(t, db.Model(&models.WatcherState{}).
		Where("user_id = ?", user.ID).
		Update("watcher_status", "").Error)

	require.NoError(t, EnsureUserResources(wrappedDB, user.ID))

	var state models.WatcherState
	require.NoError(t, db.Where("user_id = ?", user.ID).First(&state).Error)
	require.Equal(t, StatusStopped, state.WatcherStatus)
}
