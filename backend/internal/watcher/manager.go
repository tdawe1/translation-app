package watcher

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
)

// NewTestManager creates a watcher manager for testing
// This is a test-only constructor that doesn't start background goroutines
// Pass a test database if you need real DB operations; otherwise uses a no-op stub
func NewTestManager(db database.Database) *UserWatcherManager {
	// Create a mock Redis client for testing (using miniredis would be better)
	// For now, we'll use a real Redis client pointing to test DB
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       15, // Test database
	})

	// Use provided DB or minimal stub
	if db == nil {
		db = &minimalTestDB{}
	}

	return &UserWatcherManager{
		db:           db,
		redis:        redisClient,
		watchers:     make(map[uuid.UUID]*WatcherInstance),
		stateManager: NewStateManager(db),
		jobProcessor: NewJobProcessor(db, redisClient),
	}
}

// minimalTestDB is a minimal database.Database implementation for testing
type minimalTestDB struct{}

func (m *minimalTestDB) Create(value interface{}) *gorm.DB                      { return nil }
func (m *minimalTestDB) First(dest interface{}, conds ...interface{}) *gorm.DB  { return nil }
func (m *minimalTestDB) Where(query interface{}, args ...interface{}) *gorm.DB  { return nil }
func (m *minimalTestDB) Model(value interface{}) *gorm.DB                       { return nil }
func (m *minimalTestDB) Begin(opts ...*sql.TxOptions) *gorm.DB                  { return nil }
func (m *minimalTestDB) Exec(sql string, values ...interface{}) *gorm.DB        { return nil }
func (m *minimalTestDB) Save(value interface{}) *gorm.DB                        { return nil }
func (m *minimalTestDB) Updates(values interface{}) *gorm.DB                    { return nil }
func (m *minimalTestDB) UpdateColumn(column string, value interface{}) *gorm.DB { return nil }
func (m *minimalTestDB) Update(column string, value interface{}) *gorm.DB       { return nil }

// Job represents a detected job from RSS or WebSocket
type Job struct {
	ID     string    `json:"id"`
	Title  string    `json:"title"`
	Reward float64   `json:"reward"`
	URL    string    `json:"url"`
	Source string    `json:"source"` // "rss" or "websocket"
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
	db           database.Database
	redis        *redis.Client
	watchers     map[uuid.UUID]*WatcherInstance
	mu           sync.RWMutex
	stateManager *StateManager
	jobProcessor *JobProcessor
}

// NewUserWatcherManager creates a new watcher manager
func NewUserWatcherManager(db database.Database, redisClient *redis.Client) *UserWatcherManager {
	return &UserWatcherManager{
		db:           db,
		redis:        redisClient,
		watchers:     make(map[uuid.UUID]*WatcherInstance),
		stateManager: NewStateManager(db),
		jobProcessor: NewJobProcessor(db, redisClient),
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
	config, err := m.stateManager.LoadConfig(userID)
	if err != nil {
		return fmt.Errorf("config not found for user %s: %w", userID, err)
	}

	state, err := m.stateManager.LoadState(userID)
	if err != nil {
		return fmt.Errorf("state not found for user %s: %w", userID, err)
	}

	// Check if user is admin for heartbeat interval
	var user models.User
	err = m.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return fmt.Errorf("failed to load user: %w", err)
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create monitors
	rss := NewRSSMonitor(config.RSSFeedURL, userID, config.MinReward)
	ws := NewWebSocketMonitor(userID, config.GengoSessionToken, config.GengoUserID)

	instance := &WatcherInstance{
		UserID:    userID,
		Config:    config,
		State:     state,
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
	if err := m.stateManager.UpdateStatus(userID, StatusRunning); err != nil {
		// Log but don't fail
		fmt.Printf("Failed to update state: %v", err)
	}

	// Notify user
	ctx = context.Background()
	if err := m.jobProcessor.PublishEvent(ctx, userID, EventTypeWatcherStarted); err != nil {
		fmt.Printf("Failed to publish event: %v", err)
	}

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
	if err := m.stateManager.UpdateStatus(userID, StatusStopped); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	// Notify user
	ctx := context.Background()
	if err := m.jobProcessor.PublishEvent(ctx, userID, EventTypeWatcherStopped); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// GetStatus returns the status of a user's watcher
func (m *UserWatcherManager) GetStatus(userID uuid.UUID) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, exists := m.watchers[userID]
	if exists {
		if instance.Running {
			return StatusRunning, nil
		}
		return StatusStopped, nil
	}

	// Check database for state
	return m.stateManager.GetStatus(userID)
}

// runWatcher runs the monitoring loop for a watcher instance
func (m *UserWatcherManager) runWatcher(instance *WatcherInstance) {
	// Create job channel for this instance
	jobChan := make(chan Job, 100)

	// Start RSS monitor
	go func() {
		if err := instance.RSS.Start(instance.Context, jobChan); err != nil {
			m.jobProcessor.PublishError(instance.Context, instance.UserID, err.Error())
		}
	}()

	// Start WebSocket monitor
	go func() {
		instance.WebSocket.Start(instance.Context, jobChan)
	}()

	// Process jobs from channel
	for {
		select {
		case <-instance.Context.Done():
			return
		case job := <-jobChan:
			if err := m.jobProcessor.ProcessJob(instance.Context, job); err != nil {
				// Log error but continue processing
				fmt.Printf("Error processing job: %v\n", err)
			}
		}
	}
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

	ctx := context.Background()
	for userID, instance := range m.watchers {
		if instance.Running {
			instance.Cancel()
			instance.Running = false
			m.stateManager.UpdateStatus(userID, StatusStopped)
			m.jobProcessor.PublishEvent(ctx, userID, EventTypeWatcherStopped)
		}
	}
}

// UpdateConfig updates the watcher config for a user
func (m *UserWatcherManager) UpdateConfig(config *models.WatcherConfig) error {
	return m.stateManager.UpdateConfig(config)
}

// GetConfig retrieves the watcher config for a user
func (m *UserWatcherManager) GetConfig(userID uuid.UUID) (*models.WatcherConfig, error) {
	return m.stateManager.LoadConfig(userID)
}

// GetState retrieves the watcher state for a user
func (m *UserWatcherManager) GetState(userID uuid.UUID) (*models.WatcherState, error) {
	return m.stateManager.LoadState(userID)
}
