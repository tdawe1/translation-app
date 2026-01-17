package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/tdawe1/translation-app/internal/database"
	apperrors "github.com/tdawe1/translation-app/internal/errors"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/models"
)

type TranslationHandler struct {
	db    database.Database
	redis *redis.Client
}

func NewTranslationHandler(db database.Database, redisClient *redis.Client) *TranslationHandler {
	return &TranslationHandler{
		db:    db,
		redis: redisClient,
	}
}

type CreateJobRequest struct {
	SourceFile   string `json:"source_file" validate:"required"`
	SourceLang   string `json:"source_lang,omitempty"`
	TargetLang   string `json:"target_lang,omitempty"`
	ProjectType  string `json:"project_type,omitempty"`
	ApprovalMode string `json:"approval_mode,omitempty"`
}

type UpdateSegmentRequest struct {
	Target string `json:"target" validate:"required"`
}

type RejectJobRequest struct {
	Reason string `json:"reason"`
}

func normalizeCreateJobRequest(req *CreateJobRequest) string {
	if req == nil {
		return "Invalid request body"
	}
	if req.SourceFile == "" {
		return "source_file is required"
	}
	if req.SourceLang == "" {
		req.SourceLang = "ja"
	}
	if req.TargetLang == "" {
		req.TargetLang = "en"
	}
	if req.ProjectType == "" {
		req.ProjectType = "routine"
	}
	if req.ApprovalMode == "" {
		req.ApprovalMode = "async"
	}
	if req.ProjectType != "critical" && req.ProjectType != "routine" {
		return "project_type must be 'critical' or 'routine'"
	}
	if req.ApprovalMode != "blocking" && req.ApprovalMode != "async" {
		return "approval_mode must be 'blocking' or 'async'"
	}
	return ""
}

func loadJobForUser(db database.Database, jobID uuid.UUID, userID uuid.UUID, preloadSegments bool) (*models.TranslationJob, error) {
	var job models.TranslationJob
	query := db.Where("id = ? AND user_id = ?", jobID, userID)
	if preloadSegments {
		query = query.Preload("Segments")
	}
	if err := query.First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

type redisJobMetadata struct {
	JobID  string `json:"job_id"`
	UserID string `json:"user_id"`
}

func redisMetadataMatches(job *models.TranslationJob, data map[string]string) bool {
	if job == nil {
		return false
	}

	jobID := ""
	userID := ""
	if data != nil {
		jobID = data["job_id"]
		userID = data["user_id"]
	}

	if jobID == "" || userID == "" {
		if data == nil {
			return false
		}
		payload, ok := data["data"]
		if !ok || payload == "" {
			return false
		}
		var meta redisJobMetadata
		if err := json.Unmarshal([]byte(payload), &meta); err != nil {
			return false
		}
		if jobID == "" {
			jobID = meta.JobID
		}
		if userID == "" {
			userID = meta.UserID
		}
	}

	if jobID == "" || userID == "" {
		return false
	}

	return jobID == job.ID.String() && userID == job.UserID.String()
}

func (h *TranslationHandler) ListJobs(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.listJobsLogic)(c)
}

func (h *TranslationHandler) listJobsLogic(c *fiber.Ctx, userID uuid.UUID) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	status := c.Query("status", "")
	sort := c.Query("sort", "newest")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	if sort != "newest" && sort != "oldest" {
		sort = "newest"
	}

	offset := (page - 1) * pageSize
	order := "created_at DESC"
	if sort == "oldest" {
		order = "created_at ASC"
	}

	query := h.db.Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var totalCount int64
	if err := query.Model(&models.TranslationJob{}).Count(&totalCount).Error; err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to count jobs")
	}

	var jobs []models.TranslationJob
	if err := query.Order(order).Offset(offset).Limit(pageSize).Find(&jobs).Error; err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to fetch jobs")
	}

	jobResponses := make([]map[string]interface{}, len(jobs))
	for i, job := range jobs {
		jobResponses[i] = jobToResponse(&job)
	}

	return c.JSON(fiber.Map{
		"jobs":        jobResponses,
		"total_count": totalCount,
		"page":        page,
		"page_size":   pageSize,
	})
}

func (h *TranslationHandler) GetJob(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.getJobLogic)(c)
}

func (h *TranslationHandler) getJobLogic(c *fiber.Ctx, userID uuid.UUID) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidJobID, "Invalid job ID")
	}

	job, err := loadJobForUser(h.db, jobID, userID, true)
	if err != nil {
		return RespondWithError(c, fiber.StatusNotFound, apperrors.ErrJobNotFound, "Job not found")
	}

	if !job.IsTerminal() && job.RedisJobID != "" {
		h.syncJobStatusFromRedis(c.Context(), job)
		h.syncSegmentsFromRedis(c.Context(), job)
	}

	return c.JSON(jobToResponse(job))
}

func validateSourceFile(sourcePath string) (string, error) {
	uploadsDir := "/app/data/uploads"

	cleanPath := filepath.Clean(sourcePath)

	if filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("absolute paths not allowed")
	}

	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "/../") {
		return "", fmt.Errorf("path traversal not allowed")
	}

	fullPath := filepath.Join(uploadsDir, cleanPath)

	if !strings.HasPrefix(fullPath, uploadsDir) {
		return "", fmt.Errorf("path must be within uploads directory")
	}

	return cleanPath, nil
}

func (h *TranslationHandler) CreateJob(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.createJobLogic)(c)
}

func (h *TranslationHandler) createJobLogic(c *fiber.Ctx, userID uuid.UUID) error {
	var req CreateJobRequest
	if err := c.BodyParser(&req); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid request body")
	}

	if msg := normalizeCreateJobRequest(&req); msg != "" {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, msg)
	}

	validatedSourceFile, err := validateSourceFile(req.SourceFile)
	if err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, err.Error())
	}

	job := models.TranslationJob{
		UserID:       userID,
		SourceFile:   validatedSourceFile,
		SourceLang:   req.SourceLang,
		TargetLang:   req.TargetLang,
		ProjectType:  req.ProjectType,
		ApprovalMode: req.ApprovalMode,
		Status:       models.TranslationJobStatusPending,
	}

	if err := h.db.Create(&job).Error; err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to create job")
	}

	redisJobID := fmt.Sprintf("user:%s:trans:%s", userID.String(), job.ID.String())
	job.RedisJobID = redisJobID
	if err := h.db.Model(&job).Update("redis_job_id", redisJobID).Error; err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrUpdateError, "Failed to update job")
	}

	jobData := map[string]interface{}{
		"id":            job.ID.String(),
		"job_id":        job.ID.String(),
		"user_id":       userID.String(),
		"source_file":   req.SourceFile,
		"source_lang":   req.SourceLang,
		"target_lang":   req.TargetLang,
		"project_type":  req.ProjectType,
		"approval_mode": req.ApprovalMode,
		"status":        "pending",
		"created_at":    job.CreatedAt.Format(time.RFC3339),
	}

	jobJSON, err := json.Marshal(jobData)
	if err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Failed to process job")
	}

	if err := h.redis.HSet(c.Context(), redisJobID, map[string]interface{}{
		"data":    string(jobJSON),
		"status":  "pending",
		"job_id":  job.ID.String(),
		"user_id": userID.String(),
	}).Err(); err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Failed to queue job")
	}

	if err := h.redis.LPush(c.Context(), fmt.Sprintf("user:%s:trans:queue", userID.String()), redisJobID).Err(); err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrInternal, "Failed to queue job")
	}

	return c.JSON(jobToResponse(&job))
}

func (h *TranslationHandler) ApproveJob(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.approveJobLogic)(c)
}

func (h *TranslationHandler) approveJobLogic(c *fiber.Ctx, userID uuid.UUID) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidJobID, "Invalid job ID")
	}

	job, err := loadJobForUser(h.db, jobID, userID, false)
	if err != nil {
		return RespondWithError(c, fiber.StatusNotFound, apperrors.ErrJobNotFound, "Job not found")
	}

	if !job.CanApprove() {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Job cannot be approved in the current status")
	}

	approvedAt := time.Now()
	updates := map[string]interface{}{
		"status":      models.TranslationJobStatusApproved,
		"approved_at": approvedAt,
		"approved_by": userID.String(),
	}

	if err := h.db.Model(job).Updates(updates).Error; err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrUpdateError, "Failed to approve job")
	}

	if job.RedisJobID != "" {
		h.redis.HSet(c.Context(), job.RedisJobID, "status", "approved")
	}

	h.db.Where("id = ?", jobID).Preload("Segments").First(job)

	return c.JSON(jobToResponse(job))
}

func (h *TranslationHandler) RejectJob(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.rejectJobLogic)(c)
}

func (h *TranslationHandler) rejectJobLogic(c *fiber.Ctx, userID uuid.UUID) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidJobID, "Invalid job ID")
	}

	job, err := loadJobForUser(h.db, jobID, userID, false)
	if err != nil {
		return RespondWithError(c, fiber.StatusNotFound, apperrors.ErrJobNotFound, "Job not found")
	}

	if job.Status != models.TranslationJobStatusPendingApproval {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Job cannot be rejected in the current status")
	}

	var req RejectJobRequest
	if err := c.BodyParser(&req); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid request body")
	}

	updates := map[string]interface{}{
		"status": models.TranslationJobStatusRejected,
		"error":  req.Reason,
	}

	if err := h.db.Model(job).Updates(updates).Error; err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrUpdateError, "Failed to reject job")
	}

	if job.RedisJobID != "" {
		h.redis.HSet(c.Context(), job.RedisJobID, "status", "rejected")
	}

	h.db.Where("id = ?", jobID).First(job)

	return c.JSON(jobToResponse(job))
}

func (h *TranslationHandler) UpdateSegment(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.updateSegmentLogic)(c)
}

func (h *TranslationHandler) updateSegmentLogic(c *fiber.Ctx, userID uuid.UUID) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidJobID, "Invalid job ID")
	}

	segmentID, err := uuid.Parse(c.Params("segment_uuid"))
	if err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidSegmentID, "Invalid segment UUID")
	}

	var req UpdateSegmentRequest
	if err := c.BodyParser(&req); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid request body")
	}

	if req.Target == "" {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "target is required")
	}

	var job models.TranslationJob
	if err := h.db.Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
		return RespondWithError(c, fiber.StatusNotFound, apperrors.ErrJobNotFound, "Job not found")
	}

	var segment models.TranslationSegment
	if err := h.db.Where("id = ? AND job_id = ?", segmentID, jobID).First(&segment).Error; err != nil {
		return RespondWithError(c, fiber.StatusNotFound, apperrors.ErrSegmentNotFound, "Segment not found")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"target":       req.Target,
		"judge_winner": "edited",
		"edited_by":    userID.String(),
		"edited_at":    now,
		"is_flagged":   false,
	}

	if err := h.db.Model(&segment).Updates(updates).Error; err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrUpdateError, "Failed to update segment")
	}

	if job.RedisJobID != "" {
		ctx := c.Context()
		segmentsKey := fmt.Sprintf("%s:segments", job.RedisJobID)
		segmentData, err := h.redis.LRange(ctx, segmentsKey, 0, -1).Result()
		if err == nil {
			for i, data := range segmentData {
				var payload map[string]interface{}
				if err := json.Unmarshal([]byte(data), &payload); err != nil {
					continue
				}
				segmentKey, ok := payload["segment_id"].(string)
				if !ok || segmentKey != segment.SegmentID {
					continue
				}
				payload["target"] = req.Target
				payload["judge_winner"] = "edited"
				payload["is_flagged"] = false
				payload["edited_by"] = userID.String()
				payload["edited_at"] = now.Format(time.RFC3339)
				if jsonData, err := json.Marshal(payload); err == nil {
					h.redis.LSet(ctx, segmentsKey, int64(i), jsonData)
				}
				break
			}
		}
	}

	var flaggedCount int64
	if err := h.db.Model(&models.TranslationSegment{}).
		Where("job_id = ? AND is_flagged = ?", jobID, true).
		Count(&flaggedCount).Error; err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to count flagged segments")
	}

	if err := h.db.Model(&job).Updates(map[string]interface{}{
		"flagged_count":  flaggedCount,
		"has_user_edits": true,
	}).Error; err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrUpdateError, "Failed to update job")
	}

	h.db.Where("id = ?", segmentID).First(&segment)

	return c.JSON(segmentToResponse(&segment))
}

func (h *TranslationHandler) GetFlaggedSegments(c *fiber.Ctx) error {

	return middleware.RequireAuth(h.getFlaggedSegmentsLogic)(c)
}

func (h *TranslationHandler) getFlaggedSegmentsLogic(c *fiber.Ctx, userID uuid.UUID) error {
	jobID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidJobID, "Invalid job ID")
	}

	var job models.TranslationJob
	if err := h.db.Where("id = ? AND user_id = ?", jobID, userID).First(&job).Error; err != nil {
		return RespondWithError(c, fiber.StatusNotFound, apperrors.ErrJobNotFound, "Job not found")
	}

	var segments []models.TranslationSegment
	if err := h.db.Where("job_id = ? AND is_flagged = ?", jobID, true).
		Order("created_at ASC").
		Find(&segments).Error; err != nil {
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to fetch segments")
	}

	response := make([]map[string]interface{}, len(segments))
	for i, seg := range segments {
		response[i] = segmentToResponse(&seg)
	}

	return c.JSON(fiber.Map{
		"job_id":   jobID.String(),
		"segments": response,
		"count":    len(segments),
	})
}

func (h *TranslationHandler) syncJobStatusFromRedis(ctx context.Context, job *models.TranslationJob) {
	if job.RedisJobID == "" {
		return
	}

	data, err := h.redis.HGetAll(ctx, job.RedisJobID).Result()
	if err != nil || len(data) == 0 {
		return
	}

	if !redisMetadataMatches(job, data) {
		return
	}

	if status, ok := data["status"]; ok {
		switch status {
		case "processing":
			job.Status = models.TranslationJobStatusProcessing
		case "translating":
			job.Status = models.TranslationJobStatusTranslating
		case "pending_approval":
			job.Status = models.TranslationJobStatusPendingApproval
		case "completed":
			job.Status = models.TranslationJobStatusCompleted
		case "failed":
			job.Status = models.TranslationJobStatusFailed
		}
	}

	if progressStr, ok := data["progress"]; ok {
		if progress, err := strconv.ParseFloat(progressStr, 64); err == nil {
			job.Progress = progress
		}
	}

	h.db.Model(&models.TranslationJob{}).
		Where("id = ? AND user_id = ?", job.ID, job.UserID).
		Updates(map[string]interface{}{
			"status":   job.Status,
			"progress": job.Progress,
		})
}

func (h *TranslationHandler) syncSegmentsFromRedis(ctx context.Context, job *models.TranslationJob) {
	segmentsKey := fmt.Sprintf("%s:segments", job.RedisJobID)
	segmentData, err := h.redis.LRange(ctx, segmentsKey, 0, -1).Result()
	if err != nil || len(segmentData) == 0 {
		return
	}

	data, err := h.redis.HGetAll(ctx, job.RedisJobID).Result()
	if err != nil || len(data) == 0 {
		return
	}

	if !redisMetadataMatches(job, data) {
		return
	}

	var existingSegments []models.TranslationSegment
	h.db.Where("job_id = ?", job.ID).Find(&existingSegments)

	segmentMap := make(map[string]*models.TranslationSegment)
	for i := range existingSegments {
		segmentMap[existingSegments[i].SegmentID] = &existingSegments[i]
	}

	for _, data := range segmentData {
		var seg struct {
			SegmentID       string  `json:"segment_id"`
			JobID           string  `json:"job_id"`
			UserID          string  `json:"user_id"`
			Source          string  `json:"source"`
			Target          string  `json:"target"`
			JudgeWinner     string  `json:"judge_winner"`
			JudgeConfidence float64 `json:"judge_confidence"`
			JudgeReasoning  string  `json:"judge_reasoning"`
			IsFlagged       bool    `json:"is_flagged"`
			FlagReason      string  `json:"flag_reason"`
			ModelAOutput    string  `json:"model_a_output"`
			ModelBOutput    string  `json:"model_b_output"`
		}

		if err := json.Unmarshal([]byte(data), &seg); err != nil {
			continue
		}

		if seg.JobID != "" && seg.JobID != job.ID.String() {
			continue
		}
		if seg.UserID != "" && seg.UserID != job.UserID.String() {
			continue
		}

		existing, ok := segmentMap[seg.SegmentID]
		if !ok {
			segment := models.TranslationSegment{
				Base:            models.Base{ID: uuid.New()},
				JobID:           job.ID,
				SegmentID:       seg.SegmentID,
				Source:          seg.Source,
				Target:          seg.Target,
				JudgeWinner:     seg.JudgeWinner,
				JudgeConfidence: seg.JudgeConfidence,
				JudgeReasoning:  seg.JudgeReasoning,
				IsFlagged:       seg.IsFlagged,
				FlagReason:      seg.FlagReason,
				ModelAOutput:    seg.ModelAOutput,
				ModelBOutput:    seg.ModelBOutput,
			}
			h.db.Create(&segment)
		} else if !job.HasUserEdits {
			h.db.Model(existing).Updates(map[string]interface{}{
				"target":           seg.Target,
				"judge_winner":     seg.JudgeWinner,
				"judge_confidence": seg.JudgeConfidence,
				"judge_reasoning":  seg.JudgeReasoning,
				"is_flagged":       seg.IsFlagged,
				"flag_reason":      seg.FlagReason,
				"model_a_output":   seg.ModelAOutput,
				"model_b_output":   seg.ModelBOutput,
			})
		}
	}

	var counts struct {
		SegmentCount int64
		FlaggedCount int64
	}
	h.db.Model(&models.TranslationSegment{}).
		Select("COUNT(*) as segment_count, SUM(CASE WHEN is_flagged = true THEN 1 ELSE 0 END) as flagged_count").
		Where("job_id = ?", job.ID).
		Scan(&counts)
	h.db.Model(job).Updates(map[string]interface{}{
		"segment_count": counts.SegmentCount,
		"flagged_count": counts.FlaggedCount,
	})
}

func jobToResponse(job *models.TranslationJob) map[string]interface{} {
	segments := make([]map[string]interface{}, len(job.Segments))
	for i, seg := range job.Segments {
		segments[i] = segmentToResponse(&seg)
	}

	resp := map[string]interface{}{
		"id":                job.ID.String(),
		"user_id":           job.UserID.String(),
		"source_file":       job.SourceFile,
		"target_file":       job.TargetFile,
		"source_lang":       job.SourceLang,
		"target_lang":       job.TargetLang,
		"status":            job.Status,
		"project_type":      job.ProjectType,
		"approval_mode":     job.ApprovalMode,
		"overall_score":     job.OverallScore,
		"segment_count":     job.SegmentCount,
		"flagged_count":     job.FlaggedCount,
		"judge_resolutions": job.JudgeResolutions,
		"progress":          job.Progress,
		"created_at":        job.CreatedAt,
		"updated_at":        job.UpdatedAt,
	}

	if job.Error != "" {
		resp["error"] = job.Error
	}
	if job.WorkerID != "" {
		resp["worker_id"] = job.WorkerID
	}
	if job.CompletedAt != nil {
		resp["completed_at"] = job.CompletedAt
	}
	if job.ApprovedAt != nil {
		resp["approved_at"] = job.ApprovedAt
		resp["approved_by"] = job.ApprovedBy
	}
	if len(segments) > 0 {
		resp["segments"] = segments
	}

	return resp
}

func segmentToResponse(seg *models.TranslationSegment) map[string]interface{} {
	resp := map[string]interface{}{
		"id":               seg.ID.String(),
		"job_id":           seg.JobID.String(),
		"segment_id":       seg.SegmentID,
		"source":           seg.Source,
		"target":           seg.Target,
		"judge_winner":     seg.JudgeWinner,
		"judge_confidence": seg.JudgeConfidence,
		"is_flagged":       seg.IsFlagged,
		"created_at":       seg.CreatedAt,
		"updated_at":       seg.UpdatedAt,
	}

	if seg.Context != "" && seg.Context != "{}" {
		resp["context"] = seg.Context
	}
	if seg.JudgeReasoning != "" {
		resp["judge_reasoning"] = seg.JudgeReasoning
	}
	if seg.FlagReason != "" {
		resp["flag_reason"] = seg.FlagReason
	}
	if seg.ModelAOutput != "" {
		resp["model_a_output"] = seg.ModelAOutput
	}
	if seg.ModelBOutput != "" {
		resp["model_b_output"] = seg.ModelBOutput
	}
	if seg.GlossaryTerms != "" && seg.GlossaryTerms != "[]" {
		resp["glossary_terms"] = seg.GlossaryTerms
	}
	if seg.EditedBy != "" {
		resp["edited_by"] = seg.EditedBy
		resp["edited_at"] = seg.EditedAt
	}

	return resp
}
