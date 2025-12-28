package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tdawe1/translation-app/internal/models"
)

// Job represents a detected job from RSS or WebSocket
type Job struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Reward float64 `json:"reward"`
	URL    string  `json:"url"`
	Source string  `json:"source"` // "rss" or "websocket"
	UserID uuid.UUID `json:"user_id"`
}

// WatcherInstance represents an active watcher for a user
type WatcherInstance struct {
	UserID    uuid.UUID
	Config    *models.WatcherConfig
	State     *models.WatcherState
	RSS       *RSSMonitor
	WebSocket *WebSocketMonitor
	Context   context.Context
	Cancel    context.CancelFunc
	Running   bool
	mu        sync.RWMutex
}

// UserWatcherManager manages per-user watcher instances
type UserWatcherManager struct {
	db       *gorm.DB
	redis    *redis.Client
	watchers map[uuid.UUID]*WatcherInstance
	mu       sync.RWMutex
	jobChan  chan Job
}

// NewUserWatcherManager creates a new watcher manager
func NewUserWatcherManager(db *gorm.DB, redisClient *redis.Client) *UserWatcherManager {
	return &UserWatcherManager{
		db:       db,
		redis:    redisClient,
		watchers: make(map[uuid.UUID]*WatcherInstance),
		jobChan:  make(chan Job, 1000),
	}
}

// StartWatcher starts a watcher for a user
func (m *UserWatcherManager) StartWatcher(userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running
	if instance, exists := m.watchers[userID]; exists && instance.Running {
		return fmt.Errorf("watcher already running for user %s", userID)
	}

	// Load user config and state
	var config models.WatcherConfig
	if err := m.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		return fmt.Errorf("config not found for user %s: %w", userID, err)
	}

	var state models.WatcherState
	if err := m.db.Where("user_id = ?", userID).First(&state).Error; err != nil {
		return fmt.Errorf("state not found for user %s: %w", userID, err)
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create monitors
	rss := NewRSSMonitor(config.RSSFeedURL, userID, config.MinReward)
	ws := NewWebSocketMonitor("", userID) // Session token from config

	instance := &WatcherInstance{
		UserID:    userID,
		Config:    &config,
		State:     &state,
		RSS:       rss,
		WebSocket: ws,
		Context:   ctx,
		Cancel:    cancel,
		Running:   true,
	}

	m.watchers[userID] = instance

	// Start monitoring in background
	go m.runWatcher(instance)

	// Update state
	m.updateState(userID, "running")

	// Notify user
	m.publishEvent(userID, "watcher_started")

	return nil
}

// StopWatcher stops a watcher for a user
func (m *UserWatcherManager) StopWatcher(userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.watchers[userID]
	if !exists {
		return fmt.Errorf("no watcher running for user %s", userID)
	}

	instance.Cancel()
	instance.Running = false

	// Update state
	m.updateState(userID, "stopped")

	// Notify user
	m.publishEvent(userID, "watcher_stopped")

	return nil
}

// GetStatus returns the status of a user's watcher
func (m *UserWatcherManager) GetStatus(userID uuid.UUID) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, exists := m.watchers[userID]
	if !exists {
		// Check database for state
		var state models.WatcherState
		if err := m.db.Where("user_id = ?", userID).First(&state).Error; err != nil {
			return "unknown", nil
		}
		return state.WatcherStatus, nil
	}

	if instance.Running {
		return "running", nil
	}

	return "stopped", nil
}

// runWatcher runs the monitoring loop for a watcher instance
func (m *UserWatcherManager) runWatcher(instance *WatcherInstance) {
	// Create job channel for this instance
	jobChan := make(chan Job, 100)

	// Start RSS monitor
	go func() {
		if err := instance.RSS.Start(instance.Context, jobChan); err != nil {
			m.publishError(instance.UserID, err.Error())
		}
	}()

	// Start WebSocket monitor
	go func() {
		if err := instance.WebSocket.Start(instance.Context, jobChan); err != nil {
			m.publishError(instance.UserID, err.Error())
		}
	}()

	// Process jobs from channel
	for {
		select {
		case <-instance.Context.Done():
			return
		case job := <-jobChan:
			m.handleJob(job)
		}
	}
}

// handleJob processes a new job
func (m *UserWatcherManager) handleJob(job Job) {
	// Check if already seen (deduplication)
	if m.isJobSeen(job.UserID, job.ID) {
		return
	}

	// Check reward filter
	var config models.WatcherConfig
	if err := m.db.Where("user_id = ?", job.UserID).First(&config).Error; err != nil {
		return
	}

	if job.Reward < config.MinReward || job.Reward > config.MaxReward {
		return
	}

	// Record job
	m.recordJob(job)

	// Update statistics
	m.incrementJobCount(job.UserID)

	// Publish to user's Redis channel
	m.publishJob(job)
}

// isJobSeen checks if a job has already been seen
func (m *UserWatcherManager) isJobSeen(userID uuid.UUID, jobID string) bool {
	key := fmt.Sprintf("user:%s:seen_jobs", userID)

	// Use Redis SISMEMBER to check if job ID exists
	ctx := context.Background()
	result := m.redis.SIsMember(ctx, key, jobID)
	if result.Err() != nil {
		return false
	}

	return result.Val()
}

// recordJob records a job as seen
func (m *UserWatcherManager) recordJob(job Job) {
	ctx := context.Background()
	key := fmt.Sprintf("user:%s:seen_jobs", job.UserID)
	m.redis.SAdd(ctx, key, job.ID)

	// Also update in database
	var state models.WatcherState
	m.db.Where("user_id = ?", job.UserID).First(&state)

	// Parse existing job IDs
	// For simplicity, just append to the list
	// In production, you'd want to limit this list size
}

// incrementJobCount increments the job count for a user
func (m *UserWatcherManager) incrementJobCount(userID uuid.UUID) {
	m.db.Model(&models.WatcherState{}).
		Where("user_id = ?", userID).
		UpdateColumn("total_jobs_found", gorm.Expr("total_jobs_found + 1")).
		Update("last_activity", time.Now())
}

// publishJob publishes a job to the user's Redis channel
func (m *UserWatcherManager) publishJob(job Job) {
	ctx := context.Background()
	channel := fmt.Sprintf("user:%s:jobs", job.UserID)
	jobData, _ := json.Marshal(job)
	m.redis.Publish(ctx, channel, jobData)
}

// publishEvent publishes an event to the user's Redis channel
func (m *UserWatcherManager) publishEvent(userID uuid.UUID, event string) {
	ctx := context.Background()
	channel := fmt.Sprintf("user:%s:events", userID)
	m.redis.Publish(ctx, channel, fmt.Sprintf(`{"type":"%s"}`, event))
}

// publishError publishes an error to the user's Redis channel
func (m *UserWatcherManager) publishError(userID uuid.UUID, errMsg string) {
	ctx := context.Background()
	channel := fmt.Sprintf("user:%s:errors", userID)
	m.redis.Publish(ctx, channel, fmt.Sprintf(`{"error":"%s"}`, errMsg))
}

// updateState updates the watcher state in the database
func (m *UserWatcherManager) updateState(userID uuid.UUID, status string) {
	m.db.Model(&models.WatcherState{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"watcher_status": status,
			"last_activity":  time.Now(),
		})
}

// GetActiveWatchers returns the count of active watchers
func (m *UserWatcherManager) GetActiveWatchers() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, instance := range m.watchers {
		if instance.Running {
			count++
		}
	}
	return count
}

// StopAll stops all watchers (for shutdown)
func (m *UserWatcherManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, instance := range m.watchers {
		if instance.Running {
			instance.Cancel()
			instance.Running = false
			m.updateState(userID, "stopped")
		}
	}
}
