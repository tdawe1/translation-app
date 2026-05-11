package watcher

import "fmt"

const (
	// Default job channel buffer size
	DefaultJobBufferSize = 1000

	// Default seen jobs set size limit
	DefaultSeenJobsLimit = 10000
)

// Redis key patterns for user-specific data
const (
	// RedisKeySeenJobs is the key pattern for storing seen job IDs
	RedisKeySeenJobs = "user:%s:seen_jobs"

	// RedisKeyJobsChannel is the pub/sub channel for new jobs
	RedisKeyJobsChannel = "user:%s:jobs"

	// RedisKeyEventsChannel is the pub/sub channel for watcher events
	RedisKeyEventsChannel = "user:%s:events"

	// RedisKeyErrorsChannel is the pub/sub channel for errors
	RedisKeyErrorsChannel = "user:%s:errors"
)

// Watcher status constants
const (
	StatusStopped = "stopped"
	StatusRunning = "running"
	StatusError   = "error"
)

// Runtime domain status constants.
const (
	OverallStatusStopped  = "stopped"
	OverallStatusRunning  = "running"
	OverallStatusDegraded = "degraded"
	OverallStatusBlocked  = "blocked"

	FeedStatusStopped    = "stopped"
	FeedStatusMonitoring = "monitoring"

	BrowserStatusUnconfigured = "unconfigured"
	BrowserStatusDashboard    = "dashboard"
	BrowserStatusStarting     = "starting"
	BrowserStatusReady        = "ready"
	BrowserStatusBusy         = "busy"
	BrowserStatusBlocked      = "blocked"
	BrowserStatusFailed       = "failed"
	BrowserStatusStopped      = "stopped"

	ActionStatusIdle      = "idle"
	ActionStatusQueued    = "queued"
	ActionStatusOpening   = "opening"
	ActionStatusOpen      = "open"
	ActionStatusAccepting = "accepting"
	ActionStatusAccepted  = "accepted"
	ActionStatusFailed    = "failed"
	ActionStatusBlocked   = "blocked"

	AlertStatusNone     = "none"
	AlertStatusWarning  = "warning"
	AlertStatusCritical = "critical"

	ProfileStatusUnseeded = "unseeded"
	ProfileStatusSeeded   = "seeded"
	ProfileStatusVerified = "verified"
	ProfileStatusBlocked  = "blocked"
)

// Event types.
const (
	EventTypeWorkerStarted       = "worker.started"
	EventTypeWorkerStopped       = "worker.stopped"
	EventTypeWorkerReady         = "worker.ready"
	EventTypeWorkerBlocked       = "worker.blocked"
	EventTypeWorkerRestoreStart  = "worker.restore_started"
	EventTypeWorkerRestoreOK     = "worker.restore_succeeded"
	EventTypeWorkerRestoreFailed = "worker.restore_failed"
	EventTypeWorkerShutdown      = "worker.shutdown_persisted"
	EventTypeBrowserUnconfigured = "browser.unconfigured"
	EventTypeBrowserDashboard    = "browser.dashboard_mode"
	EventTypeBrowserStarted      = "browser.started"
	EventTypeBrowserReady        = "browser.ready"
	EventTypeBrowserStartFailed  = "browser.start_failed"
	EventTypeBrowserJobOpenStart = "browser.job_open_started"
	EventTypeBrowserJobOpenOK    = "browser.job_open_succeeded"
	EventTypeBrowserJobOpenFail  = "browser.job_open_failed"
	EventTypeBrowserScreenshot   = "browser.screenshot_captured"
	EventTypeBrowserCaptcha      = "browser.captcha_detected"
	EventTypeBrowserSuspicious   = "browser.suspicious_login_detected"
	EventTypeActionAcceptStarted = "action.accept_started"
	EventTypeActionAcceptOK      = "action.accept_succeeded"
	EventTypeActionAcceptFailed  = "action.accept_failed"
	EventTypeJobDetected         = "job.detected"
	EventTypeJobMatched          = "job.matched"
	EventTypeRSSPollStarted      = "rss.poll_started"
	EventTypeRSSPollOK           = "rss.poll_ok"
	EventTypeWebSocketConnected  = "websocket.connected"
	EventTypeWebSocketPong       = "websocket.pong"
	EventTypeWebSocketMessage    = "websocket.message"
	EventTypeWebSocketClosed     = "websocket.disconnected"
	EventTypeWatcherHealth       = "watcher.health"
)

// GetSeenJobsKey returns the Redis key for a user's seen jobs set
func GetSeenJobsKey(userID string) string {
	return fmt.Sprintf(RedisKeySeenJobs, userID)
}

// GetJobsChannel returns the Redis pub/sub channel for a user's jobs
func GetJobsChannel(userID string) string {
	return fmt.Sprintf(RedisKeyJobsChannel, userID)
}

// GetEventsChannel returns the Redis pub/sub channel for a user's events
func GetEventsChannel(userID string) string {
	return fmt.Sprintf(RedisKeyEventsChannel, userID)
}

// GetErrorsChannel returns the Redis pub/sub channel for a user's errors
func GetErrorsChannel(userID string) string {
	return fmt.Sprintf(RedisKeyErrorsChannel, userID)
}
