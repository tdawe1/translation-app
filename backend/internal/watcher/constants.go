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

// Event types
const (
	EventTypeWatcherStarted = "watcher_started"
	EventTypeWatcherStopped = "watcher_stopped"
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
