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
	"gorm.io/gorm"

	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/handlers"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/models"
)

type translationTestEnv struct {
	app   *fiber.App
	db    *gorm.DB
	redis *redis.Client
	user  *models.User
	token string
}

func setupTranslationTestEnv(t *testing.T) translationTestEnv {
	db := RequireDB(t)
	redisClient := RequireRedis(t)
	require.NotNil(t, redisClient, "Redis required for translation tests")

	wrappedDB := database.Wrap(db)

	user := CreateTestUser(t, db, "translation-test@example.com")
	token := GenerateTestToken(t, user.ID)

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher Test",
		DisableStartupMessage: true,
	})

	jwtCfg := middleware.NewJWTConfig(middleware.WithSecret("test-secret-for-testing-only-32-chars-min"))
	jwtMiddleware := middleware.JWTValidator(jwtCfg)

	handler := handlers.NewTranslationHandler(wrappedDB, redisClient)

	group := app.Group("/api/v1/translation", jwtMiddleware)
	group.Get("/jobs", handler.ListJobs)
	group.Get("/jobs/:id", handler.GetJob)
	group.Post("/jobs", handler.CreateJob)
	group.Post("/jobs/:id/approve", handler.ApproveJob)
	group.Post("/jobs/:id/reject", handler.RejectJob)
	group.Put("/jobs/:id/segments/:segment_uuid", handler.UpdateSegment)
	group.Get("/jobs/:id/flagged", handler.GetFlaggedSegments)

	return translationTestEnv{
		app:   app,
		db:    db,
		redis: redisClient,
		user:  user,
		token: token,
	}
}

func createTranslationJob(t *testing.T, env translationTestEnv, status models.TranslationJobStatus) *models.TranslationJob {
	job := &models.TranslationJob{
		UserID:       env.user.ID,
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

func createTranslationSegment(t *testing.T, env translationTestEnv, jobID uuid.UUID, flagged bool) *models.TranslationSegment {
	segment := &models.TranslationSegment{
		JobID:       jobID,
		SegmentID:   "segment-1",
		Source:      "hello",
		Target:      "hola",
		JudgeWinner: "model_a",
		IsFlagged:   flagged,
	}
	require.NoError(t, env.db.Create(segment).Error)
	return segment
}

func TestTranslation_CreateJob(t *testing.T) {
	env := setupTranslationTestEnv(t)

	payload := []byte(`{"source_file":"sample.docx"}`)
	req := httptest.NewRequest("POST", "/api/v1/translation/jobs", bytes.NewBuffer(payload))
	req.Header.Set("Authorization", "Bearer "+env.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "pending", result["status"])
	assert.Equal(t, "sample.docx", result["source_file"])
}

func TestTranslation_ListJobs(t *testing.T) {
	env := setupTranslationTestEnv(t)
	createTranslationJob(t, env, models.TranslationJobStatusPending)
	createTranslationJob(t, env, models.TranslationJobStatusProcessing)

	req := httptest.NewRequest("GET", "/api/v1/translation/jobs?page=1&page_size=1&sort=newest", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)

	resp, err := env.app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, float64(1), result["page"])
	assert.Equal(t, float64(1), result["page_size"])
	assert.Equal(t, float64(2), result["total_count"])
	jobs := result["jobs"].([]interface{})
	assert.Len(t, jobs, 1)
}

func TestTranslation_GetJob(t *testing.T) {
	env := setupTranslationTestEnv(t)
	job := createTranslationJob(t, env, models.TranslationJobStatusPending)
	createTranslationSegment(t, env, job.ID, false)

	req := httptest.NewRequest("GET", "/api/v1/translation/jobs/"+job.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+env.token)

	resp, err := env.app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	segments := result["segments"].([]interface{})
	assert.Len(t, segments, 1)
}

func TestTranslation_ApproveJob(t *testing.T) {
	env := setupTranslationTestEnv(t)
	job := createTranslationJob(t, env, models.TranslationJobStatusPendingApproval)

	req := httptest.NewRequest("POST", "/api/v1/translation/jobs/"+job.ID.String()+"/approve", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)

	resp, err := env.app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var updated models.TranslationJob
	require.NoError(t, env.db.First(&updated, "id = ?", job.ID).Error)
	assert.Equal(t, models.TranslationJobStatusApproved, updated.Status)
}

func TestTranslation_RejectJob(t *testing.T) {
	env := setupTranslationTestEnv(t)
	job := createTranslationJob(t, env, models.TranslationJobStatusPendingApproval)

	payload := []byte(`{"reason":"quality"}`)
	req := httptest.NewRequest("POST", "/api/v1/translation/jobs/"+job.ID.String()+"/reject", bytes.NewBuffer(payload))
	req.Header.Set("Authorization", "Bearer "+env.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var updated models.TranslationJob
	require.NoError(t, env.db.First(&updated, "id = ?", job.ID).Error)
	assert.Equal(t, models.TranslationJobStatusRejected, updated.Status)
	assert.Equal(t, "quality", updated.Error)
}

func TestTranslation_UpdateSegment(t *testing.T) {
	env := setupTranslationTestEnv(t)
	job := createTranslationJob(t, env, models.TranslationJobStatusPendingApproval)
	segment := createTranslationSegment(t, env, job.ID, true)

	payload := []byte(`{"target":"updated"}`)
	req := httptest.NewRequest("PUT", "/api/v1/translation/jobs/"+job.ID.String()+"/segments/"+segment.ID.String(), bytes.NewBuffer(payload))
	req.Header.Set("Authorization", "Bearer "+env.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := env.app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "updated", result["target"])
	assert.Equal(t, false, result["is_flagged"])
}

func TestTranslation_GetFlaggedSegments(t *testing.T) {
	env := setupTranslationTestEnv(t)
	job := createTranslationJob(t, env, models.TranslationJobStatusPending)
	createTranslationSegment(t, env, job.ID, true)

	req := httptest.NewRequest("GET", "/api/v1/translation/jobs/"+job.ID.String()+"/flagged", nil)
	req.Header.Set("Authorization", "Bearer "+env.token)

	resp, err := env.app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, float64(1), result["count"])
	segments := result["segments"].([]interface{})
	assert.Len(t, segments, 1)
}
