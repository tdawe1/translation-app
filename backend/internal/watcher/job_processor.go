package watcher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/tdawe1/translation-app/internal/models"
)

// JobProcessor handles job filtering, deduplication, and publishing
type JobProcessor struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(db *gorm.DB, redis *redis.Client) *JobProcessor {
	return &JobProcessor{
		db:    db,
		redis: redis,
	}
}

// ProcessJob handles a new job: checks filters, deduplicates, records, and publishes
func (p *JobProcessor) ProcessJob(ctx context.Context, job Job) error {
	// Check if already seen (deduplication)
	if p.isJobSeen(ctx, job) {
		return nil
	}

	// Load config for reward filter
	config, err := p.loadConfig(job.UserID)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check reward filter
	if !p.matchesRewardFilter(job, config) {
		return nil
	}

	// Record job as seen
	if err := p.recordJob(ctx, job); err != nil {
		return fmt.Errorf("failed to record job: %w", err)
	}

	// Update statistics
	p.incrementJobCount(job.UserID)

	// Publish to user's Redis channel
	if err := p.publishJob(ctx, job); err != nil {
		return fmt.Errorf("failed to publish job: %w", err)
	}

	return nil
}

// isJobSeen checks if a job has already been seen using Redis SISMEMBER
func (p *JobProcessor) isJobSeen(ctx context.Context, job Job) bool {
	key := GetSeenJobsKey(job.UserID.String())
	result := p.redis.SIsMember(ctx, key, job.ID)
	if result.Err() != nil {
		return false
	}
	return result.Val()
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
func (p *JobProcessor) recordJob(ctx context.Context, job Job) error {
	key := GetSeenJobsKey(job.UserID.String())
	return p.redis.SAdd(ctx, key, job.ID).Err()
}

// incrementJobCount increments the job counter for a user
func (p *JobProcessor) incrementJobCount(userID uuid.UUID) {
	p.db.Model(&models.WatcherState{}).
		Where("user_id = ?", userID).
		UpdateColumn("total_jobs_found", gorm.Expr("total_jobs_found + 1")).
		Update("last_activity", gorm.Expr("NOW()"))
}

// publishJob publishes a job to the user's Redis channel
func (p *JobProcessor) publishJob(ctx context.Context, job Job) error {
	channel := GetJobsChannel(job.UserID.String())
	jobData, err := json.Marshal(job)
	if err != nil {
		return err
	}
	return p.redis.Publish(ctx, channel, jobData).Err()
}

// PublishEvent publishes an event to the user's Redis channel
func (p *JobProcessor) PublishEvent(ctx context.Context, userID uuid.UUID, event string) error {
	channel := GetEventsChannel(userID.String())
	message := fmt.Sprintf(`{"type":"%s"}`, event)
	return p.redis.Publish(ctx, channel, message).Err()
}

// PublishError publishes an error to the user's Redis channel
func (p *JobProcessor) PublishError(ctx context.Context, userID uuid.UUID, errMsg string) error {
	channel := GetErrorsChannel(userID.String())
	message := fmt.Sprintf(`{"error":"%s"}`, errMsg)
	return p.redis.Publish(ctx, channel, message).Err()
}
