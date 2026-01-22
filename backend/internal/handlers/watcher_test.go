package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/watcher"
	"github.com/tdawe1/translation-app/tests"
	"gorm.io/gorm"
)

const testSecret = "test-secret-for-testing-only-32-chars-min"

func init() {
	os.Setenv("JWT_SECRET", testSecret)
	os.Setenv("TEST_ENV", "true")
}

func setupTestApp(t *testing.T) (*fiber.App, *WatcherHandler, *gorm.DB) {
	t.Helper()

	db := tests.RequireDB(t)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	cfg := middleware.NewJWTConfig(middleware.WithSecret(testSecret))
	app.Use(middleware.JWTValidator(cfg))

	handler := NewWatcherHandler(watcher.NewTestManager(db), db)

	return app, handler, db
}

func TestWatcherHandler_GetConfig_RequiresAuth(t *testing.T) {
	app, handler, _ := setupTestApp(t)
	app.Get("/api/v1/watcher/config", handler.GetConfig)

	req := httptest.NewRequest("GET", "/api/v1/watcher/config", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestWatcherHandler_GetConfig_ReturnsUserConfig(t *testing.T) {
	app, handler, db := setupTestApp(t)
	app.Get("/api/v1/watcher/config", handler.GetConfig)

	user := tests.CreateTestUser(t, db, "user1@example.com")
	token := tests.GenerateTestToken(t, user.ID)

	req := httptest.NewRequest("GET", "/api/v1/watcher/config", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), result["user_id"])
}

func TestWatcherHandler_UpdateConfig_ValidInput(t *testing.T) {
	app, handler, db := setupTestApp(t)
	app.Put("/api/v1/watcher/config", handler.UpdateConfig)

	user := tests.CreateTestUser(t, db, "user2@example.com")
	token := tests.GenerateTestToken(t, user.ID)

	updateReq := map[string]interface{}{
		"min_reward":                   5.0,
		"max_reward":                   100.0,
		"included_language_pairs":      []string{"en->ja", "en->fr"},
		"enable_desktop_notifications": true,
	}
	reqBody, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PUT", "/api/v1/watcher/config", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, 5.0, result["min_reward"])
	assert.Equal(t, 100.0, result["max_reward"])
}

func TestWatcherHandler_UpdateConfig_InvalidInput(t *testing.T) {
	app, handler, db := setupTestApp(t)
	app.Put("/api/v1/watcher/config", handler.UpdateConfig)

	user := tests.CreateTestUser(t, db, "user3@example.com")
	token := tests.GenerateTestToken(t, user.ID)

	reqBody := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest("PUT", "/api/v1/watcher/config", reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestWatcherHandler_MultiTenancy(t *testing.T) {
	app, handler, db := setupTestApp(t)
	app.Put("/api/v1/watcher/config", handler.UpdateConfig)
	app.Get("/api/v1/watcher/config", handler.GetConfig)

	user1 := tests.CreateTestUser(t, db, "user1@example.com")
	user2 := tests.CreateTestUser(t, db, "user2@example.com")

	token1 := tests.GenerateTestToken(t, user1.ID)
	token2 := tests.GenerateTestToken(t, user2.ID)

	updateReq := map[string]interface{}{
		"min_reward": 10.0,
	}
	reqBody, _ := json.Marshal(updateReq)
	req1 := httptest.NewRequest("PUT", "/api/v1/watcher/config", bytes.NewBuffer(reqBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+token1)
	_, err := app.Test(req1)
	require.NoError(t, err)

	req2 := httptest.NewRequest("GET", "/api/v1/watcher/config", nil)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token2)
	resp, err := app.Test(req2)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.NotEqual(t, 10.0, result["min_reward"])
}
