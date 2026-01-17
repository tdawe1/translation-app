package tests

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/handlers"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/models"
)

type multiTenancyTestEnv struct {
	app    *fiber.App
	db     database.Database
	redis  *redis.Client
	userA  *models.User
	userB  *models.User
	tokenA string
	tokenB string
}

const testJWTSecret = "test-secret-for-testing-only-32-chars-min"

func setupMultiTenancyTestEnv(t *testing.T) multiTenancyTestEnv {
	t.Setenv("JWT_SECRET", testJWTSecret)

	db := RequireDB(t)
	redisClient := RequireRedis(t)
	require.NotNil(t, redisClient, "Redis required for multi-tenancy tests")

	wrappedDB := database.Wrap(db)

	userA := CreateTestUser(t, db, "usera-test@example.com")
	userB := CreateTestUser(t, db, "userb-test@example.com")

	tokenA := GenerateTestToken(t, userA.ID)
	tokenB := GenerateTestToken(t, userB.ID)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	jwtCfg := middleware.NewJWTConfig(middleware.WithSecret(testJWTSecret))
	jwtMiddleware := middleware.JWTValidator(jwtCfg)

	transHandler := handlers.NewTranslationHandler(wrappedDB, redisClient)
	watcherHandler := handlers.NewWatcherHandler(nil, wrappedDB)

	transGroup := app.Group("/api/v1/translation", jwtMiddleware)
	transGroup.Get("/jobs", transHandler.ListJobs)
	transGroup.Get("/jobs/:id", transHandler.GetJob)
	transGroup.Delete("/jobs/:id", transHandler.DeleteJob)
	transGroup.Post("/jobs", transHandler.CreateJob)

	watcherGroup := app.Group("/api/v1/watcher", jwtMiddleware)
	watcherGroup.Get("/config", watcherHandler.GetConfig)
	watcherGroup.Put("/config", watcherHandler.UpdateConfig)
	watcherGroup.Get("/state", watcherHandler.GetState)

	return multiTenancyTestEnv{
		app:    app,
		db:     wrappedDB,
		redis:  redisClient,
		userA:  userA,
		userB:  userB,
		tokenA: tokenA,
		tokenB: tokenB,
	}
}

func createTestJob(t *testing.T, env multiTenancyTestEnv, user *models.User, status models.TranslationJobStatus) *models.TranslationJob {
	job := &models.TranslationJob{
		UserID:       user.ID,
		SourceFile:   "sample.docx",
		SourceLang:   "ja",
		TargetLang:   "en",
		ProjectType:  "routine",
		ApprovalMode: "async",
		Status:       status,
	}
	require.NoError(t, env.db.Create(job).Error)
	return job
}

func TestMultiTenancy_UserCannotAccessOtherUserJobs(t *testing.T) {
	env := setupMultiTenancyTestEnv(t)

	jobA := createTestJob(t, env, env.userA, models.TranslationJobStatusPending)

	t.Run("User A can access their own job", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/translation/jobs/"+jobA.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+env.tokenA)

		resp, err := env.app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, env.userA.ID.String(), result["user_id"])
		assert.Equal(t, "sample.docx", result["source_file"])
	})

	t.Run("User B CANNOT access User A's job (returns 404, not 403)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/translation/jobs/"+jobA.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+env.tokenB)

		resp, err := env.app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode, "Should return 404 to prevent account enumeration")

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Contains(t, result["error"], "not found", "Error message should be generic")
	})
}

func TestMultiTenancy_UserCannotDeleteOtherUserJobs(t *testing.T) {
	env := setupMultiTenancyTestEnv(t)

	jobA := createTestJob(t, env, env.userA, models.TranslationJobStatusCompleted)

	t.Run("User A can delete their own job", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/translation/jobs/"+jobA.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+env.tokenA)

		resp, err := env.app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("User B CANNOT delete User A's job (returns 404)", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/translation/jobs/"+jobA.ID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+env.tokenB)

		resp, err := env.app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode, "Should return 404 to prevent account enumeration")

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Contains(t, result["error"], "not found")
	})
}

func TestMultiTenancy_UserCannotListOtherUserJobs(t *testing.T) {
	env := setupMultiTenancyTestEnv(t)

	createTestJob(t, env, env.userA, models.TranslationJobStatusPending)
	createTestJob(t, env, env.userA, models.TranslationJobStatusProcessing)
	createTestJob(t, env, env.userB, models.TranslationJobStatusPending)
	createTestJob(t, env, env.userB, models.TranslationJobStatusCompleted)

	t.Run("User A sees only their own jobs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/translation/jobs", nil)
		req.Header.Set("Authorization", "Bearer "+env.tokenA)

		resp, err := env.app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, float64(2), result["total_count"], "User A should see exactly 2 jobs")

		jobs := result["jobs"].([]interface{})
		assert.Len(t, jobs, 2)

		for _, job := range jobs {
			jobData := job.(map[string]interface{})
			assert.Equal(t, env.userA.ID.String(), jobData["user_id"], "All jobs should belong to User A")
		}
	})

	t.Run("User B sees only their own jobs", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/translation/jobs", nil)
		req.Header.Set("Authorization", "Bearer "+env.tokenB)

		resp, err := env.app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, float64(2), result["total_count"], "User B should see exactly 2 jobs")

		jobs := result["jobs"].([]interface{})
		assert.Len(t, jobs, 2)

		for _, job := range jobs {
			jobData := job.(map[string]interface{})
			assert.Equal(t, env.userB.ID.String(), jobData["user_id"], "All jobs should belong to User B")
		}
	})
}

func TestMultiTenancy_WatcherIsolation(t *testing.T) {
	env := setupMultiTenancyTestEnv(t)

	t.Run("User A can get their own watcher config", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/watcher/config", nil)
		req.Header.Set("Authorization", "Bearer "+env.tokenA)

		resp, err := env.app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, env.userA.ID.String(), result["user_id"])
	})

	t.Run("User B can get their own watcher config", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/watcher/config", nil)
		req.Header.Set("Authorization", "Bearer "+env.tokenB)

		resp, err := env.app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, env.userB.ID.String(), result["user_id"])
	})

	t.Run("User A updates only their own config", func(t *testing.T) {
		payload := map[string]interface{}{
			"min_reward": 2.0,
			"max_reward": 30.0,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("PUT", "/api/v1/watcher/config", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+env.tokenA)
		req.Header.Set("Content-Type", "application/json")

		resp, err := env.app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, env.userA.ID.String(), result["user_id"])
		assert.Equal(t, 2.0, result["min_reward"])
	})

	t.Run("User B updates only their own config", func(t *testing.T) {
		payload := map[string]interface{}{
			"min_reward": 5.0,
			"max_reward": 40.0,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest("PUT", "/api/v1/watcher/config", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+env.tokenB)
		req.Header.Set("Content-Type", "application/json")

		resp, err := env.app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, env.userB.ID.String(), result["user_id"])
		assert.Equal(t, 5.0, result["min_reward"])
	})

	t.Run("Verify configs remain isolated after updates", func(t *testing.T) {
		reqA := httptest.NewRequest("GET", "/api/v1/watcher/config", nil)
		reqA.Header.Set("Authorization", "Bearer "+env.tokenA)

		respA, err := env.app.Test(reqA)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, respA.StatusCode)

		var configA map[string]interface{}
		require.NoError(t, json.NewDecoder(respA.Body).Decode(&configA))
		assert.Equal(t, 2.0, configA["min_reward"], "User A's config should have their own value")

		reqB := httptest.NewRequest("GET", "/api/v1/watcher/config", nil)
		reqB.Header.Set("Authorization", "Bearer "+env.tokenB)

		respB, err := env.app.Test(reqB)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, respB.StatusCode)

		var configB map[string]interface{}
		require.NoError(t, json.NewDecoder(respB.Body).Decode(&configB))
		assert.Equal(t, 5.0, configB["min_reward"], "User B's config should have their own value")
	})
}

func TestMultiTenancy_NonExistentJobReturns404ForAllUsers(t *testing.T) {
	env := setupMultiTenancyTestEnv(t)

	nonExistentID := uuid.New()

	t.Run("User A gets 404 for non-existent job", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/translation/jobs/"+nonExistentID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+env.tokenA)

		resp, err := env.app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("User B gets 404 for non-existent job", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/translation/jobs/"+nonExistentID.String(), nil)
		req.Header.Set("Authorization", "Bearer "+env.tokenB)

		resp, err := env.app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("Both users receive identical error messages", func(t *testing.T) {
		reqA := httptest.NewRequest("GET", "/api/v1/translation/jobs/"+nonExistentID.String(), nil)
		reqA.Header.Set("Authorization", "Bearer "+env.tokenA)
		respA, _ := env.app.Test(reqA)

		reqB := httptest.NewRequest("GET", "/api/v1/translation/jobs/"+nonExistentID.String(), nil)
		reqB.Header.Set("Authorization", "Bearer "+env.tokenB)
		respB, _ := env.app.Test(reqB)

		var resultA, resultB map[string]interface{}
		json.NewDecoder(respA.Body).Decode(&resultA)
		json.NewDecoder(respB.Body).Decode(&resultB)

		assert.Equal(t, resultA["error"], resultB["error"], "Error messages should be identical to prevent enumeration")
	})
}
