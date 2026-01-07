package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/seeds"
)

func TestAdminSeeder_EnsureAdminUser(t *testing.T) {
	gormDB := RequireDB(t)
	db := database.Wrap(gormDB)
	tokenSvc := auth.NewTokenService("test-secret-for-testing-only-32-chars-min")
	seeder := seeds.NewAdminSeeder(db, tokenSvc)

	user, token, err := seeder.EnsureAdminUser("test-seed-admin@example.com", "TestPassword123!")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "admin", user.Role)
	assert.NotEmpty(t, token)

	// Verify calling again with same email updates instead of duplicates
	user2, token2, err := seeder.EnsureAdminUser("test-seed-admin@example.com", "NewPassword456!")
	require.NoError(t, err)
	assert.Equal(t, user.ID, user2.ID)
	assert.NotEmpty(t, token2)
}
