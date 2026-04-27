package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func browserStateString(value string) *string {
	return &value
}

func browserStateBool(value bool) *bool {
	return &value
}

func TestBuildBrowserStateUpdatesAcceptsPublicState(t *testing.T) {
	now := time.Date(2026, 4, 24, 10, 15, 0, 0, time.UTC)

	updates, msg := buildBrowserStateUpdates(SyncBrowserStateRequest{
		CurrentURL:          browserStateString("https://gengo.com/dashboard/jobs/job-123"),
		CurrentTitle:        browserStateString("Japanese to English"),
		CurrentActionStep:   browserStateString("Opened job page from dashboard"),
		CurrentJobID:        browserStateString("job-123"),
		LoggedInState:       browserStateString("unknown"),
		BrowserProcessAlive: browserStateBool(true),
		DevToolsConnected:   browserStateBool(false),
	}, now)

	require.Empty(t, msg)
	assert.Equal(t, "dashboard", updates["browser_status"])
	assert.Equal(t, "idle", updates["action_status"])
	assert.Equal(t, "https://gengo.com/dashboard/jobs/job-123", updates["current_url"])
	assert.Equal(t, "Japanese to English", updates["current_title"])
	assert.Equal(t, "Opened job page from dashboard", updates["current_action_step"])
	assert.Equal(t, "job-123", updates["current_job_id"])
	assert.Equal(t, "unknown", updates["logged_in_state"])
	assert.Equal(t, true, updates["browser_process_alive"])
	assert.Equal(t, false, updates["dev_tools_connected"])
	assert.Equal(t, now, updates["last_browser_heartbeat_at"])
	assert.Equal(t, now, updates["last_activity"])
}

func TestBuildBrowserStateUpdatesRejectsLocalURLs(t *testing.T) {
	localURLs := []string{
		"http://localhost:3000/dashboard",
		"http://127.0.0.1:8000/job",
		"http://192.168.1.5/job",
		"http://10.0.0.2/job",
		"http://example.local/job",
		"file:///tmp/job",
	}

	for _, localURL := range localURLs {
		t.Run(localURL, func(t *testing.T) {
			updates, msg := buildBrowserStateUpdates(SyncBrowserStateRequest{
				CurrentURL: browserStateString(localURL),
			}, time.Now())

			assert.Nil(t, updates)
			assert.Equal(t, "current_url must be a safe public HTTP(S) URL", msg)
		})
	}
}

func TestBuildBrowserStateUpdatesAcceptsFrontendURLSeparately(t *testing.T) {
	now := time.Date(2026, 4, 26, 9, 30, 0, 0, time.UTC)

	updates, msg := buildBrowserStateUpdates(SyncBrowserStateRequest{
		FrontendURL:   browserStateString("http://10.73.0.2:37180/dashboard"),
		FrontendTitle: browserStateString("GengoWatcher Dashboard"),
	}, now)

	require.Empty(t, msg)
	assert.Equal(t, "http://10.73.0.2:37180/dashboard", updates["frontend_url"])
	assert.Equal(t, "GengoWatcher Dashboard", updates["frontend_title"])
	assert.Equal(t, now, updates["frontend_last_seen_at"])
	assert.Equal(t, now, updates["last_activity"])
	assert.NotContains(t, updates, "browser_status")
	assert.NotContains(t, updates, "last_browser_heartbeat_at")
}

func TestBuildBrowserStateUpdatesRejectsInvalidFrontendURL(t *testing.T) {
	updates, msg := buildBrowserStateUpdates(SyncBrowserStateRequest{
		FrontendURL: browserStateString("http://user:pass@10.73.0.2:37180/dashboard"),
	}, time.Now())

	assert.Nil(t, updates)
	assert.Equal(t, "frontend_url must be an HTTP(S) URL without credentials", msg)
}

func TestBuildBrowserStateUpdatesRejectsUnexpectedLoginState(t *testing.T) {
	updates, msg := buildBrowserStateUpdates(SyncBrowserStateRequest{
		LoggedInState: browserStateString("maybe"),
	}, time.Now())

	assert.Nil(t, updates)
	assert.Equal(t, "logged_in_state must be unknown, logged_in, or logged_out", msg)
}
