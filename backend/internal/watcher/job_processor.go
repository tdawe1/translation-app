package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
)

// JobProcessor handles job filtering, deduplication, and publishing
type JobProcessor struct {
	db           database.Database
	redis        *redis.Client
	stateManager *StateManager
}

type JobProcessResult struct {
	Matched   bool
	Duplicate bool
	Filtered  bool
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(db database.Database, redis *redis.Client) *JobProcessor {
	return &JobProcessor{
		db:           db,
		redis:        redis,
		stateManager: NewStateManager(db),
	}
}

// ProcessJob handles a new job: checks filters, deduplicates, records, and publishes
func (p *JobProcessor) ProcessJob(ctx context.Context, job Job) error {
	_, err := p.ProcessJobWithResult(ctx, job)
	return err
}

// ProcessJobWithResult handles a job and reports whether it matched watcher filters.
func (p *JobProcessor) ProcessJobWithResult(ctx context.Context, job Job) (JobProcessResult, error) {
	result := JobProcessResult{}
	startTime := time.Now()
	log.Printf("[JOB-PROC] User %s: Processing job %s (source=%s, reward=$%.2f)",
		job.UserID, job.ID, job.Source, job.Reward)

	// Check if already seen (deduplication)
	seen, err := p.isJobSeen(ctx, job)
	if err != nil {
		log.Printf("[JOB-PROC] User %s: Redis error checking seen job %s: %v", job.UserID, job.ID, err)
		// Continue processing on Redis error to avoid missing jobs
	} else if seen {
		log.Printf("[JOB-PROC] User %s: Job %s already seen, skipping", job.UserID, job.ID)
		result.Duplicate = true
		return result, nil
	}

	// Load config for reward filter
	config, err := p.loadConfig(job.UserID)
	if err != nil {
		log.Printf("[JOB-PROC] User %s: Failed to load config for job %s: %v", job.UserID, job.ID, err)
		return result, fmt.Errorf("failed to load config: %w", err)
	}

	// Check reward filter
	if !p.matchesRewardFilter(job, config) {
		log.Printf("[JOB-PROC] User %s: Job %s filtered by reward ($%.2f not in $%.2f-$%.2f)",
			job.UserID, job.ID, job.Reward, config.MinReward, config.MaxReward)
		result.Filtered = true
		return result, nil
	}

	// Record job as seen
	if err := p.recordJob(ctx, job); err != nil {
		log.Printf("[JOB-PROC] User %s: Failed to record job %s as seen: %v", job.UserID, job.ID, err)
		return result, fmt.Errorf("failed to record job: %w", err)
	}

	// Update statistics
	if err := p.incrementJobCount(job.UserID); err != nil {
		log.Printf("[JOB-PROC] User %s: Failed to increment job count for %s: %v", job.UserID, job.ID, err)
		// Non-fatal: continue processing
	}

	if err := p.stateManager.UpdateRuntime(job.UserID, map[string]interface{}{
		"current_job_id":      job.ID,
		"current_action_step": EventTypeJobDetected,
		"action_status":       ActionStatusIdle,
		"last_activity":       time.Now().UTC(),
	}); err != nil {
		log.Printf("[JOB-PROC] User %s: Failed to update runtime snapshot for job %s: %v", job.UserID, job.ID, err)
	}

	eventData := map[string]interface{}{
		"id":         job.ID,
		"title":      job.Title,
		"reward":     job.Reward,
		"url":        job.URL,
		"source":     job.Source,
		"currency":   job.Currency,
		"timestamp":  job.Timestamp,
		"lang_pair":  job.LangPair,
		"word_count": job.WordCount,
	}
	if _, err := p.stateManager.AppendEvent(job.UserID, WatcherEventInput{
		Level:   "info",
		Source:  normalizeEventSource(job.Source),
		Type:    EventTypeJobDetected,
		JobID:   job.ID,
		Message: fmt.Sprintf("Job detected: %s", job.Title),
		Data:    eventData,
	}); err != nil {
		log.Printf("[JOB-PROC] User %s: Failed to persist job event %s: %v", job.UserID, job.ID, err)
	}

	// Publish to user's Redis channel
	if err := p.publishJob(ctx, job); err != nil {
		log.Printf("[JOB-PROC] User %s: Failed to publish job %s: %v", job.UserID, job.ID, err)
		return result, fmt.Errorf("failed to publish job: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("[JOB-PROC] User %s: Job %s processed successfully in %v", job.UserID, job.ID, duration)
	result.Matched = true
	return result, nil
}

// isJobSeen checks if a job has already been seen using Redis SISMEMBER
// Returns (seen, error) - error indicates Redis connection issue
func (p *JobProcessor) isJobSeen(ctx context.Context, job Job) (bool, error) {
	key := GetSeenJobsKey(job.UserID.String())
	result := p.redis.SIsMember(ctx, key, job.ID)
	if result.Err() != nil {
		return false, result.Err()
	}
	return result.Val(), nil
}

// loadConfig retrieves the watcher config for a user
func (p *JobProcessor) loadConfig(userID uuid.UUID) (*models.WatcherConfig, error) {
	var config models.WatcherConfig
	err := p.db.Where("user_id = ?", userID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// matchesRewardFilter checks if the job reward is within the configured range
func (p *JobProcessor) matchesRewardFilter(job Job, config *models.WatcherConfig) bool {
	if job.Reward < config.MinReward {
		return false
	}
	if job.Reward > config.MaxReward {
		return false
	}
	return true
}

// recordJob adds the job ID to the seen jobs set in Redis
// P0-1 FIX: Set TTL of 24 hours to prevent unbounded growth
func (p *JobProcessor) recordJob(ctx context.Context, job Job) error {
	key := GetSeenJobsKey(job.UserID.String())
	err := p.redis.SAdd(ctx, key, job.ID).Err()
	if err != nil {
		return err
	}
	// Set TTL to prevent unbounded growth (24 hours)
	_ = p.redis.Expire(ctx, key, 24*time.Hour).Err()
	return nil
}

// incrementJobCount increments the job counter for a user
func (p *JobProcessor) incrementJobCount(userID uuid.UUID) error {
	result := p.db.Model(&models.WatcherState{}).
		Where("user_id = ?", userID).
		UpdateColumn("total_jobs_found", gorm.Expr("total_jobs_found + 1")).
		Update("last_activity", gorm.Expr("NOW()"))
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// publishJob publishes a job to the user's Redis channel
func (p *JobProcessor) publishJob(ctx context.Context, job Job) error {
	channel := GetJobsChannel(job.UserID.String())
	payload := map[string]interface{}{
		"type":      "job",
		"data":      job,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	jobData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	result := p.redis.Publish(ctx, channel, jobData)
	if result.Err() != nil {
		return fmt.Errorf("redis publish: %w", result.Err())
	}
	log.Printf("[JOB-PROC] User %s: Published job %s to channel (subscribers: %d)",
		job.UserID, job.ID, result.Val()) // Val() returns number of subscribers
	return nil
}

// PublishEvent publishes an event to the user's Redis channel
func (p *JobProcessor) PublishEvent(ctx context.Context, userID uuid.UUID, event string, data map[string]interface{}) error {
	channel := GetEventsChannel(userID.String())
	payload := map[string]interface{}{
		"type":      "event",
		"event":     event,
		"data":      data,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	message, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	err = p.redis.Publish(ctx, channel, message).Err()
	if err != nil {
		log.Printf("[JOB-PROC] User %s: Failed to publish event '%s': %v", userID, event, err)
		return err
	}
	log.Printf("[JOB-PROC] User %s: Published event '%s' to %s", userID, event, channel)
	return nil
}

// PublishError publishes an error to the user's Redis channel
func (p *JobProcessor) PublishError(ctx context.Context, userID uuid.UUID, errMsg string) error {
	channel := GetErrorsChannel(userID.String())
	payload := map[string]interface{}{
		"type":      "error",
		"message":   errMsg,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	message, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}
	err = p.redis.Publish(ctx, channel, message).Err()
	if err != nil {
		log.Printf("[JOB-PROC] User %s: Failed to publish error: %v", userID, err)
		return err
	}
	log.Printf("[JOB-PROC] User %s: Published error to %s: %s", userID, channel, errMsg)
	return nil
}

func normalizeEventSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "rss":
		return "rss"
	case "websocket", "gengo_ws", "gengo-websocket":
		return "gengo_ws"
	case "browser":
		return "browser"
	case "action":
		return "action"
	case "external":
		return "external"
	default:
		return "system"
	}
}
