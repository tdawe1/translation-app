package watcher

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

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
		db:                   db,
		redis:                redisClient,
		watchers:             make(map[uuid.UUID]*WatcherInstance),
		stateManager:         NewStateManager(db),
		jobProcessor:         NewJobProcessor(db, redisClient),
		browserFactory:       func(uuid.UUID) BrowserController { return noopBrowserController{} },
		actionCoordinatorCfg: ActionCoordinatorConfig{QueueSize: defaultActionQueueSize},
	}
}

// minimalTestDB is a minimal database.Database implementation for testing
type minimalTestDB struct{}

func (m *minimalTestDB) Create(value interface{}) *gorm.DB                       { return nil }
func (m *minimalTestDB) First(dest interface{}, conds ...interface{}) *gorm.DB   { return nil }
func (m *minimalTestDB) Where(query interface{}, args ...interface{}) *gorm.DB   { return nil }
func (m *minimalTestDB) Model(value interface{}) *gorm.DB                        { return nil }
func (m *minimalTestDB) Begin(opts ...*sql.TxOptions) *gorm.DB                   { return nil }
func (m *minimalTestDB) Exec(sql string, values ...interface{}) *gorm.DB         { return nil }
func (m *minimalTestDB) Save(value interface{}) *gorm.DB                         { return nil }
func (m *minimalTestDB) Updates(values interface{}) *gorm.DB                     { return nil }
func (m *minimalTestDB) UpdateColumn(column string, value interface{}) *gorm.DB  { return nil }
func (m *minimalTestDB) Update(column string, value interface{}) *gorm.DB        { return nil }
func (m *minimalTestDB) Delete(value interface{}, conds ...interface{}) *gorm.DB { return nil }
func (m *minimalTestDB) Offset(offset int) *gorm.DB                              { return nil }
func (m *minimalTestDB) Limit(limit int) *gorm.DB                                { return nil }
func (m *minimalTestDB) Order(value interface{}) *gorm.DB                        { return nil }
func (m *minimalTestDB) Count(count *int64) *gorm.DB                             { return nil }

type noopBrowserController struct{}

func (noopBrowserController) Start(context.Context) error { return nil }
func (noopBrowserController) Stop(context.Context) error  { return nil }
func (noopBrowserController) Restart(context.Context) error {
	return nil
}
func (noopBrowserController) Health(context.Context) error {
	return nil
}
func (noopBrowserController) CaptureScreenshot(context.Context) (*BrowserActionResult, error) {
	return &BrowserActionResult{
		Outcome: BrowserOutcomeOpened,
		Message: "noop screenshot",
	}, nil
}
func (noopBrowserController) OpenJob(context.Context, Job) (*BrowserActionResult, error) {
	return &BrowserActionResult{Outcome: BrowserOutcomeOpened, Message: "noop browser"}, nil
}
func (noopBrowserController) AcceptJob(context.Context, Job, string) (*BrowserActionResult, error) {
	return &BrowserActionResult{Outcome: BrowserOutcomeAccepted, Message: "noop accept"}, nil
}

// Job represents a detected job from RSS or WebSocket
type Job struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Reward    float64   `json:"reward"`
	URL       string    `json:"url"`
	Source    string    `json:"source"` // "rss", "websocket", or external bridge source
	Currency  string    `json:"currency,omitempty"`
	Timestamp float64   `json:"timestamp,omitempty"`
	LangPair  string    `json:"lang_pair,omitempty"`
	WordCount int       `json:"word_count,omitempty"`
	UserID    uuid.UUID `json:"user_id"`
}

// WatcherInstance represents an active watcher for a user
type WatcherInstance struct {
	UserID            uuid.UUID
	Config            *models.WatcherConfig
	State             *models.WatcherState
	RSS               *RSSMonitor
	WebSocket         *WebSocketMonitor
	Browser           BrowserController
	ActionCoordinator *ActionCoordinator
	Context           context.Context
	Cancel            context.CancelFunc
	Running           bool
	mu                sync.RWMutex
}

type BrowserFactory func(userID uuid.UUID) BrowserController

type ManagerOption func(*UserWatcherManager)

// UserWatcherManager manages per-user watcher instances
type UserWatcherManager struct {
	db                   database.Database
	redis                *redis.Client
	watchers             map[uuid.UUID]*WatcherInstance
	mu                   sync.RWMutex
	stateManager         *StateManager
	jobProcessor         *JobProcessor
	browserFactory       BrowserFactory
	actionCoordinatorCfg ActionCoordinatorConfig
}

func WithBrowserFactory(factory BrowserFactory) ManagerOption {
	return func(m *UserWatcherManager) {
		m.browserFactory = factory
	}
}

func WithActionCoordinatorConfig(config ActionCoordinatorConfig) ManagerOption {
	return func(m *UserWatcherManager) {
		m.actionCoordinatorCfg = config
	}
}

// NewUserWatcherManager creates a new watcher manager
func NewUserWatcherManager(db database.Database, redisClient *redis.Client, opts ...ManagerOption) *UserWatcherManager {
	manager := &UserWatcherManager{
		db:                   db,
		redis:                redisClient,
		watchers:             make(map[uuid.UUID]*WatcherInstance),
		stateManager:         NewStateManager(db),
		jobProcessor:         NewJobProcessor(db, redisClient),
		actionCoordinatorCfg: ActionCoordinatorConfigFromEnv(),
	}
	manager.browserFactory = func(userID uuid.UUID) BrowserController {
		return NewConfiguredBrowserWorker(userID, BrowserWorkerConfigFromEnv())
	}
	for _, opt := range opts {
		opt(manager)
	}
	return manager
}

// ProcessExternalJob runs an externally discovered watcher job through the normal pipeline.
func (m *UserWatcherManager) ProcessExternalJob(
	ctx context.Context,
	userID uuid.UUID,
	job Job,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	job.UserID = userID
	if job.Source == "" {
		job.Source = "external"
	}
	result, err := m.jobProcessor.ProcessJobWithResult(ctx, job)
	if err != nil {
		return err
	}
	if !result.Matched {
		return nil
	}

	m.mu.RLock()
	instance := m.watchers[userID]
	m.mu.RUnlock()
	if instance == nil || !instance.Running || instance.ActionCoordinator == nil {
		return nil
	}
	return instance.ActionCoordinator.Submit(job, shouldAutoAccept(job, instance.Config))
}

// UpdateBrowserState stores the dashboard browser snapshot and streams it to
// connected dashboard clients as a watcher health patch.
func (m *UserWatcherManager) UpdateBrowserState(
	ctx context.Context,
	userID uuid.UUID,
	updates map[string]interface{},
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(updates) == 0 {
		return nil
	}
	if err := m.stateManager.UpdateRuntime(userID, updates); err != nil {
		return err
	}
	if err := m.jobProcessor.PublishEvent(ctx, userID, EventTypeWatcherHealth, updates); err != nil {
		log.Printf("[WATCHER] Failed to publish browser state update for user %s: %v", userID, err)
	}
	return nil
}

func (m *UserWatcherManager) RestartBrowser(ctx context.Context, userID uuid.UUID) error {
	instance, err := m.runningInstance(userID)
	if err != nil {
		return err
	}
	if instance.ActionCoordinator == nil {
		return fmt.Errorf("action coordinator unavailable for user %s", userID)
	}
	return instance.ActionCoordinator.RestartBrowser(ctx)
}

func (m *UserWatcherManager) CaptureBrowserScreenshot(
	ctx context.Context,
	userID uuid.UUID,
) (*BrowserActionResult, error) {
	instance, err := m.runningInstance(userID)
	if err != nil {
		return nil, err
	}
	if instance.ActionCoordinator == nil {
		return nil, fmt.Errorf("action coordinator unavailable for user %s", userID)
	}
	return instance.ActionCoordinator.CaptureScreenshot(ctx)
}

func (m *UserWatcherManager) runningInstance(userID uuid.UUID) (*WatcherInstance, error) {
	m.mu.RLock()
	instance := m.watchers[userID]
	m.mu.RUnlock()
	if instance == nil || !instance.Running {
		return nil, fmt.Errorf("watcher is not running for user %s", userID)
	}
	return instance, nil
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

	log.Printf("[WATCHER] Starting watcher for user %s (email: %s)", userID, user.Email)
	log.Printf("[WATCHER] Config: RSS=%s, MinReward=$%.2f, MaxReward=$%.2f, AutoAccept=%v",
		config.RSSFeedURL, config.MinReward, config.MaxReward, config.AutoAcceptEnabled)

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create monitors
	rss := NewRSSMonitor(config.RSSFeedURL, userID, config.MinReward)
	ws := NewWebSocketMonitor(userID, config.GengoSessionToken, config.GengoUserKey, config.GengoUserID, user.IsAdmin())
	var browser BrowserController
	if m.browserFactory != nil {
		browser = m.browserFactory(userID)
	}
	actionReporter := &dbActionReporter{
		userID:       userID,
		stateManager: m.stateManager,
		jobProcessor: m.jobProcessor,
		db:           m.db,
	}
	actionCoordinator := NewActionCoordinator(userID, browser, actionReporter, m.actionCoordinatorCfg)
	runtimeUpdater := func(updates map[string]interface{}) error {
		if err := m.stateManager.UpdateRuntime(userID, updates); err != nil {
			return err
		}
		for _, event := range runtimeEventsFromUpdates(updates) {
			if _, err := m.stateManager.AppendEvent(userID, event); err != nil {
				log.Printf("[WATCHER] Failed to append runtime event for user %s: %v", userID, err)
				continue
			}
			if err := m.jobProcessor.PublishEvent(context.Background(), userID, event.Type, map[string]interface{}{
				"message": event.Message,
				"source":  event.Source,
				"level":   event.Level,
				"data":    event.Data,
			}); err != nil {
				log.Printf("[WATCHER] Failed to publish runtime event for user %s: %v", userID, err)
			}
		}
		if isHealthUpdate(updates) {
			if err := m.jobProcessor.PublishEvent(context.Background(), userID, EventTypeWatcherHealth, updates); err != nil {
				log.Printf("[WATCHER] Failed to publish health update for user %s: %v", userID, err)
			}
		}
		return nil
	}
	rss.RuntimeUpdate = runtimeUpdater
	ws.RuntimeUpdate = runtimeUpdater

	instance := &WatcherInstance{
		UserID:            userID,
		Config:            config,
		State:             state,
		RSS:               rss,
		WebSocket:         ws,
		Browser:           browser,
		ActionCoordinator: actionCoordinator,
		Context:           ctx,
		Cancel:            cancel,
		Running:           true,
	}

	m.watchers[userID] = instance

	log.Printf("[WATCHER] Watcher instance created for user %s", userID)

	// Start monitoring in background
	go m.runWatcher(instance)

	profileStatus := ProfileStatusUnseeded
	overallStatus := OverallStatusDegraded
	alertStatus := AlertStatusWarning
	browserStatus := BrowserStatusUnconfigured
	browserEventType := EventTypeBrowserUnconfigured
	runtimeMode := "Gengo credentials are required for WebSocket monitoring"
	actionStep := "Monitoring RSS; WebSocket requires Gengo credentials"
	lastError := "missing Gengo session token or user ID"
	if config.GengoSessionToken != "" && config.GengoUserID != "" {
		profileStatus = ProfileStatusSeeded
		overallStatus = OverallStatusRunning
		alertStatus = AlertStatusNone
		browserStatus = BrowserStatusStarting
		browserEventType = EventTypeBrowserStarted
		runtimeMode = "Server-owned worker browser starting"
		actionStep = "Monitoring RSS and realtime WebSocket"
		lastError = ""
	}

	if err := m.stateManager.UpdateRuntime(userID, map[string]interface{}{
		"watcher_status":      StatusRunning,
		"overall_status":      overallStatus,
		"feed_status":         FeedStatusMonitoring,
		"browser_status":      browserStatus,
		"action_status":       ActionStatusIdle,
		"alert_status":        alertStatus,
		"profile_status":      profileStatus,
		"logged_in_state":     "unknown",
		"current_action_step": actionStep,
		"current_job_id":      "",
		"last_error":          lastError,
		"last_activity":       time.Now().UTC(),
	}); err != nil {
		log.Printf("[WATCHER] Failed to update state for user %s: %v", userID, err)
	}

	if _, err := m.stateManager.AppendEvent(userID, WatcherEventInput{
		Level:   "info",
		Source:  "system",
		Type:    EventTypeWorkerStarted,
		Message: "Watcher worker started",
		Data: map[string]interface{}{
			"watcher_status": StatusRunning,
			"overall_status": overallStatus,
		},
	}); err != nil {
		log.Printf("[WATCHER] Failed to persist start event for user %s: %v", userID, err)
	}
	if _, err := m.stateManager.AppendEvent(userID, WatcherEventInput{
		Level:   "info",
		Source:  "browser",
		Type:    browserEventType,
		Message: runtimeMode,
		Data: map[string]interface{}{
			"profile_status": profileStatus,
		},
	}); err != nil {
		log.Printf("[WATCHER] Failed to persist browser event for user %s: %v", userID, err)
	}

	// Notify user
	ctx = context.Background()
	if err := m.jobProcessor.PublishEvent(ctx, userID, EventTypeWorkerStarted, map[string]interface{}{
		"watcher_status": StatusRunning,
		"overall_status": overallStatus,
	}); err != nil {
		log.Printf("[WATCHER] Failed to publish start event for user %s: %v", userID, err)
	}
	if err := m.jobProcessor.PublishEvent(ctx, userID, browserEventType, map[string]interface{}{
		"profile_status": profileStatus,
		"message":        runtimeMode,
	}); err != nil {
		log.Printf("[WATCHER] Failed to publish browser event for user %s: %v", userID, err)
	}
	actionCoordinator.Start(instance.Context)

	log.Printf("[WATCHER] Watcher started successfully for user %s at %s", userID, time.Now().Format(time.RFC3339))
	return nil
}

// RestoreRunningWatchers starts watcher workers that were marked running before
// the backend process restarted.
func (m *UserWatcherManager) RestoreRunningWatchers(ctx context.Context) error {
	var states []models.WatcherState
	if err := m.db.Where("watcher_status = ?", StatusRunning).Find(&states).Error; err != nil {
		return fmt.Errorf("load running watcher states: %w", err)
	}

	for _, state := range states {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		m.appendAndPublishManagerEvent(context.Background(), state.UserID, EventTypeWorkerRestoreStart, WatcherEventInput{
			Level:   "info",
			Source:  "system",
			Type:    EventTypeWorkerRestoreStart,
			Message: "Restoring watcher after backend restart",
		})

		if err := m.StartWatcher(state.UserID); err != nil {
			log.Printf("[WATCHER] Failed to restore watcher for user %s: %v", state.UserID, err)
			m.appendAndPublishManagerEvent(context.Background(), state.UserID, EventTypeWorkerRestoreFailed, WatcherEventInput{
				Level:   "warning",
				Source:  "system",
				Type:    EventTypeWorkerRestoreFailed,
				Message: "Watcher restore failed",
				Data: map[string]interface{}{
					"error": err.Error(),
				},
			})
			continue
		}
		m.appendAndPublishManagerEvent(context.Background(), state.UserID, EventTypeWorkerRestoreOK, WatcherEventInput{
			Level:   "info",
			Source:  "system",
			Type:    EventTypeWorkerRestoreOK,
			Message: "Watcher restored after backend restart",
		})
		log.Printf("[WATCHER] Restored watcher for user %s", state.UserID)
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

	log.Printf("[WATCHER] Stopping watcher for user %s", userID)

	instance.Cancel()
	instance.Running = false
	if instance.ActionCoordinator != nil {
		instance.ActionCoordinator.Stop(context.Background())
	}

	// Update state
	if err := m.stateManager.UpdateRuntime(userID, map[string]interface{}{
		"watcher_status":        StatusStopped,
		"overall_status":        OverallStatusStopped,
		"feed_status":           FeedStatusStopped,
		"browser_status":        BrowserStatusStopped,
		"action_status":         ActionStatusIdle,
		"alert_status":          AlertStatusNone,
		"browser_process_alive": false,
		"dev_tools_connected":   false,
		"current_action_step":   "",
		"current_job_id":        "",
		"last_error":            "",
		"last_activity":         time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("failed to update state: %w", err)
	}

	if _, err := m.stateManager.AppendEvent(userID, WatcherEventInput{
		Level:   "info",
		Source:  "system",
		Type:    EventTypeWorkerStopped,
		Message: "Watcher worker stopped",
	}); err != nil {
		log.Printf("[WATCHER] Failed to persist stop event for user %s: %v", userID, err)
	}

	// Notify user
	ctx := context.Background()
	if err := m.jobProcessor.PublishEvent(ctx, userID, EventTypeWorkerStopped, map[string]interface{}{
		"watcher_status": StatusStopped,
		"overall_status": OverallStatusStopped,
	}); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	log.Printf("[WATCHER] Watcher stopped for user %s at %s", userID, time.Now().Format(time.RFC3339))
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
	log.Printf("[WATCHER] Starting monitoring loop for user %s", instance.UserID)

	// Create job channel for this instance
	jobChan := make(chan Job, 100)

	// Start RSS monitor with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[WATCHER] RSS monitor PANIC for user %s: %v", instance.UserID, r)
				m.jobProcessor.PublishError(instance.Context, instance.UserID, fmt.Sprintf("RSS monitor panic: %v", r))
			}
		}()
		log.Printf("[WATCHER] RSS monitor started for user %s (feed: %s)", instance.UserID, instance.RSS.GetFeedURL())
		if err := instance.RSS.Start(instance.Context, jobChan); err != nil {
			log.Printf("[WATCHER] RSS monitor error for user %s: %v", instance.UserID, err)
			m.jobProcessor.PublishError(instance.Context, instance.UserID, err.Error())
		}
		log.Printf("[WATCHER] RSS monitor stopped for user %s", instance.UserID)
	}()

	// Start WebSocket monitor with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[WATCHER] WebSocket monitor PANIC for user %s: %v", instance.UserID, r)
				m.jobProcessor.PublishError(instance.Context, instance.UserID, fmt.Sprintf("WebSocket monitor panic: %v", r))
			}
		}()
		wsEnabled := instance.Config.GengoSessionToken != "" && instance.Config.GengoUserID != ""
		log.Printf("[WATCHER] WebSocket monitor starting for user %s (enabled: %v)", instance.UserID, wsEnabled)
		instance.WebSocket.Start(instance.Context, jobChan)
		log.Printf("[WATCHER] WebSocket monitor stopped for user %s", instance.UserID)
	}()

	// Process jobs from channel with panic recovery
	jobCount := 0
	for {
		select {
		case <-instance.Context.Done():
			log.Printf("[WATCHER] Monitoring loop ended for user %s (processed %d jobs)", instance.UserID, jobCount)
			return
		case job := <-jobChan:
			jobCount++
			log.Printf("[WATCHER] Processing job #%d for user %s: %s (%s, $%.2f)",
				jobCount, instance.UserID, job.ID, job.Source, job.Reward)
			result, err := m.jobProcessor.ProcessJobWithResult(instance.Context, job)
			if err != nil {
				log.Printf("[WATCHER] Error processing job for user %s: %v", instance.UserID, err)
			} else {
				log.Printf("[WATCHER] Job processed successfully for user %s: %s", instance.UserID, job.ID)
				if result.Matched && instance.ActionCoordinator != nil {
					autoAccept := shouldAutoAccept(job, instance.Config)
					if err := instance.ActionCoordinator.Submit(job, autoAccept); err != nil {
						log.Printf("[WATCHER] Failed to queue browser action for user %s job %s: %v", instance.UserID, job.ID, err)
					}
				}
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
			if instance.ActionCoordinator != nil {
				instance.ActionCoordinator.Stop(context.Background())
			}
			m.stateManager.UpdateRuntime(userID, map[string]interface{}{
				"watcher_status":        StatusStopped,
				"overall_status":        OverallStatusStopped,
				"feed_status":           FeedStatusStopped,
				"browser_status":        BrowserStatusStopped,
				"action_status":         ActionStatusIdle,
				"alert_status":          AlertStatusNone,
				"browser_process_alive": false,
				"dev_tools_connected":   false,
				"last_activity":         time.Now().UTC(),
			})
			m.jobProcessor.PublishEvent(ctx, userID, EventTypeWorkerStopped, map[string]interface{}{
				"watcher_status": StatusStopped,
				"overall_status": OverallStatusStopped,
			})
		}
	}
}

// ShutdownRunningWatchers stops in-process workers for backend shutdown while
// preserving watcher_status=running so RestoreRunningWatchers can bring them
// back on process start.
func (m *UserWatcherManager) ShutdownRunningWatchers() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for userID, instance := range m.watchers {
		if !instance.Running {
			continue
		}
		instance.Cancel()
		instance.Running = false
		if instance.ActionCoordinator != nil {
			instance.ActionCoordinator.Stop(context.Background())
		}
		if err := m.stateManager.UpdateRuntime(userID, map[string]interface{}{
			"watcher_status":        StatusRunning,
			"overall_status":        OverallStatusDegraded,
			"feed_status":           FeedStatusStopped,
			"browser_status":        BrowserStatusStopped,
			"browser_process_alive": false,
			"dev_tools_connected":   false,
			"current_action_step":   "Backend shutting down; watcher will restore on startup",
			"last_activity":         time.Now().UTC(),
		}); err != nil {
			log.Printf("[WATCHER] Failed to persist shutdown state for user %s: %v", userID, err)
		}
		m.appendAndPublishManagerEvent(context.Background(), userID, EventTypeWorkerShutdown, WatcherEventInput{
			Level:   "info",
			Source:  "system",
			Type:    EventTypeWorkerShutdown,
			Message: "Backend shutdown persisted watcher for restore",
		})
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

// GetEvents retrieves recent watcher events for a user.
func (m *UserWatcherManager) GetEvents(userID uuid.UUID, limit int) ([]models.WatcherEvent, error) {
	return m.stateManager.ListEvents(userID, limit)
}

func (m *UserWatcherManager) appendAndPublishManagerEvent(
	ctx context.Context,
	userID uuid.UUID,
	eventType string,
	input WatcherEventInput,
) {
	if ctx == nil {
		ctx = context.Background()
	}
	if input.Type == "" {
		input.Type = eventType
	}
	if input.Source == "" {
		input.Source = "system"
	}
	if input.Level == "" {
		input.Level = "info"
	}
	if _, err := m.stateManager.AppendEvent(userID, input); err != nil {
		log.Printf("[WATCHER] Failed to append manager event %s for user %s: %v", eventType, userID, err)
		return
	}
	data := input.Data
	if data == nil {
		data = map[string]interface{}{}
	}
	if err := m.jobProcessor.PublishEvent(ctx, userID, eventType, map[string]interface{}{
		"message": input.Message,
		"source":  input.Source,
		"level":   input.Level,
		"data":    data,
	}); err != nil {
		log.Printf("[WATCHER] Failed to publish manager event %s for user %s: %v", eventType, userID, err)
	}
}

func isHealthUpdate(updates map[string]interface{}) bool {
	if _, ok := updates["last_ws_pong_at"]; ok {
		return true
	}
	if _, ok := updates["last_ws_connect_at"]; ok {
		return true
	}
	if _, ok := updates["last_ws_message_at"]; ok {
		return true
	}
	if _, ok := updates["last_browser_heartbeat_at"]; ok {
		return true
	}
	if _, ok := updates["last_rss_poll_ok_at"]; ok {
		return true
	}
	return false
}

func runtimeEventsFromUpdates(updates map[string]interface{}) []WatcherEventInput {
	events := make([]WatcherEventInput, 0, 2)
	if value, ok := updates["last_rss_poll_started_at"]; ok {
		events = append(events, WatcherEventInput{
			Level:   "info",
			Source:  "rss",
			Type:    EventTypeRSSPollStarted,
			Message: "RSS poll started",
			Data:    map[string]interface{}{"last_rss_poll_started_at": value},
		})
	}
	if value, ok := updates["last_rss_poll_ok_at"]; ok {
		data := map[string]interface{}{"last_rss_poll_ok_at": value}
		if failures, ok := updates["rss_consecutive_failures"]; ok {
			data["rss_consecutive_failures"] = failures
		}
		events = append(events, WatcherEventInput{
			Level:   "info",
			Source:  "rss",
			Type:    EventTypeRSSPollOK,
			Message: "RSS poll completed",
			Data:    data,
		})
	}
	if value, ok := updates["last_ws_connect_at"]; ok {
		events = append(events, WatcherEventInput{
			Level:   "info",
			Source:  "websocket",
			Type:    EventTypeWebSocketConnected,
			Message: "Gengo WSS connected",
			Data:    map[string]interface{}{"last_ws_connect_at": value},
		})
	}
	if value, ok := updates["last_ws_message_at"]; ok {
		events = append(events, WatcherEventInput{
			Level:   "info",
			Source:  "websocket",
			Type:    EventTypeWebSocketMessage,
			Message: "Gengo WSS message received",
			Data:    map[string]interface{}{"last_ws_message_at": value},
		})
	}
	if value, ok := updates["last_ws_pong_at"]; ok {
		events = append(events, WatcherEventInput{
			Level:   "info",
			Source:  "websocket",
			Type:    EventTypeWebSocketPong,
			Message: "Gengo WSS pong received",
			Data:    map[string]interface{}{"last_ws_pong_at": value},
		})
	}
	if reason, ok := updates["last_ws_close_reason"].(string); ok && reason != "" {
		events = append(events, WatcherEventInput{
			Level:   "warning",
			Source:  "websocket",
			Type:    EventTypeWebSocketClosed,
			Message: "Gengo WSS disconnected: " + reason,
			Data:    map[string]interface{}{"last_ws_close_reason": reason},
		})
	}
	return events
}
