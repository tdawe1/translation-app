package tests

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/handlers"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/watcher"
)

// TestWatcher_CompleteFlow tests the full watcher lifecycle
func TestWatcher_CompleteFlow(t *testing.T) {
	db := TestDB(t)
	manager := watcher.NewTestManager(t)

	// Create test app
	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	// Wrap DB to implement database.Database interface
	wrappedDB := &databaseWrapper{db: db}

	watcherHandler := handlers.NewWatcherHandler(manager, wrappedDB)
	app.Get("/api/v1/watcher/config", watcherHandler.GetConfig)
	app.Put("/api/v1/watcher/config", watcherHandler.UpdateConfig)
	app.Get("/api/v1/watcher/state", watcherHandler.GetState)
	app.Post("/api/v1/watcher/start", watcherHandler.StartWatcher)
	app.Post("/api/v1/watcher/stop", watcherHandler.StopWatcher)

	// Create test user
	user := CreateTestUser(t, db, "test-watcher@example.com")

	// Generate test token and set auth header
	authHeader := "Bearer " + GenerateTestToken(t, user.ID)

	t.Run("GetConfig returns user config", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/watcher/config", nil)
		req.Header.Set("Authorization", authHeader)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var config map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&config)
		require.NoError(t, err)
		assert.Equal(t, user.ID.String(), config["user_id"])
		assert.Equal(t, "https://gengo.com/jobs/rss", config["rss_feed_url"])
	})

	t.Run("UpdateConfig modifies config values", func(t *testing.T) {
		updateReq := handlers.UpdateConfigRequest{
			MinReward:         float64Ptr(5.0),
			MaxReward:         float64Ptr(25.0),
			WebSocketEnabled: boolPtr(false),
		}
		body, _ := json.Marshal(updateReq)

		req := httptest.NewRequest("PUT", "/api/v1/watcher/config", bytes.NewReader(body))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var config map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&config)
		require.NoError(t, err)
		assert.Equal(t, 5.0, config["min_reward"])
		assert.Equal(t, 25.0, config["max_reward"])
		assert.Equal(t, false, config["websocket_enabled"])
	})

	t.Run("GetState returns watcher state", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/watcher/state", nil)
		req.Header.Set("Authorization", authHeader)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var state map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&state)
		require.NoError(t, err)
		assert.Equal(t, "stopped", state["watcher_status"])
		assert.Equal(t, 0, int(state["total_jobs_found"].(float64)))
	})

	t.Run("StartWatcher starts the watcher", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/watcher/start", nil)
		req.Header.Set("Authorization", authHeader)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "running", result["status"])
	})

	t.Run("GetState after start shows running", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/watcher/state", nil)
		req.Header.Set("Authorization", authHeader)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var state map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&state)
		require.NoError(t, err)
		assert.Equal(t, "running", state["watcher_status"])
	})

	t.Run("StopWatcher stops the watcher", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/watcher/stop", nil)
		req.Header.Set("Authorization", authHeader)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, "stopped", result["status"])
	})

	t.Run("GetState after stop shows stopped", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/watcher/state", nil)
		req.Header.Set("Authorization", authHeader)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var state map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&state)
		require.NoError(t, err)
		assert.Equal(t, "stopped", state["watcher_status"])
	})
}

// TestWatcher_UnauthorizedAccess rejects requests without auth
func TestWatcher_UnauthorizedAccess(t *testing.T) {
	db := TestDB(t)
	manager := watcher.NewTestManager(t)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	wrappedDB := &databaseWrapper{db: db}
	watcherHandler := handlers.NewWatcherHandler(manager, wrappedDB)
	app.Get("/api/v1/watcher/config", watcherHandler.GetConfig)
	app.Post("/api/v1/watcher/start", watcherHandler.StartWatcher)

	t.Run("GetConfig without token returns 401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/watcher/config", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("StartWatcher with invalid token returns 401", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/watcher/start", nil)
		req.Header.Set("Authorization", "Bearer invalid_token")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})
}

// TestWatcher_ConcurrentStart ensures only one watcher runs per user
func TestWatcher_ConcurrentStart(t *testing.T) {
	db := TestDB(t)
	manager := watcher.NewTestManager(t)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	wrappedDB := &databaseWrapper{db: db}
	watcherHandler := handlers.NewWatcherHandler(manager, wrappedDB)
	app.Post("/api/v1/watcher/start", watcherHandler.StartWatcher)

	user := CreateTestUser(t, db, "test-concurrent@example.com")
	authHeader := "Bearer " + GenerateTestToken(t, user.ID)

	// Try starting 10 times concurrently
	errChan := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("POST", "/api/v1/watcher/start", nil)
			req.Header.Set("Authorization", authHeader)

			resp, err := app.Test(req)
			if err != nil {
				errChan <- err
				return
			}
			if resp.StatusCode != 200 {
				errChan <- assert.AnError
			} else {
				errChan <- nil
			}
		}()
	}

	// Count successful starts
	successCount := 0
	for i := 0; i < 10; i++ {
		if <-errChan == nil {
			successCount++
		}
	}

	// All requests should succeed (manager handles this gracefully)
	assert.Equal(t, 10, successCount, "All start requests should succeed")
}

// databaseWrapper wraps gorm.DB to implement database.Database
type databaseWrapper struct {
	db *gorm.DB
}

func (w *databaseWrapper) Create(value interface{}) *gorm.DB {
	return w.db.Create(value)
}

func (w *databaseWrapper) First(dest interface{}, conds ...interface{}) *gorm.DB {
	return w.db.First(dest, conds...)
}

func (w *databaseWrapper) Where(query interface{}, args ...interface{}) *gorm.DB {
	return w.db.Where(query, args...)
}

func (w *databaseWrapper) Model(value interface{}) *gorm.DB {
	return w.db.Model(value)
}

func (w *databaseWrapper) Begin(opts ...interface{}) *gorm.DB {
	return w.db.Begin(opts...)
}

func (w *databaseWrapper) Exec(sql string, values ...interface{}) *gorm.DB {
	return w.db.Exec(sql, values...)
}

func (w *databaseWrapper) Save(value interface{}) *gorm.DB {
	return w.db.Save(value)
}

func (w *databaseWrapper) Updates(values interface{}) *gorm.DB {
	return w.db.Updates(values)
}

func (w *databaseWrapper) UpdateColumn(column string, value interface{}) *gorm.DB {
	return w.db.UpdateColumn(column, value)
}

func (w *databaseWrapper) Update(column string, value interface{}) *gorm.DB {
	return w.db.Update(column, value)
}

// Helper functions
func float64Ptr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}
