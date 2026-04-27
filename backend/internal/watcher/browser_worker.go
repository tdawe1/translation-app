package watcher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	defaultBrowserStartTimeout  = 15 * time.Second
	defaultBrowserActionTimeout = 12 * time.Second
	defaultBrowserProfileRoot   = "data/watcher-profiles"
	defaultBrowserArtifactRoot  = "data/watcher-artifacts"
	defaultAcceptSelector       = "text=Accept"
)

type BrowserActionOutcome string

const (
	BrowserOutcomeOpened                 BrowserActionOutcome = "opened"
	BrowserOutcomeAccepted               BrowserActionOutcome = "accepted"
	BrowserOutcomeAlreadyGone            BrowserActionOutcome = "already_gone"
	BrowserOutcomeBlockedCaptcha         BrowserActionOutcome = "blocked_captcha"
	BrowserOutcomeBlockedSuspiciousLogin BrowserActionOutcome = "blocked_suspicious_login"
	BrowserOutcomeUnexpectedPageState    BrowserActionOutcome = "unexpected_page_state"
	BrowserOutcomeBrowserFailure         BrowserActionOutcome = "browser_failure"
	BrowserOutcomeTimeout                BrowserActionOutcome = "timeout"
)

// BrowserController is the narrow browser surface used by the action coordinator.
type BrowserController interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
	Health(ctx context.Context) error
	CaptureScreenshot(ctx context.Context) (*BrowserActionResult, error)
	OpenJob(ctx context.Context, job Job) (*BrowserActionResult, error)
	AcceptJob(ctx context.Context, job Job, selector string) (*BrowserActionResult, error)
}

type BrowserActionResult struct {
	Outcome              BrowserActionOutcome `json:"outcome"`
	URL                  string               `json:"url,omitempty"`
	Title                string               `json:"title,omitempty"`
	Message              string               `json:"message,omitempty"`
	ScreenshotArtifactID string               `json:"screenshot_artifact_id,omitempty"`
}

func (r *BrowserActionResult) IsBlocked() bool {
	if r == nil {
		return false
	}
	return r.Outcome == BrowserOutcomeBlockedCaptcha ||
		r.Outcome == BrowserOutcomeBlockedSuspiciousLogin ||
		r.Outcome == BrowserOutcomeUnexpectedPageState
}

type BrowserWorkerConfig struct {
	BinaryPath    string
	Backend       string
	DebugURL      string
	ProfileRoot   string
	ArtifactRoot  string
	Headless      bool
	StartTimeout  time.Duration
	ActionTimeout time.Duration
	Disabled      bool
}

func BrowserWorkerConfigFromEnv() BrowserWorkerConfig {
	return BrowserWorkerConfig{
		BinaryPath:    strings.TrimSpace(os.Getenv("WATCHER_BROWSER_BINARY")),
		Backend:       strings.TrimSpace(os.Getenv("WATCHER_BROWSER_BACKEND")),
		DebugURL:      strings.TrimSpace(os.Getenv("WATCHER_BROWSER_DEBUG_URL")),
		ProfileRoot:   getEnvDefault("WATCHER_BROWSER_PROFILE_ROOT", defaultBrowserProfileRoot),
		ArtifactRoot:  getEnvDefault("WATCHER_BROWSER_ARTIFACT_ROOT", defaultBrowserArtifactRoot),
		Headless:      envBool("WATCHER_BROWSER_HEADLESS", defaultBrowserHeadless()),
		StartTimeout:  envDuration("WATCHER_BROWSER_START_TIMEOUT", defaultBrowserStartTimeout),
		ActionTimeout: envDuration("WATCHER_BROWSER_ACTION_TIMEOUT", defaultBrowserActionTimeout),
		Disabled:      envBool("WATCHER_BROWSER_DISABLED", false),
	}
}

func NewConfiguredBrowserWorker(userID uuid.UUID, config BrowserWorkerConfig) BrowserController {
	backend := strings.ToLower(strings.TrimSpace(config.Backend))
	debugURL := strings.TrimSpace(config.DebugURL)
	if backend == "firefox-rdp" || (backend == "" && looksLikeFirefoxRDPDebugURL(debugURL)) {
		return NewFirefoxRDPBrowserWorker(userID, config)
	}
	return NewBrowserWorker(userID, config)
}

func ArtifactPathForUser(userID uuid.UUID, artifactID string) (string, error) {
	if !safeArtifactID(artifactID) {
		return "", fmt.Errorf("invalid artifact id")
	}
	root := getEnvDefault("WATCHER_BROWSER_ARTIFACT_ROOT", defaultBrowserArtifactRoot)
	return filepath.Join(root, userID.String(), artifactID), nil
}

type BrowserWorker struct {
	userID    uuid.UUID
	config    BrowserWorkerConfig
	validator *URLValidator
	client    *http.Client

	mu         sync.Mutex
	cmd        *exec.Cmd
	devtools   *devtoolsEndpoint
	profileDir string
}

type devtoolsEndpoint struct {
	port int
	base string
}

type devtoolsTarget struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

func NewBrowserWorker(userID uuid.UUID, config BrowserWorkerConfig) *BrowserWorker {
	if config.ProfileRoot == "" {
		config.ProfileRoot = defaultBrowserProfileRoot
	}
	if config.ArtifactRoot == "" {
		config.ArtifactRoot = defaultBrowserArtifactRoot
	}
	if config.StartTimeout <= 0 {
		config.StartTimeout = defaultBrowserStartTimeout
	}
	if config.ActionTimeout <= 0 {
		config.ActionTimeout = defaultBrowserActionTimeout
	}
	return &BrowserWorker{
		userID:    userID,
		config:    config,
		validator: NewURLValidator(),
		client: &http.Client{
			Timeout: config.ActionTimeout,
		},
	}
}

func (w *BrowserWorker) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.config.Disabled {
		return fmt.Errorf("browser worker disabled by WATCHER_BROWSER_DISABLED")
	}
	if w.isRunningLocked() && w.devtools != nil {
		return nil
	}

	binary, err := w.resolveBinary()
	if err != nil {
		return err
	}

	profileDir := filepath.Join(w.config.ProfileRoot, w.userID.String())
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return fmt.Errorf("prepare browser profile: %w", err)
	}

	args := []string{
		"--remote-debugging-port=0",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-component-update",
		"--password-store=basic",
		"--user-data-dir=" + profileDir,
		"about:blank",
	}
	if w.config.Headless {
		args = append([]string{"--headless=new", "--disable-gpu"}, args...)
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start browser worker: %w", err)
	}

	w.cmd = cmd
	w.profileDir = profileDir

	endpoint, err := w.waitForDevTools(ctx, profileDir)
	if err != nil {
		_ = w.stopLocked()
		return err
	}
	w.devtools = endpoint
	return nil
}

func (w *BrowserWorker) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stopLocked()
}

func (w *BrowserWorker) Restart(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := w.Stop(ctx); err != nil {
		return err
	}
	return w.Start(ctx)
}

func (w *BrowserWorker) Health(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	w.mu.Lock()
	running := w.isRunningLocked()
	endpoint := w.devtools
	w.mu.Unlock()

	if !running || endpoint == nil {
		return w.Start(ctx)
	}

	if err := w.pingVersion(ctx, endpoint.base); err != nil {
		w.mu.Lock()
		w.devtools = nil
		w.mu.Unlock()
		return fmt.Errorf("browser DevTools health check failed: %w", err)
	}
	return nil
}

func (w *BrowserWorker) CaptureScreenshot(ctx context.Context) (*BrowserActionResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := w.Start(ctx); err != nil {
		return nil, err
	}

	target, err := w.currentPageTarget(ctx)
	if err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			Message: err.Error(),
		}, err
	}
	if target == nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeUnexpectedPageState,
			Message: "No browser page target available for screenshot",
		}, nil
	}

	actionCtx, cancel := context.WithTimeout(ctx, w.config.ActionTimeout)
	defer cancel()

	client, err := newCDPClient(actionCtx, target.WebSocketDebuggerURL)
	if err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			URL:     target.URL,
			Title:   target.Title,
			Message: err.Error(),
		}, err
	}
	defer client.Close()

	_ = client.Call(actionCtx, "Runtime.enable", nil, nil)
	_ = client.Call(actionCtx, "Page.enable", nil, nil)
	_ = waitForDocumentReady(actionCtx, client)

	var currentURL string
	if err := evaluateRuntime(actionCtx, client, `location.href`, &currentURL); err != nil {
		currentURL = target.URL
	}
	var currentTitle string
	if err := evaluateRuntime(actionCtx, client, `document.title`, &currentTitle); err != nil {
		currentTitle = target.Title
	}

	artifactID := w.captureScreenshot(actionCtx, client, "manual")
	if artifactID == "" {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			URL:     currentURL,
			Title:   currentTitle,
			Message: "Failed to capture browser screenshot",
		}, nil
	}

	return &BrowserActionResult{
		Outcome:              BrowserOutcomeOpened,
		URL:                  currentURL,
		Title:                currentTitle,
		Message:              "Browser screenshot captured",
		ScreenshotArtifactID: artifactID,
	}, nil
}

func (w *BrowserWorker) OpenJob(ctx context.Context, job Job) (*BrowserActionResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := w.validator.Validate(job.URL); err != nil {
		return nil, fmt.Errorf("job URL validation failed: %w", err)
	}
	if err := w.Start(ctx); err != nil {
		return nil, err
	}

	target, err := w.openTarget(ctx, job.URL)
	if err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			Message: err.Error(),
		}, err
	}

	result, err := w.inspectTarget(ctx, target, job, false, "")
	if err != nil {
		return result, err
	}
	if result.Outcome == "" {
		result.Outcome = BrowserOutcomeOpened
	}
	return result, nil
}

func (w *BrowserWorker) AcceptJob(ctx context.Context, job Job, selector string) (*BrowserActionResult, error) {
	if selector == "" {
		selector = defaultAcceptSelector
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := w.validator.Validate(job.URL); err != nil {
		return nil, fmt.Errorf("job URL validation failed: %w", err)
	}
	if err := w.Start(ctx); err != nil {
		return nil, err
	}

	target, err := w.findTargetForURL(ctx, job.URL)
	if err != nil || target == nil {
		target, err = w.openTarget(ctx, job.URL)
	}
	if err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			Message: err.Error(),
		}, err
	}

	result, err := w.inspectTarget(ctx, target, job, true, selector)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (w *BrowserWorker) resolveBinary() (string, error) {
	if w.config.BinaryPath != "" {
		if _, err := os.Stat(w.config.BinaryPath); err == nil {
			return w.config.BinaryPath, nil
		}
		if path, err := exec.LookPath(w.config.BinaryPath); err == nil {
			return path, nil
		}
		return "", fmt.Errorf("configured browser binary not found: %s", w.config.BinaryPath)
	}

	for _, candidate := range []string{
		"chromium",
		"chromium-browser",
		"google-chrome",
		"google-chrome-stable",
		"brave-browser",
		"microsoft-edge",
	} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no Chromium-family browser found; set WATCHER_BROWSER_BINARY")
}

func (w *BrowserWorker) waitForDevTools(ctx context.Context, profileDir string) (*devtoolsEndpoint, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, w.config.StartTimeout)
	defer cancel()

	activePortPath := filepath.Join(profileDir, "DevToolsActivePort")
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return nil, fmt.Errorf("browser DevTools endpoint not ready: %w", timeoutCtx.Err())
		case <-ticker.C:
			if !w.isRunningLocked() {
				return nil, fmt.Errorf("browser process exited before DevTools became ready")
			}
			raw, err := os.ReadFile(activePortPath)
			if err != nil {
				continue
			}
			lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
			if len(lines) == 0 {
				continue
			}
			port, err := strconv.Atoi(strings.TrimSpace(lines[0]))
			if err != nil || port <= 0 {
				continue
			}
			base := fmt.Sprintf("http://127.0.0.1:%d", port)
			if err := w.pingVersion(timeoutCtx, base); err != nil {
				continue
			}
			return &devtoolsEndpoint{port: port, base: base}, nil
		}
	}
}

func (w *BrowserWorker) pingVersion(ctx context.Context, base string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/json/version", nil)
	if err != nil {
		return err
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected DevTools status %d", resp.StatusCode)
	}
	return nil
}

func (w *BrowserWorker) openTarget(ctx context.Context, rawURL string) (*devtoolsTarget, error) {
	w.mu.Lock()
	endpoint := w.devtools
	w.mu.Unlock()
	if endpoint == nil {
		return nil, fmt.Errorf("browser DevTools endpoint unavailable")
	}

	targetURL := endpoint.base + "/json/new?" + url.QueryEscape(rawURL)
	target, err := w.callTargetEndpoint(ctx, http.MethodPut, targetURL)
	if err == nil {
		return target, nil
	}
	return w.callTargetEndpoint(ctx, http.MethodGet, targetURL)
}

func (w *BrowserWorker) callTargetEndpoint(ctx context.Context, method string, endpoint string) (*devtoolsTarget, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("DevTools target endpoint returned %d", resp.StatusCode)
	}
	var target devtoolsTarget
	if err := json.NewDecoder(resp.Body).Decode(&target); err != nil {
		return nil, fmt.Errorf("decode DevTools target: %w", err)
	}
	if target.WebSocketDebuggerURL == "" {
		return nil, fmt.Errorf("DevTools target has no websocket URL")
	}
	return &target, nil
}

func (w *BrowserWorker) findTargetForURL(ctx context.Context, rawURL string) (*devtoolsTarget, error) {
	targets, err := w.listTargets(ctx)
	if err != nil {
		return nil, err
	}
	for _, target := range targets {
		if target.Type == "page" && target.URL == rawURL && target.WebSocketDebuggerURL != "" {
			return &target, nil
		}
	}
	return nil, nil
}

func (w *BrowserWorker) currentPageTarget(ctx context.Context) (*devtoolsTarget, error) {
	targets, err := w.listTargets(ctx)
	if err != nil {
		return nil, err
	}
	var fallback *devtoolsTarget
	for index := range targets {
		target := targets[index]
		if target.Type != "page" || target.WebSocketDebuggerURL == "" {
			continue
		}
		if fallback == nil {
			copied := target
			fallback = &copied
		}
		if target.URL != "" && target.URL != "about:blank" {
			copied := target
			return &copied, nil
		}
	}
	return fallback, nil
}

func (w *BrowserWorker) listTargets(ctx context.Context) ([]devtoolsTarget, error) {
	w.mu.Lock()
	endpoint := w.devtools
	w.mu.Unlock()
	if endpoint == nil {
		return nil, fmt.Errorf("browser DevTools endpoint unavailable")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.base+"/json/list", nil)
	if err != nil {
		return nil, err
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DevTools list returned %d", resp.StatusCode)
	}
	var targets []devtoolsTarget
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, err
	}
	return targets, nil
}

func (w *BrowserWorker) inspectTarget(
	ctx context.Context,
	target *devtoolsTarget,
	job Job,
	accept bool,
	selector string,
) (*BrowserActionResult, error) {
	actionCtx, cancel := context.WithTimeout(ctx, w.config.ActionTimeout)
	defer cancel()

	client, err := newCDPClient(actionCtx, target.WebSocketDebuggerURL)
	if err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			URL:     target.URL,
			Title:   target.Title,
			Message: err.Error(),
		}, err
	}
	defer client.Close()

	_ = client.Call(actionCtx, "Runtime.enable", nil, nil)
	_ = client.Call(actionCtx, "Page.enable", nil, nil)
	if err := client.Call(actionCtx, "Page.navigate", map[string]interface{}{"url": job.URL}, nil); err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			URL:     target.URL,
			Title:   target.Title,
			Message: err.Error(),
		}, err
	}
	_ = waitForDocumentReady(actionCtx, client)

	safety, err := evaluateSafety(actionCtx, client)
	if err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			URL:     target.URL,
			Title:   target.Title,
			Message: err.Error(),
		}, err
	}
	if safety.Outcome != "" && safety.Outcome != BrowserOutcomeOpened {
		artifactID := w.captureScreenshot(actionCtx, client, job.ID)
		return &BrowserActionResult{
			Outcome:              safety.Outcome,
			URL:                  safety.URL,
			Title:                safety.Title,
			Message:              safety.Message,
			ScreenshotArtifactID: artifactID,
		}, nil
	}

	if !accept {
		artifactID := w.captureScreenshot(actionCtx, client, job.ID)
		return &BrowserActionResult{
			Outcome:              BrowserOutcomeOpened,
			URL:                  safety.URL,
			Title:                safety.Title,
			Message:              "Job page opened in worker browser",
			ScreenshotArtifactID: artifactID,
		}, nil
	}

	clickResult, err := clickAccept(actionCtx, client, selector)
	if err != nil {
		artifactID := w.captureScreenshot(actionCtx, client, job.ID)
		return &BrowserActionResult{
			Outcome:              BrowserOutcomeBrowserFailure,
			URL:                  safety.URL,
			Title:                safety.Title,
			Message:              err.Error(),
			ScreenshotArtifactID: artifactID,
		}, err
	}
	if !clickResult.Clicked {
		artifactID := w.captureScreenshot(actionCtx, client, job.ID)
		return &BrowserActionResult{
			Outcome:              BrowserOutcomeUnexpectedPageState,
			URL:                  clickResult.URL,
			Title:                clickResult.Title,
			Message:              clickResult.Message,
			ScreenshotArtifactID: artifactID,
		}, nil
	}

	acceptedURL, ok := waitForWorkbenchURL(actionCtx, client, job.ID)
	if ok {
		artifactID := w.captureScreenshot(actionCtx, client, job.ID)
		return &BrowserActionResult{
			Outcome:              BrowserOutcomeAccepted,
			URL:                  acceptedURL,
			Title:                clickResult.Title,
			Message:              "Accept click succeeded",
			ScreenshotArtifactID: artifactID,
		}, nil
	}

	postClickSafety, err := evaluateSafety(actionCtx, client)
	if err != nil {
		artifactID := w.captureScreenshot(actionCtx, client, job.ID)
		return &BrowserActionResult{
			Outcome:              BrowserOutcomeBrowserFailure,
			URL:                  clickResult.URL,
			Title:                clickResult.Title,
			Message:              err.Error(),
			ScreenshotArtifactID: artifactID,
		}, err
	}
	if postClickSafety.Outcome != "" && postClickSafety.Outcome != BrowserOutcomeOpened {
		artifactID := w.captureScreenshot(actionCtx, client, job.ID)
		return &BrowserActionResult{
			Outcome:              postClickSafety.Outcome,
			URL:                  postClickSafety.URL,
			Title:                postClickSafety.Title,
			Message:              postClickSafety.Message,
			ScreenshotArtifactID: artifactID,
		}, nil
	}

	artifactID := w.captureScreenshot(actionCtx, client, job.ID)
	return &BrowserActionResult{
		Outcome:              BrowserOutcomeTimeout,
		URL:                  postClickSafety.URL,
		Title:                postClickSafety.Title,
		Message:              "Timed out waiting for Gengo workbench after accept click",
		ScreenshotArtifactID: artifactID,
	}, nil
}

func (w *BrowserWorker) captureScreenshot(ctx context.Context, client *cdpClient, jobID string) string {
	var response struct {
		Data string `json:"data"`
	}
	if err := client.Call(ctx, "Page.captureScreenshot", map[string]interface{}{
		"format":      "png",
		"fromSurface": true,
	}, &response); err != nil {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(response.Data)
	if err != nil {
		return ""
	}

	userDir := filepath.Join(w.config.ArtifactRoot, w.userID.String())
	if err := os.MkdirAll(userDir, 0o700); err != nil {
		return ""
	}
	artifactID := safeArtifactName(jobID)
	path := filepath.Join(userDir, artifactID)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return ""
	}
	return artifactID
}

func (w *BrowserWorker) isRunningLocked() bool {
	if w.cmd == nil || w.cmd.Process == nil {
		return false
	}
	if w.cmd.ProcessState != nil && w.cmd.ProcessState.Exited() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return w.cmd.Process.Signal(syscall.Signal(0)) == nil
}

func (w *BrowserWorker) stopLocked() error {
	if w.cmd == nil || w.cmd.Process == nil {
		w.devtools = nil
		return nil
	}
	if w.cmd.ProcessState == nil || !w.cmd.ProcessState.Exited() {
		_ = w.cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- w.cmd.Wait() }()
		select {
		case <-time.After(3 * time.Second):
			_ = w.cmd.Process.Kill()
			<-done
		case <-done:
		}
	}
	w.cmd = nil
	w.devtools = nil
	return nil
}

type cdpClient struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	nextID int64
}

func newCDPClient(ctx context.Context, wsURL string) (*cdpClient, error) {
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("connect DevTools websocket: %w", err)
	}
	return &cdpClient{conn: conn}, nil
}

func (c *cdpClient) Close() error {
	return c.conn.Close()
}

func (c *cdpClient) Call(ctx context.Context, method string, params map[string]interface{}, out interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	id := c.nextID
	payload := map[string]interface{}{
		"id":     id,
		"method": method,
	}
	if params != nil {
		payload["params"] = params
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetWriteDeadline(deadline)
		_ = c.conn.SetReadDeadline(deadline)
	}
	if err := c.conn.WriteJSON(payload); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var msg map[string]json.RawMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			return err
		}
		var responseID int64
		if rawID, ok := msg["id"]; ok {
			_ = json.Unmarshal(rawID, &responseID)
		}
		if responseID != id {
			continue
		}
		if rawErr, ok := msg["error"]; ok {
			return fmt.Errorf("DevTools %s failed: %s", method, string(rawErr))
		}
		if out != nil {
			rawResult, ok := msg["result"]
			if !ok {
				return errors.New("DevTools response missing result")
			}
			if err := json.Unmarshal(rawResult, out); err != nil {
				return err
			}
		}
		return nil
	}
}

type runtimeEvaluateResponse struct {
	Result struct {
		Type  string          `json:"type"`
		Value json.RawMessage `json:"value"`
	} `json:"result"`
}

func evaluateRuntime(ctx context.Context, client *cdpClient, expression string, out interface{}) error {
	var response runtimeEvaluateResponse
	if err := client.Call(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
	}, &response); err != nil {
		return err
	}
	if len(response.Result.Value) == 0 {
		return errors.New("runtime evaluation returned no value")
	}
	return json.Unmarshal(response.Result.Value, out)
}

type safetyEvaluation struct {
	Outcome BrowserActionOutcome `json:"outcome"`
	URL     string               `json:"url"`
	Title   string               `json:"title"`
	Message string               `json:"message"`
}

func evaluateSafety(ctx context.Context, client *cdpClient) (*safetyEvaluation, error) {
	var result safetyEvaluation
	err := evaluateRuntime(ctx, client, safetyCheckExpression, &result)
	if err != nil {
		return nil, err
	}
	if result.Outcome == "" {
		result.Outcome = BrowserOutcomeOpened
	}
	return &result, nil
}

type acceptClickEvaluation struct {
	Clicked bool   `json:"clicked"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

func clickAccept(ctx context.Context, client *cdpClient, selector string) (*acceptClickEvaluation, error) {
	expression := strings.ReplaceAll(acceptClickExpression, "__SELECTOR__", strconv.Quote(selector))
	var result acceptClickEvaluation
	err := evaluateRuntime(ctx, client, expression, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func waitForDocumentReady(ctx context.Context, client *cdpClient) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		var readyState string
		if err := evaluateRuntime(ctx, client, `document.readyState`, &readyState); err == nil {
			if readyState == "complete" || readyState == "interactive" {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func waitForWorkbenchURL(ctx context.Context, client *cdpClient, jobID string) (string, bool) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	pattern := regexp.MustCompile(`/t/workbench/` + regexp.QuoteMeta(jobID) + `(?:[/?#]|$)`)
	for {
		var currentURL string
		_ = evaluateRuntime(ctx, client, `location.href`, &currentURL)
		if pattern.MatchString(currentURL) {
			return currentURL, true
		}
		select {
		case <-ctx.Done():
			return currentURL, false
		case <-ticker.C:
		}
	}
}

func getEnvDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func defaultBrowserHeadless() bool {
	return strings.TrimSpace(os.Getenv("DISPLAY")) == "" &&
		strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY")) == ""
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func safeArtifactName(jobID string) string {
	cleanJobID := regexp.MustCompile(`[^A-Za-z0-9._-]+`).ReplaceAllString(jobID, "_")
	if cleanJobID == "" {
		cleanJobID = "job"
	}
	return fmt.Sprintf("%s-%s.png", time.Now().UTC().Format("20060102T150405.000000000"), cleanJobID)
}

func safeArtifactID(artifactID string) bool {
	if artifactID == "" || strings.Contains(artifactID, "/") || strings.Contains(artifactID, "\\") {
		return false
	}
	return regexp.MustCompile(`^[A-Za-z0-9._-]+$`).MatchString(artifactID)
}

const safetyCheckExpression = `(() => {
  const text = (document.body && document.body.innerText || "").toLowerCase();
  const title = (document.title || "").toLowerCase();
  const href = String(location.href || "").toLowerCase();
  const haystack = [text, title, href].join("\n");
  const has = (values) => values.some((value) => haystack.includes(value));
  if (has(["turnstile", "captcha", "recaptcha", "hcaptcha", "cloudflare challenge"])) {
    return { outcome: "blocked_captcha", url: location.href, title: document.title, message: "Captcha or challenge marker detected" };
  }
  if (has(["suspicious login", "verify your identity", "account security", "security verification", "unusual activity"])) {
    return { outcome: "blocked_suspicious_login", url: location.href, title: document.title, message: "Suspicious login or account security prompt detected" };
  }
  if (href.includes("/login") || href.includes("/signin") || has(["sign in to continue", "log in to continue"])) {
    return { outcome: "blocked_suspicious_login", url: location.href, title: document.title, message: "Worker browser was redirected to login during action flow" };
  }
  return { outcome: "opened", url: location.href, title: document.title, message: "No hard-stop markers detected" };
})()`

const acceptClickExpression = `(() => {
  const selector = __SELECTOR__;
  const visible = (el) => {
    if (!el) return false;
    const style = window.getComputedStyle(el);
    const rect = el.getBoundingClientRect();
    return style.visibility !== "hidden" && style.display !== "none" && rect.width > 0 && rect.height > 0;
  };
  const textOf = (el) => ((el.innerText || el.value || el.getAttribute("aria-label") || el.textContent || "") + "").trim().toLowerCase();
  let candidate = null;
  if (selector && selector.startsWith("text=")) {
    const expected = selector.slice("text=".length).trim().toLowerCase();
    if (expected) {
      candidate = Array.from(document.querySelectorAll("button,input[type='submit'],a,[role='button']"))
        .find((el) => visible(el) && textOf(el) === expected);
    }
  } else if (selector) {
    try {
      candidate = document.querySelector(selector);
    } catch (_) {
      return { clicked: false, url: location.href, title: document.title, message: "Configured Accept selector is not valid CSS or text=..." };
    }
  }
  if (!candidate) {
    candidate = Array.from(document.querySelectorAll("button,input[type='submit'],a,[role='button']"))
      .find((el) => visible(el) && /^accept(?:\s|$)/i.test(textOf(el)));
  }
  if (!candidate) {
    return { clicked: false, url: location.href, title: document.title, message: "Expected visible Accept control was not found" };
  }
  candidate.click();
  return { clicked: true, url: location.href, title: document.title, message: "Clicked Accept control" };
})()`
