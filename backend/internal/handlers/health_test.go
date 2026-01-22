package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/tests"
)

func TestHealthHandler_Health_AllHealthy(t *testing.T) {
	db := tests.RequireDB(t)
	redisClient := tests.RequireRedis(t)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	handler := NewHealthHandler(db, redisClient)
	app.Get("/health", handler.Health)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, "healthy", body["db"])
	assert.Equal(t, "healthy", body["redis"])
}

func TestHealthHandler_Health_DBUnhealthy(t *testing.T) {
	db := tests.RequireDB(t)
	redisClient := tests.RequireRedis(t)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	handler := NewHealthHandler(db, redisClient)
	app.Get("/health", handler.Health)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close()

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "degraded", body["status"])
	assert.Equal(t, "unhealthy", body["db"])
	assert.Equal(t, "healthy", body["redis"])
}

func TestHealthHandler_Health_RedisUnhealthy(t *testing.T) {
	db := tests.RequireDB(t)
	redisClient := tests.RequireRedis(t)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	handler := NewHealthHandler(db, redisClient)
	app.Get("/health", handler.Health)

	redisClient.Close()

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "degraded", body["status"])
	assert.Equal(t, "healthy", body["db"])
	assert.Equal(t, "unhealthy", body["redis"])
}

func TestHealthHandler_Health_BothUnhealthy(t *testing.T) {
	db := tests.RequireDB(t)
	redisClient := tests.RequireRedis(t)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	handler := NewHealthHandler(db, redisClient)
	app.Get("/health", handler.Health)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close()
	redisClient.Close()

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "degraded", body["status"])
	assert.Equal(t, "unhealthy", body["db"])
	assert.Equal(t, "unhealthy", body["redis"])
}

func TestHealthHandler_Health_Timeout(t *testing.T) {
	db := tests.RequireDB(t)
	redisClient := tests.RequireRedis(t)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
		IdleTimeout:           100 * time.Millisecond,
		ReadTimeout:           100 * time.Millisecond,
		WriteTimeout:          100 * time.Millisecond,
	})

	handler := NewHealthHandler(db, redisClient)
	app.Get("/health", handler.Health)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)

	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, "healthy", body["db"])
	assert.Equal(t, "healthy", body["redis"])
}
