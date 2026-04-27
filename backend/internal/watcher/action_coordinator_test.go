package watcher

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/internal/models"
)

type fakeActionReporter struct {
	mu       sync.Mutex
	runtime  map[string]interface{}
	events   []WatcherEventInput
	accepted []Job
}

func (r *fakeActionReporter) UpdateRuntime(updates map[string]interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runtime == nil {
		r.runtime = map[string]interface{}{}
	}
	for key, value := range updates {
		r.runtime[key] = value
	}
	return nil
}

func (r *fakeActionReporter) AppendEvent(input WatcherEventInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, input)
	return nil
}

func (r *fakeActionReporter) PublishEvent(context.Context, string, map[string]interface{}) error {
	return nil
}

func (r *fakeActionReporter) RecordAcceptedJob(job Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.accepted = append(r.accepted, job)
	return nil
}

func (r *fakeActionReporter) hasEvent(eventType string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, event := range r.events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func (r *fakeActionReporter) runtimeValue(key string) interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runtime[key]
}

func (r *fakeActionReporter) snapshot() (map[string]interface{}, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	runtime := map[string]interface{}{}
	for key, value := range r.runtime {
		runtime[key] = value
	}
	events := make([]string, 0, len(r.events))
	for _, event := range r.events {
		events = append(events, event.Type)
	}
	return runtime, events
}

type fakeBrowserController struct {
	openResult   *BrowserActionResult
	acceptResult *BrowserActionResult
	healthErr    error
	openErr      error
	delay        time.Duration

	mu        sync.Mutex
	active    int
	maxActive int
	opened    int
}

func (b *fakeBrowserController) Start(context.Context) error { return nil }
func (b *fakeBrowserController) Stop(context.Context) error  { return nil }
func (b *fakeBrowserController) Restart(context.Context) error {
	return nil
}
func (b *fakeBrowserController) Health(context.Context) error {
	return b.healthErr
}
func (b *fakeBrowserController) CaptureScreenshot(context.Context) (*BrowserActionResult, error) {
	return &BrowserActionResult{
		Outcome:              BrowserOutcomeOpened,
		URL:                  "https://gengo.com/dashboard/jobs/1",
		Title:                "Job 1",
		ScreenshotArtifactID: "manual.png",
	}, nil
}

func (b *fakeBrowserController) OpenJob(context.Context, Job) (*BrowserActionResult, error) {
	b.begin()
	defer b.end()
	if b.delay > 0 {
		time.Sleep(b.delay)
	}
	if b.openResult != nil {
		return b.openResult, nil
	}
	if b.openErr != nil {
		return nil, b.openErr
	}
	return &BrowserActionResult{Outcome: BrowserOutcomeOpened, URL: "https://gengo.com/dashboard/jobs/1"}, nil
}

func (b *fakeBrowserController) AcceptJob(context.Context, Job, string) (*BrowserActionResult, error) {
	if b.acceptResult != nil {
		return b.acceptResult, nil
	}
	return &BrowserActionResult{Outcome: BrowserOutcomeAccepted, URL: "https://gengo.com/t/workbench/1"}, nil
}

func (b *fakeBrowserController) begin() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.active++
	b.opened++
	if b.active > b.maxActive {
		b.maxActive = b.active
	}
}

func (b *fakeBrowserController) end() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.active--
}

func TestShouldAutoAcceptUsesDedicatedRewardBounds(t *testing.T) {
	minReward := 10.0
	maxReward := 25.0
	config := testWatcherConfig(true, 1, 50)
	config.AutoAcceptMinReward = &minReward
	config.AutoAcceptMaxReward = &maxReward

	require.False(t, shouldAutoAccept(Job{Reward: 9.99}, config))
	require.True(t, shouldAutoAccept(Job{Reward: 10}, config))
	require.True(t, shouldAutoAccept(Job{Reward: 25}, config))
	require.False(t, shouldAutoAccept(Job{Reward: 25.01}, config))
}

func TestActionCoordinatorBlocksOnCaptchaResult(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := &fakeActionReporter{}
	browser := &fakeBrowserController{
		openResult: &BrowserActionResult{
			Outcome: BrowserOutcomeBlockedCaptcha,
			URL:     "https://gengo.com/dashboard/jobs/1",
			Message: "Captcha or challenge marker detected",
		},
	}
	coordinator := NewActionCoordinator(
		uuid.New(),
		browser,
		reporter,
		ActionCoordinatorConfig{QueueSize: 2},
	)
	coordinator.Start(ctx)

	err := coordinator.Submit(testJob("1"), false)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return reporter.runtimeValue("overall_status") == OverallStatusBlocked &&
			reporter.hasEvent(EventTypeBrowserCaptcha) &&
			reporter.hasEvent(EventTypeWorkerBlocked)
	}, time.Second, 10*time.Millisecond)

	err = coordinator.Submit(testJob("2"), false)
	require.Error(t, err)
}

func TestActionCoordinatorHealthDoesNotOverwriteBlockedState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := &fakeActionReporter{}
	browser := &fakeBrowserController{
		openResult: &BrowserActionResult{
			Outcome: BrowserOutcomeBlockedSuspiciousLogin,
			URL:     "https://gengo.com/auth/form/login",
			Message: "Worker browser was redirected to login during action flow",
		},
	}
	coordinator := NewActionCoordinator(
		uuid.New(),
		browser,
		reporter,
		ActionCoordinatorConfig{QueueSize: 2, HeartbeatInterval: 5 * time.Millisecond},
	)
	coordinator.Start(ctx)

	require.NoError(t, coordinator.Submit(testJob("1"), false))
	require.Eventually(t, func() bool {
		return reporter.runtimeValue("overall_status") == OverallStatusBlocked
	}, time.Second, 10*time.Millisecond)

	time.Sleep(30 * time.Millisecond)
	require.Equal(t, OverallStatusBlocked, reporter.runtimeValue("overall_status"))
	require.Equal(t, BrowserStatusBlocked, reporter.runtimeValue("browser_status"))
	require.Equal(t, ProfileStatusBlocked, reporter.runtimeValue("profile_status"))
}

func TestActionCoordinatorSerializesBrowserOpen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := &fakeActionReporter{}
	browser := &fakeBrowserController{delay: 20 * time.Millisecond}
	coordinator := NewActionCoordinator(
		uuid.New(),
		browser,
		reporter,
		ActionCoordinatorConfig{QueueSize: 2},
	)
	coordinator.Start(ctx)

	require.NoError(t, coordinator.Submit(testJob("1"), false))
	require.NoError(t, coordinator.Submit(testJob("2"), false))

	require.Eventually(t, func() bool {
		browser.mu.Lock()
		defer browser.mu.Unlock()
		return browser.opened == 2
	}, time.Second, 10*time.Millisecond)

	browser.mu.Lock()
	maxActive := browser.maxActive
	browser.mu.Unlock()
	require.Equal(t, 1, maxActive)
}

func TestActionCoordinatorMarksBrowserFailedOnHealthError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := &fakeActionReporter{}
	browser := &fakeBrowserController{healthErr: context.DeadlineExceeded}
	coordinator := NewActionCoordinator(
		uuid.New(),
		browser,
		reporter,
		ActionCoordinatorConfig{QueueSize: 2, HeartbeatInterval: 5 * time.Millisecond},
	)
	coordinator.Start(ctx)

	require.Eventually(t, func() bool {
		return reporter.runtimeValue("browser_status") == BrowserStatusFailed &&
			reporter.runtimeValue("browser_process_alive") == false &&
			reporter.hasEvent(EventTypeBrowserStartFailed)
	}, time.Second, 10*time.Millisecond)
}

func TestActionCoordinatorClassifiesOpenFailureSeparately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := &fakeActionReporter{}
	browser := &fakeBrowserController{openErr: context.DeadlineExceeded}
	coordinator := NewActionCoordinator(
		uuid.New(),
		browser,
		reporter,
		ActionCoordinatorConfig{QueueSize: 2},
	)
	coordinator.Start(ctx)

	require.NoError(t, coordinator.Submit(testJob("1"), false))

	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	ok := false
	for !ok {
		select {
		case <-deadline:
			runtime, events := reporter.snapshot()
			t.Fatalf("runtime=%v events=%v", runtime, events)
		case <-ticker.C:
			ok = reporter.runtimeValue("browser_status") == BrowserStatusFailed &&
				reporter.runtimeValue("current_action_step") == "Worker browser failed to open job page" &&
				reporter.hasEvent(EventTypeBrowserJobOpenFail)
		}
	}
}

func TestActionCoordinatorRecordsOpenScreenshotArtifact(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := &fakeActionReporter{}
	browser := &fakeBrowserController{
		openResult: &BrowserActionResult{
			Outcome:              BrowserOutcomeOpened,
			URL:                  "https://gengo.com/dashboard/jobs/1",
			Title:                "Job 1",
			ScreenshotArtifactID: "attempt-1.png",
		},
	}
	coordinator := NewActionCoordinator(
		uuid.New(),
		browser,
		reporter,
		ActionCoordinatorConfig{QueueSize: 2},
	)
	coordinator.Start(ctx)

	require.NoError(t, coordinator.Submit(testJob("1"), false))

	require.Eventually(t, func() bool {
		return reporter.runtimeValue("latest_screenshot_artifact_id") == "attempt-1.png" &&
			reporter.hasEvent(EventTypeBrowserJobOpenOK)
	}, time.Second, 10*time.Millisecond)
}

func TestActionCoordinatorCapturesManualScreenshot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reporter := &fakeActionReporter{}
	browser := &fakeBrowserController{}
	coordinator := NewActionCoordinator(
		uuid.New(),
		browser,
		reporter,
		ActionCoordinatorConfig{QueueSize: 2},
	)

	result, err := coordinator.CaptureScreenshot(ctx)

	require.NoError(t, err)
	require.Equal(t, "manual.png", result.ScreenshotArtifactID)
	require.Equal(t, "manual.png", reporter.runtimeValue("latest_screenshot_artifact_id"))
	require.True(t, reporter.hasEvent(EventTypeBrowserScreenshot))
}

func testJob(id string) Job {
	return Job{
		ID:     id,
		Title:  "Job " + id,
		Reward: 12,
		URL:    "https://gengo.com/dashboard/jobs/" + id,
		Source: "rss",
		UserID: uuid.New(),
	}
}

func testWatcherConfig(autoAccept bool, minReward float64, maxReward float64) *models.WatcherConfig {
	return &models.WatcherConfig{
		AutoAcceptEnabled: autoAccept,
		MinReward:         minReward,
		MaxReward:         maxReward,
	}
}
