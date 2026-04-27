package watcher

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
)

const (
	defaultActionQueueSize         = 32
	defaultBrowserHeartbeatTimeout = 5 * time.Second
)

type ActionCoordinatorConfig struct {
	QueueSize         int
	AcceptSelector    string
	HeartbeatInterval time.Duration
}

func ActionCoordinatorConfigFromEnv() ActionCoordinatorConfig {
	return ActionCoordinatorConfig{
		QueueSize:         defaultActionQueueSize,
		AcceptSelector:    getEnvDefault("WATCHER_BROWSER_ACCEPT_SELECTOR", defaultAcceptSelector),
		HeartbeatInterval: envDuration("WATCHER_BROWSER_HEARTBEAT_INTERVAL", 30*time.Second),
	}
}

type actionRequest struct {
	job        Job
	autoAccept bool
}

// ActionReporter is the persistence/fanout boundary used by the coordinator.
type ActionReporter interface {
	UpdateRuntime(map[string]interface{}) error
	AppendEvent(WatcherEventInput) error
	PublishEvent(context.Context, string, map[string]interface{}) error
	RecordAcceptedJob(Job) error
}

type dbActionReporter struct {
	userID       uuid.UUID
	stateManager *StateManager
	jobProcessor *JobProcessor
	db           database.Database
}

func (r *dbActionReporter) UpdateRuntime(updates map[string]interface{}) error {
	return r.stateManager.UpdateRuntime(r.userID, updates)
}

func (r *dbActionReporter) AppendEvent(input WatcherEventInput) error {
	_, err := r.stateManager.AppendEvent(r.userID, input)
	return err
}

func (r *dbActionReporter) PublishEvent(ctx context.Context, event string, data map[string]interface{}) error {
	return r.jobProcessor.PublishEvent(ctx, r.userID, event, data)
}

func (r *dbActionReporter) RecordAcceptedJob(job Job) error {
	return r.db.Model(&models.WatcherState{}).
		Where("user_id = ?", r.userID).
		Updates(map[string]interface{}{
			"total_jobs_accepted": gorm.Expr("total_jobs_accepted + 1"),
			"total_earnings":      gorm.Expr("total_earnings + ?", job.Reward),
			"last_activity":       time.Now().UTC(),
		}).Error
}

// ActionCoordinator serializes browser-side job actions for one user's worker.
type ActionCoordinator struct {
	userID  uuid.UUID
	browser BrowserController
	report  ActionReporter
	config  ActionCoordinatorConfig

	queue         chan actionRequest
	done          chan struct{}
	once          sync.Once
	blockMu       sync.RWMutex
	blocked       bool
	actionMu      sync.RWMutex
	actionActive  bool
	actionTouched bool
}

func NewActionCoordinator(
	userID uuid.UUID,
	browser BrowserController,
	report ActionReporter,
	config ActionCoordinatorConfig,
) *ActionCoordinator {
	if config.QueueSize <= 0 {
		config.QueueSize = defaultActionQueueSize
	}
	if config.AcceptSelector == "" {
		config.AcceptSelector = defaultAcceptSelector
	}
	if config.HeartbeatInterval < 0 {
		config.HeartbeatInterval = 0
	}
	return &ActionCoordinator{
		userID:  userID,
		browser: browser,
		report:  report,
		config:  config,
		queue:   make(chan actionRequest, config.QueueSize),
		done:    make(chan struct{}),
	}
}

func (c *ActionCoordinator) Start(ctx context.Context) {
	go c.run(ctx)
	go c.startBrowser(ctx)
	go c.monitorBrowserHealth(ctx)
}

func (c *ActionCoordinator) Stop(ctx context.Context) {
	c.once.Do(func() {
		close(c.done)
	})
	if c.browser != nil {
		if err := c.browser.Stop(ctx); err != nil {
			log.Printf("[ACTION] User %s: failed to stop browser worker: %v", c.userID, err)
		}
	}
}

func (c *ActionCoordinator) RestartBrowser(ctx context.Context) error {
	if c.browser == nil {
		return fmt.Errorf("browser worker unavailable")
	}
	c.updateRuntime(map[string]interface{}{
		"browser_status":      BrowserStatusStarting,
		"current_action_step": "Restarting worker browser",
		"last_activity":       time.Now().UTC(),
	})
	c.appendAndPublish(EventTypeBrowserStarted, WatcherEventInput{
		Level:   "info",
		Source:  "browser",
		Type:    EventTypeBrowserStarted,
		Message: "Restarting worker browser",
	})

	if err := c.browser.Restart(ctx); err != nil {
		c.updateRuntime(map[string]interface{}{
			"overall_status":            OverallStatusDegraded,
			"browser_status":            BrowserStatusFailed,
			"alert_status":              AlertStatusWarning,
			"browser_process_alive":     false,
			"dev_tools_connected":       false,
			"last_browser_heartbeat_at": nil,
			"last_error":                err.Error(),
			"current_action_step":       "Browser restart failed; feed monitoring continues",
			"last_activity":             time.Now().UTC(),
		})
		c.appendAndPublish(EventTypeBrowserStartFailed, WatcherEventInput{
			Level:   "warning",
			Source:  "browser",
			Type:    EventTypeBrowserStartFailed,
			Message: "Worker browser restart failed",
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})
		return err
	}

	now := time.Now().UTC()
	c.updateRuntime(map[string]interface{}{
		"overall_status":            OverallStatusRunning,
		"browser_status":            BrowserStatusReady,
		"profile_status":            ProfileStatusVerified,
		"browser_process_alive":     true,
		"dev_tools_connected":       true,
		"last_browser_heartbeat_at": now,
		"last_error":                "",
		"current_action_step":       "Worker browser restarted",
		"last_activity":             now,
	})
	c.appendAndPublish(EventTypeBrowserReady, WatcherEventInput{
		Level:   "info",
		Source:  "browser",
		Type:    EventTypeBrowserReady,
		Message: "Worker browser restarted",
	})
	return nil
}

func (c *ActionCoordinator) CaptureScreenshot(ctx context.Context) (*BrowserActionResult, error) {
	if c.browser == nil {
		return nil, fmt.Errorf("browser worker unavailable")
	}
	result, err := c.browser.CaptureScreenshot(ctx)
	if err != nil {
		c.updateRuntime(map[string]interface{}{
			"browser_status":      BrowserStatusFailed,
			"alert_status":        AlertStatusWarning,
			"last_error":          err.Error(),
			"current_action_step": "Manual screenshot failed",
			"last_activity":       time.Now().UTC(),
		})
		c.appendAndPublish(EventTypeBrowserScreenshot, WatcherEventInput{
			Level:   "warning",
			Source:  "browser",
			Type:    EventTypeBrowserScreenshot,
			Message: "Manual screenshot failed",
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})
		return result, err
	}
	if result == nil {
		result = &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			Message: "Browser worker returned no screenshot result",
		}
	}

	now := time.Now().UTC()
	c.updateRuntime(resultRuntimeUpdates(result, map[string]interface{}{
		"browser_status":            BrowserStatusReady,
		"browser_process_alive":     true,
		"dev_tools_connected":       true,
		"last_browser_heartbeat_at": now,
		"last_error":                "",
		"current_action_step":       "Manual screenshot captured",
		"last_activity":             now,
	}))
	c.appendAndPublish(EventTypeBrowserScreenshot, WatcherEventInput{
		Level:   "info",
		Source:  "browser",
		Type:    EventTypeBrowserScreenshot,
		Message: "Manual screenshot captured",
		Data:    resultEventData(result, Job{}),
	})
	return result, nil
}

func (c *ActionCoordinator) Submit(job Job, autoAccept bool) error {
	c.blockMu.RLock()
	blocked := c.blocked
	c.blockMu.RUnlock()
	if blocked {
		return fmt.Errorf("browser worker is blocked")
	}

	select {
	case c.queue <- actionRequest{job: job, autoAccept: autoAccept}:
		return nil
	default:
		return fmt.Errorf("action queue full")
	}
}

func (c *ActionCoordinator) startBrowser(ctx context.Context) {
	if c.browser == nil {
		return
	}
	if !c.hasActionTouched() {
		c.updateRuntime(map[string]interface{}{
			"browser_status": BrowserStatusStarting,
			"last_activity":  time.Now().UTC(),
		})
	}
	if err := c.browser.Start(ctx); err != nil {
		msg := err.Error()
		if c.isBlocked() {
			return
		}
		c.updateRuntime(map[string]interface{}{
			"overall_status":            OverallStatusDegraded,
			"browser_status":            BrowserStatusFailed,
			"alert_status":              AlertStatusWarning,
			"profile_status":            ProfileStatusSeeded,
			"browser_process_alive":     false,
			"dev_tools_connected":       false,
			"last_error":                msg,
			"current_action_step":       "Browser worker unavailable; feed monitoring continues",
			"last_browser_heartbeat_at": nil,
			"last_activity":             time.Now().UTC(),
		})
		c.appendAndPublish(EventTypeBrowserStartFailed, WatcherEventInput{
			Level:   "warning",
			Source:  "browser",
			Type:    EventTypeBrowserStartFailed,
			Message: "Worker browser failed to start",
			Data: map[string]interface{}{
				"error": msg,
			},
		})
		return
	}

	now := time.Now().UTC()
	if !c.hasActionTouched() && !c.isBlocked() {
		c.updateRuntime(map[string]interface{}{
			"browser_status":            BrowserStatusReady,
			"profile_status":            ProfileStatusVerified,
			"browser_process_alive":     true,
			"dev_tools_connected":       true,
			"last_browser_heartbeat_at": now,
			"last_activity":             now,
		})
	}
	c.appendAndPublish(EventTypeBrowserReady, WatcherEventInput{
		Level:   "info",
		Source:  "browser",
		Type:    EventTypeBrowserReady,
		Message: "Worker browser ready",
	})
}

func (c *ActionCoordinator) isBlocked() bool {
	c.blockMu.RLock()
	defer c.blockMu.RUnlock()
	return c.blocked
}

func (c *ActionCoordinator) monitorBrowserHealth(ctx context.Context) {
	if c.browser == nil || c.config.HeartbeatInterval <= 0 {
		return
	}

	ticker := time.NewTicker(c.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case <-ticker.C:
			c.checkBrowserHealth(ctx)
		}
	}
}

func (c *ActionCoordinator) checkBrowserHealth(ctx context.Context) {
	c.blockMu.RLock()
	blocked := c.blocked
	c.blockMu.RUnlock()
	if blocked {
		return
	}

	healthCtx, cancel := context.WithTimeout(ctx, defaultBrowserHeartbeatTimeout)
	defer cancel()

	now := time.Now().UTC()
	if err := c.browser.Health(healthCtx); err != nil {
		if ctx.Err() != nil {
			return
		}
		c.updateRuntime(map[string]interface{}{
			"overall_status":            OverallStatusDegraded,
			"browser_status":            BrowserStatusFailed,
			"alert_status":              AlertStatusWarning,
			"browser_process_alive":     false,
			"dev_tools_connected":       false,
			"last_browser_heartbeat_at": nil,
			"last_error":                err.Error(),
			"current_action_step":       "Browser worker health check failed; feed monitoring continues",
			"last_activity":             now,
		})
		c.appendAndPublish(EventTypeBrowserStartFailed, WatcherEventInput{
			Level:   "warning",
			Source:  "browser",
			Type:    EventTypeBrowserStartFailed,
			Message: "Worker browser health check failed",
			Data: map[string]interface{}{
				"error": err.Error(),
			},
		})
		return
	}

	c.updateRuntime(map[string]interface{}{
		"browser_status":            BrowserStatusReady,
		"profile_status":            ProfileStatusVerified,
		"browser_process_alive":     true,
		"dev_tools_connected":       true,
		"last_browser_heartbeat_at": now,
		"last_error":                "",
		"last_activity":             now,
	})
}

func (c *ActionCoordinator) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case req := <-c.queue:
			c.handle(ctx, req)
		}
	}
}

func (c *ActionCoordinator) handle(ctx context.Context, req actionRequest) {
	c.setActionActive(true)
	defer c.setActionActive(false)
	c.setActionTouched()

	job := req.job
	now := time.Now().UTC()
	c.updateRuntime(map[string]interface{}{
		"current_job_id":      job.ID,
		"current_action_step": "Job queued for worker browser",
		"action_status":       ActionStatusQueued,
		"last_activity":       now,
	})
	c.appendAndPublish(EventTypeJobMatched, WatcherEventInput{
		Level:   "info",
		Source:  normalizeEventSource(job.Source),
		Type:    EventTypeJobMatched,
		JobID:   job.ID,
		Message: fmt.Sprintf("Job matched watcher filters: %s", job.Title),
		Data:    jobEventData(job),
	})

	if c.browser == nil {
		c.failJob(job, EventTypeBrowserJobOpenFail, BrowserOutcomeBrowserFailure, "Browser worker unavailable")
		return
	}

	c.updateRuntime(map[string]interface{}{
		"browser_status":      BrowserStatusBusy,
		"action_status":       ActionStatusOpening,
		"current_action_step": "Opening job page in worker browser",
		"last_activity":       time.Now().UTC(),
	})
	c.appendAndPublish(EventTypeBrowserJobOpenStart, WatcherEventInput{
		Level:   "info",
		Source:  "browser",
		Type:    EventTypeBrowserJobOpenStart,
		JobID:   job.ID,
		Message: "Opening job page in worker browser",
		Data:    jobEventData(job),
	})

	result, err := c.browser.OpenJob(ctx, job)
	if err != nil && result == nil {
		c.failJob(job, EventTypeBrowserJobOpenFail, BrowserOutcomeBrowserFailure, err.Error())
		return
	}
	if c.handleBlocked(job, result) {
		return
	}
	if err != nil {
		c.failJob(job, EventTypeBrowserJobOpenFail, BrowserOutcomeBrowserFailure, err.Error())
		return
	}

	c.updateRuntime(resultRuntimeUpdates(result, map[string]interface{}{
		"browser_status":            BrowserStatusReady,
		"action_status":             ActionStatusOpen,
		"browser_process_alive":     true,
		"dev_tools_connected":       true,
		"last_browser_heartbeat_at": time.Now().UTC(),
		"current_action_step":       "Job page opened in worker browser",
		"last_error":                "",
		"last_activity":             time.Now().UTC(),
	}))
	c.appendAndPublish(EventTypeBrowserJobOpenOK, WatcherEventInput{
		Level:   "info",
		Source:  "browser",
		Type:    EventTypeBrowserJobOpenOK,
		JobID:   job.ID,
		Message: "Job page opened in worker browser",
		Data:    resultEventData(result, job),
	})

	if !req.autoAccept {
		c.updateRuntime(map[string]interface{}{
			"action_status":       ActionStatusIdle,
			"current_action_step": "Job page opened; auto-accept disabled",
			"last_activity":       time.Now().UTC(),
		})
		return
	}

	c.updateRuntime(map[string]interface{}{
		"action_status":       ActionStatusAccepting,
		"current_action_step": "Clicking Accept in worker browser",
		"last_activity":       time.Now().UTC(),
	})
	c.appendAndPublish(EventTypeActionAcceptStarted, WatcherEventInput{
		Level:   "info",
		Source:  "action",
		Type:    EventTypeActionAcceptStarted,
		JobID:   job.ID,
		Message: "Accept click started",
		Data:    jobEventData(job),
	})

	acceptResult, err := c.browser.AcceptJob(ctx, job, c.config.AcceptSelector)
	if c.handleBlocked(job, acceptResult) {
		return
	}
	if err != nil {
		c.failJob(job, EventTypeActionAcceptFailed, BrowserOutcomeBrowserFailure, err.Error())
		return
	}
	if acceptResult == nil {
		c.failJob(job, EventTypeActionAcceptFailed, BrowserOutcomeBrowserFailure, "Browser worker returned no accept result")
		return
	}

	switch acceptResult.Outcome {
	case BrowserOutcomeAccepted:
		if c.report != nil {
			if err := c.report.RecordAcceptedJob(job); err != nil {
				log.Printf("[ACTION] User %s: failed to record accepted job %s: %v", c.userID, job.ID, err)
			}
		}
		c.updateRuntime(resultRuntimeUpdates(acceptResult, map[string]interface{}{
			"browser_status":      BrowserStatusReady,
			"action_status":       ActionStatusAccepted,
			"current_action_step": "Job accepted in worker browser",
			"last_error":          "",
			"last_activity":       time.Now().UTC(),
		}))
		c.appendAndPublish(EventTypeActionAcceptOK, WatcherEventInput{
			Level:   "info",
			Source:  "action",
			Type:    EventTypeActionAcceptOK,
			JobID:   job.ID,
			Message: "Job accepted in worker browser",
			Data:    resultEventData(acceptResult, job),
		})
	case BrowserOutcomeAlreadyGone, BrowserOutcomeTimeout, BrowserOutcomeBrowserFailure:
		c.failJob(job, EventTypeActionAcceptFailed, acceptResult.Outcome, acceptResult.Message)
	default:
		c.blockJob(job, acceptResult, EventTypeWorkerBlocked)
	}
}

func (c *ActionCoordinator) setActionActive(active bool) {
	c.actionMu.Lock()
	defer c.actionMu.Unlock()
	c.actionActive = active
}

func (c *ActionCoordinator) setActionTouched() {
	c.actionMu.Lock()
	defer c.actionMu.Unlock()
	c.actionTouched = true
}

func (c *ActionCoordinator) isActionActive() bool {
	c.actionMu.RLock()
	defer c.actionMu.RUnlock()
	return c.actionActive
}

func (c *ActionCoordinator) hasActionTouched() bool {
	c.actionMu.RLock()
	defer c.actionMu.RUnlock()
	return c.actionTouched
}

func (c *ActionCoordinator) failJob(
	job Job,
	eventType string,
	outcome BrowserActionOutcome,
	message string,
) {
	if strings.TrimSpace(message) == "" {
		message = string(outcome)
	}
	if eventType == "" {
		eventType = EventTypeActionAcceptFailed
	}
	source := "action"
	step := "Worker browser action failed"
	if eventType == EventTypeBrowserJobOpenFail {
		source = "browser"
		step = "Worker browser failed to open job page"
	}
	c.updateRuntime(map[string]interface{}{
		"overall_status":      OverallStatusDegraded,
		"browser_status":      BrowserStatusFailed,
		"action_status":       ActionStatusFailed,
		"alert_status":        AlertStatusWarning,
		"current_action_step": step,
		"last_error":          message,
		"last_activity":       time.Now().UTC(),
	})
	c.appendAndPublish(eventType, WatcherEventInput{
		Level:   "warning",
		Source:  source,
		Type:    eventType,
		JobID:   job.ID,
		Message: message,
		Data: map[string]interface{}{
			"outcome": outcome,
		},
	})
}

func (c *ActionCoordinator) handleBlocked(job Job, result *BrowserActionResult) bool {
	if result == nil || !result.IsBlocked() {
		return false
	}
	eventType := EventTypeWorkerBlocked
	switch result.Outcome {
	case BrowserOutcomeBlockedCaptcha:
		eventType = EventTypeBrowserCaptcha
	case BrowserOutcomeBlockedSuspiciousLogin:
		eventType = EventTypeBrowserSuspicious
	}
	c.blockJob(job, result, eventType)
	return true
}

func (c *ActionCoordinator) blockJob(job Job, result *BrowserActionResult, eventType string) {
	c.blockMu.Lock()
	c.blocked = true
	c.blockMu.Unlock()

	message := "Worker browser blocked"
	if result != nil && strings.TrimSpace(result.Message) != "" {
		message = result.Message
	}
	updates := map[string]interface{}{
		"overall_status":      OverallStatusBlocked,
		"browser_status":      BrowserStatusBlocked,
		"action_status":       ActionStatusBlocked,
		"alert_status":        AlertStatusCritical,
		"profile_status":      ProfileStatusBlocked,
		"last_critical_alert": message,
		"last_error":          message,
		"current_action_step": "Worker blocked; manual inspection required",
		"last_activity":       time.Now().UTC(),
	}
	if result != nil {
		updates = resultRuntimeUpdates(result, updates)
	}
	c.updateRuntime(updates)
	c.appendAndPublish(eventType, WatcherEventInput{
		Level:   "critical",
		Source:  "browser",
		Type:    eventType,
		JobID:   job.ID,
		Message: message,
		Data:    resultEventData(result, job),
	})
	if eventType != EventTypeWorkerBlocked {
		c.appendAndPublish(EventTypeWorkerBlocked, WatcherEventInput{
			Level:   "critical",
			Source:  "system",
			Type:    EventTypeWorkerBlocked,
			JobID:   job.ID,
			Message: "Worker action pipeline stopped",
			Data:    resultEventData(result, job),
		})
	}
}

func (c *ActionCoordinator) updateRuntime(updates map[string]interface{}) {
	if len(updates) == 0 || c.report == nil {
		return
	}
	if err := c.report.UpdateRuntime(updates); err != nil {
		log.Printf("[ACTION] User %s: failed to update runtime: %v", c.userID, err)
	}
}

func (c *ActionCoordinator) appendAndPublish(eventType string, input WatcherEventInput) {
	if c.report == nil {
		return
	}
	if input.Type == "" {
		input.Type = eventType
	}
	if input.Data == nil {
		input.Data = map[string]interface{}{}
	}
	if err := c.report.AppendEvent(input); err != nil {
		log.Printf("[ACTION] User %s: failed to append event %s: %v", c.userID, input.Type, err)
	}
	if err := c.report.PublishEvent(context.Background(), eventType, map[string]interface{}{
		"level":   input.Level,
		"source":  input.Source,
		"type":    input.Type,
		"job_id":  input.JobID,
		"message": input.Message,
		"data":    input.Data,
	}); err != nil {
		log.Printf("[ACTION] User %s: failed to publish event %s: %v", c.userID, eventType, err)
	}
}

func shouldAutoAccept(job Job, config *models.WatcherConfig) bool {
	if config == nil || !config.AutoAcceptEnabled {
		return false
	}
	minReward := config.MinReward
	maxReward := config.MaxReward
	if config.AutoAcceptMinReward != nil {
		minReward = *config.AutoAcceptMinReward
	}
	if config.AutoAcceptMaxReward != nil {
		maxReward = *config.AutoAcceptMaxReward
	}
	return job.Reward >= minReward && job.Reward <= maxReward
}

func resultRuntimeUpdates(result *BrowserActionResult, updates map[string]interface{}) map[string]interface{} {
	if updates == nil {
		updates = map[string]interface{}{}
	}
	if result == nil {
		return updates
	}
	if result.URL != "" {
		updates["current_url"] = result.URL
	}
	if result.Title != "" {
		updates["current_title"] = result.Title
	}
	if result.ScreenshotArtifactID != "" {
		updates["latest_screenshot_artifact_id"] = result.ScreenshotArtifactID
	}
	return updates
}

func resultEventData(result *BrowserActionResult, job Job) map[string]interface{} {
	data := jobEventData(job)
	if result == nil {
		return data
	}
	data["outcome"] = result.Outcome
	data["url"] = result.URL
	data["title"] = result.Title
	data["message"] = result.Message
	if result.ScreenshotArtifactID != "" {
		data["screenshot_artifact_id"] = result.ScreenshotArtifactID
	}
	return data
}

func jobEventData(job Job) map[string]interface{} {
	return map[string]interface{}{
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
}
