package watcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	defaultFirefoxRDPDebugURL = "ws://127.0.0.1:6000"
	gengoRealtimePath         = "/t/jobs/status/available/realtime"
	gengoDashboardPath        = "/t/dashboard"
)

type FirefoxRDPBrowserWorker struct {
	userID    uuid.UUID
	config    BrowserWorkerConfig
	validator *URLValidator
}

func NewFirefoxRDPBrowserWorker(userID uuid.UUID, config BrowserWorkerConfig) *FirefoxRDPBrowserWorker {
	if config.DebugURL == "" {
		config.DebugURL = defaultFirefoxRDPDebugURL
	}
	if config.ActionTimeout <= 0 {
		config.ActionTimeout = defaultBrowserActionTimeout
	}
	return &FirefoxRDPBrowserWorker{
		userID:    userID,
		config:    config,
		validator: NewURLValidator(),
	}
}

func looksLikeFirefoxRDPDebugURL(debugURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(debugURL))
	if err != nil {
		return false
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return false
	}
	return strings.TrimRight(parsed.Path, "/") != "/session"
}

func (w *FirefoxRDPBrowserWorker) Start(ctx context.Context) error {
	client, err := w.openClient(ctx)
	if err != nil {
		return err
	}
	return client.Close()
}

func (w *FirefoxRDPBrowserWorker) Stop(context.Context) error {
	return nil
}

func (w *FirefoxRDPBrowserWorker) Restart(ctx context.Context) error {
	return w.Start(ctx)
}

func (w *FirefoxRDPBrowserWorker) Health(ctx context.Context) error {
	return w.Start(ctx)
}

func (w *FirefoxRDPBrowserWorker) CaptureScreenshot(ctx context.Context) (*BrowserActionResult, error) {
	actionCtx, cancel := context.WithTimeout(ctx, w.config.ActionTimeout)
	defer cancel()

	client, err := w.openClient(actionCtx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	tab, err := client.gengoTab(actionCtx, gengoRealtimePath, gengoDashboardPath)
	if err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			Message: err.Error(),
		}, err
	}
	state, err := client.pageState(actionCtx, tab)
	if err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			URL:     tab.URL,
			Title:   tab.Title,
			Message: err.Error(),
		}, err
	}
	return &BrowserActionResult{
		Outcome: BrowserOutcomeOpened,
		URL:     state.URL,
		Title:   state.Title,
		Message: "Firefox DevTools session attached; screenshot capture is unavailable through RDP",
	}, nil
}

func (w *FirefoxRDPBrowserWorker) OpenJob(ctx context.Context, job Job) (*BrowserActionResult, error) {
	if err := w.validator.Validate(job.URL); err != nil {
		return nil, fmt.Errorf("job URL validation failed: %w", err)
	}
	return w.withJobPage(ctx, job, false, "")
}

func (w *FirefoxRDPBrowserWorker) AcceptJob(ctx context.Context, job Job, selector string) (*BrowserActionResult, error) {
	if selector == "" {
		selector = defaultAcceptSelector
	}
	if err := w.validator.Validate(job.URL); err != nil {
		return nil, fmt.Errorf("job URL validation failed: %w", err)
	}
	return w.withJobPage(ctx, job, true, selector)
}

func (w *FirefoxRDPBrowserWorker) withJobPage(
	ctx context.Context,
	job Job,
	accept bool,
	selector string,
) (*BrowserActionResult, error) {
	actionCtx, cancel := context.WithTimeout(ctx, w.config.ActionTimeout)
	defer cancel()

	client, err := w.openClient(actionCtx)
	if err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			Message: err.Error(),
		}, err
	}
	defer client.Close()

	openerTab, err := client.gengoTab(actionCtx, gengoRealtimePath, gengoDashboardPath)
	if err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			Message: err.Error(),
		}, err
	}
	if err := client.openJobWindow(actionCtx, openerTab, job.URL); err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			URL:     openerTab.URL,
			Title:   openerTab.Title,
			Message: err.Error(),
		}, err
	}
	tab, ok := client.waitForAnyGengoURLContains(actionCtx, "/t/jobs/details/")
	if !ok {
		tab = openerTab
	}

	safety, err := client.safety(actionCtx, tab)
	if err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			URL:     tab.URL,
			Title:   tab.Title,
			Message: err.Error(),
		}, err
	}
	if safety.Outcome != "" && safety.Outcome != BrowserOutcomeOpened {
		return &BrowserActionResult{
			Outcome: safety.Outcome,
			URL:     safety.URL,
			Title:   safety.Title,
			Message: safety.Message,
		}, nil
	}
	if !accept {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeOpened,
			URL:     safety.URL,
			Title:   safety.Title,
			Message: "Job page opened in Firefox DevTools browser",
		}, nil
	}

	clickResult, err := client.clickAccept(actionCtx, tab, selector)
	if err != nil {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeBrowserFailure,
			URL:     safety.URL,
			Title:   safety.Title,
			Message: err.Error(),
		}, err
	}
	if !clickResult.Clicked {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeUnexpectedPageState,
			URL:     clickResult.URL,
			Title:   clickResult.Title,
			Message: clickResult.Message,
		}, nil
	}
	if acceptedURL, ok := client.waitForWorkbenchURL(actionCtx, tab, job.ID); ok {
		return &BrowserActionResult{
			Outcome: BrowserOutcomeAccepted,
			URL:     acceptedURL,
			Title:   clickResult.Title,
			Message: "Accept click succeeded in Firefox DevTools browser",
		}, nil
	}
	postClickSafety, err := client.safety(actionCtx, tab)
	if err == nil && postClickSafety.Outcome != "" && postClickSafety.Outcome != BrowserOutcomeOpened {
		return &BrowserActionResult{
			Outcome: postClickSafety.Outcome,
			URL:     postClickSafety.URL,
			Title:   postClickSafety.Title,
			Message: postClickSafety.Message,
		}, nil
	}
	return &BrowserActionResult{
		Outcome: BrowserOutcomeTimeout,
		URL:     clickResult.URL,
		Title:   clickResult.Title,
		Message: "Timed out waiting for Gengo workbench after accept click",
	}, nil
}

func (w *FirefoxRDPBrowserWorker) openClient(ctx context.Context) (*firefoxRDPClient, error) {
	debugURL := strings.TrimSpace(w.config.DebugURL)
	if debugURL == "" {
		debugURL = defaultFirefoxRDPDebugURL
	}
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.DialContext(ctx, debugURL, http.Header{})
	if err != nil {
		return nil, fmt.Errorf("connect Firefox DevTools websocket %s: %w", debugURL, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetReadDeadline(deadline)
	}
	var hello map[string]interface{}
	if err := conn.ReadJSON(&hello); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("read Firefox DevTools hello: %w", err)
	}
	if hello["from"] != "root" {
		_ = conn.Close()
		return nil, errors.New("Firefox DevTools did not send a root hello packet")
	}
	return &firefoxRDPClient{conn: conn}, nil
}

type firefoxRDPClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type firefoxTab struct {
	Actor         string
	TargetActor   string
	ConsoleActor  string
	InnerWindowID interface{}
	URL           string
	Title         string
}

func (c *firefoxRDPClient) Close() error {
	return c.conn.Close()
}

func (c *firefoxRDPClient) request(ctx context.Context, actor string, packetType string, payload map[string]interface{}) (map[string]interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	packet := map[string]interface{}{
		"to":   actor,
		"type": packetType,
	}
	for key, value := range payload {
		packet[key] = value
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetWriteDeadline(deadline)
		_ = c.conn.SetReadDeadline(deadline)
	}
	if err := c.conn.WriteJSON(packet); err != nil {
		return nil, err
	}
	return c.readResponse(ctx, actor, "")
}

func (c *firefoxRDPClient) readResponse(ctx context.Context, actor string, expectedType string) (map[string]interface{}, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		var response map[string]interface{}
		if err := c.conn.ReadJSON(&response); err != nil {
			return nil, err
		}
		if response["from"] != actor {
			continue
		}
		if isFirefoxNotification(response) {
			continue
		}
		if expectedType != "" && response["type"] != expectedType {
			continue
		}
		if rawErr, ok := response["error"]; ok {
			detail := strings.TrimSpace(fmt.Sprint(response["message"]))
			if detail == "" {
				detail = fmt.Sprint(rawErr)
			}
			return nil, errors.New(detail)
		}
		return response, nil
	}
}

func isFirefoxNotification(packet map[string]interface{}) bool {
	switch packet["type"] {
	case "consoleAPICall", "lastPrivateContextExited", "networkEvent", "networkEventUpdate",
		"pageError", "tabDetached", "tabListChanged", "tabNavigated":
		return true
	default:
		return false
	}
}

func (c *firefoxRDPClient) gengoTab(ctx context.Context, preferredFragments ...string) (*firefoxTab, error) {
	response, err := c.request(ctx, "root", "listTabs", nil)
	if err != nil {
		return nil, err
	}
	rawTabs, ok := response["tabs"].([]interface{})
	if !ok {
		return nil, errors.New("Firefox DevTools did not return a tab list")
	}
	var candidates []map[string]interface{}
	for _, rawTab := range rawTabs {
		tab, ok := rawTab.(map[string]interface{})
		if !ok {
			continue
		}
		if isGengoURL(stringValue(tab["url"])) {
			candidates = append(candidates, tab)
		}
	}
	activeCandidates := make([]map[string]interface{}, 0, len(candidates))
	for _, tab := range candidates {
		if !isGengoLoginURL(stringValue(tab["url"])) {
			activeCandidates = append(activeCandidates, tab)
		}
	}
	if len(activeCandidates) > 0 {
		candidates = activeCandidates
	}
	for _, fragment := range preferredFragments {
		for _, tab := range candidates {
			if strings.Contains(stringValue(tab["url"]), fragment) {
				return c.resolveTab(ctx, tab)
			}
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("No open gengo.com tab found in Firefox DevTools session")
	}
	return c.resolveTab(ctx, candidates[0])
}

func (c *firefoxRDPClient) resolveTab(ctx context.Context, tab map[string]interface{}) (*firefoxTab, error) {
	resolved := &firefoxTab{
		Actor:        stringValue(tab["actor"]),
		ConsoleActor: stringValue(tab["consoleActor"]),
		URL:          stringValue(tab["url"]),
		Title:        stringValue(tab["title"]),
	}
	if resolved.ConsoleActor != "" {
		return resolved, nil
	}
	if resolved.Actor == "" {
		return resolved, nil
	}
	response, err := c.request(ctx, resolved.Actor, "getTarget", nil)
	if err != nil {
		return nil, err
	}
	frame, _ := response["frame"].(map[string]interface{})
	if frame == nil {
		return resolved, nil
	}
	resolved.TargetActor = stringValue(frame["actor"])
	resolved.ConsoleActor = stringValue(frame["consoleActor"])
	resolved.InnerWindowID = frame["innerWindowId"]
	if value := stringValue(frame["url"]); value != "" {
		resolved.URL = value
	}
	if value := stringValue(frame["title"]); value != "" {
		resolved.Title = value
	}
	return resolved, nil
}

func (c *firefoxRDPClient) evaluateJSON(ctx context.Context, tab *firefoxTab, expression string, out interface{}) error {
	actor := strings.TrimSpace(tab.ConsoleActor)
	if actor == "" {
		return errors.New("Firefox DevTools did not expose a console actor for the gengo.com tab")
	}
	payload := map[string]interface{}{
		"text":                                  expression,
		"frameActor":                            nil,
		"url":                                   nil,
		"selectedNodeActor":                     nil,
		"selectedObjectActor":                   nil,
		"innerWindowID":                         tab.InnerWindowID,
		"mapped":                                nil,
		"eager":                                 false,
		"disableBreaks":                         true,
		"preferConsoleCommandsOverLocalSymbols": false,
		"evalInTracer":                          false,
	}
	if _, err := c.request(ctx, actor, "evaluateJSAsync", payload); err != nil {
		return err
	}
	response, err := c.readResponse(ctx, actor, "evaluationResult")
	if err != nil {
		return err
	}
	value, err := firefoxEvaluationValue(response)
	if err != nil {
		return err
	}
	text, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected serialized JSON string from Firefox evaluation, got %T", value)
	}
	return json.Unmarshal([]byte(text), out)
}

func firefoxEvaluationValue(packet map[string]interface{}) (interface{}, error) {
	if hasException, _ := packet["hasException"].(bool); hasException {
		return nil, errors.New(fmt.Sprint(packet["exceptionMessage"]))
	}
	result := packet["result"]
	if object, ok := result.(map[string]interface{}); ok {
		switch object["type"] {
		case "longString":
			initial := fmt.Sprint(object["initial"])
			if length, ok := numericInt(object["length"]); ok && len(initial) < length {
				return nil, errors.New("Firefox DevTools returned a truncated long string result")
			}
			return initial, nil
		case "string", "number", "boolean", "bigint":
			return object["value"], nil
		case "null", "undefined":
			return nil, nil
		}
	}
	return result, nil
}

func (c *firefoxRDPClient) openJobWindow(ctx context.Context, tab *firefoxTab, rawURL string) error {
	var result struct {
		Queued bool   `json:"queued"`
		URL    string `json:"url"`
	}
	if err := c.evaluateJSON(ctx, tab, firefoxOpenJobWindowExpression(rawURL), &result); err != nil {
		return err
	}
	if !result.Queued {
		return errors.New("Firefox DevTools did not queue job-window navigation")
	}
	return nil
}

func (c *firefoxRDPClient) waitForAnyGengoURLContains(ctx context.Context, fragment string) (*firefoxTab, bool) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		tab, err := c.gengoTabByURLFragment(ctx, fragment)
		if err == nil && tab != nil {
			return tab, true
		}
		select {
		case <-ctx.Done():
			return &firefoxTab{}, false
		case <-ticker.C:
		}
	}
}

func (c *firefoxRDPClient) gengoTabByURLFragment(ctx context.Context, fragment string) (*firefoxTab, error) {
	response, err := c.request(ctx, "root", "listTabs", nil)
	if err != nil {
		return nil, err
	}
	rawTabs, _ := response["tabs"].([]interface{})
	for _, rawTab := range rawTabs {
		tab, ok := rawTab.(map[string]interface{})
		if !ok {
			continue
		}
		rawURL := stringValue(tab["url"])
		if !isGengoURL(rawURL) || !strings.Contains(rawURL, fragment) {
			continue
		}
		return c.resolveTab(ctx, tab)
	}
	return nil, errors.New("Firefox gengo.com tab with requested URL fragment not found")
}

func (c *firefoxRDPClient) pageState(ctx context.Context, tab *firefoxTab) (*struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}, error) {
	var state struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	if err := c.evaluateJSON(ctx, tab, firefoxPageStateExpression, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (c *firefoxRDPClient) safety(ctx context.Context, tab *firefoxTab) (*safetyEvaluation, error) {
	var result safetyEvaluation
	if err := c.evaluateJSON(ctx, tab, firefoxSafetyExpression, &result); err != nil {
		return nil, err
	}
	if result.Outcome == "" {
		result.Outcome = BrowserOutcomeOpened
	}
	return &result, nil
}

func (c *firefoxRDPClient) clickAccept(ctx context.Context, tab *firefoxTab, selector string) (*acceptClickEvaluation, error) {
	var result acceptClickEvaluation
	expression := strings.ReplaceAll(firefoxAcceptClickExpression, "__SELECTOR__", strconv.Quote(selector))
	if err := c.evaluateJSON(ctx, tab, expression, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *firefoxRDPClient) waitForWorkbenchURL(ctx context.Context, tab *firefoxTab, jobID string) (string, bool) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		var state struct {
			Href string `json:"href"`
		}
		if err := c.evaluateJSON(ctx, tab, firefoxHrefExpression, &state); err == nil {
			if regexpWorkbenchURL(jobID, state.Href) {
				return state.Href, true
			}
		}
		select {
		case <-ctx.Done():
			return "", false
		case <-ticker.C:
		}
	}
}

func regexpWorkbenchURL(jobID string, rawURL string) bool {
	return regexp.MustCompile(`/t/workbench/` + regexp.QuoteMeta(jobID) + `(?:[/?#]|$)`).MatchString(rawURL)
}

func isGengoURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	return host == "gengo.com" || strings.HasSuffix(host, ".gengo.com")
}

func isGengoLoginURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	path := strings.ToLower(parsed.Path)
	return strings.Contains(path, "/login") || strings.Contains(path, "/signin") || strings.Contains(path, "/auth/form")
}

func numericInt(value interface{}) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
}

func stringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func firefoxOpenJobWindowExpression(rawURL string) string {
	return `(() => {
const targetUrl = ` + strconv.Quote(rawURL) + `;
const openJob = () => {
  const jobWindow = window.open(targetUrl, "GengoWatcher Job Window");
  if (jobWindow) {
    try { jobWindow.opener = null; } catch (_) {}
    try { jobWindow.focus(); } catch (_) {}
  }
};
setTimeout(openJob, 0);
return JSON.stringify({ queued: true, url: targetUrl });
})()`
}

const firefoxPageStateExpression = `JSON.stringify({
url: location.href || "",
title: document.title || ""
})`

const firefoxHrefExpression = `JSON.stringify({ href: location.href || "" })`

const firefoxSafetyExpression = `(() => {
  const text = (document.body && document.body.innerText || "").toLowerCase();
  const title = (document.title || "").toLowerCase();
  const href = String(location.href || "").toLowerCase();
  const haystack = [text, title, href].join("\n");
  const has = (values) => values.some((value) => haystack.includes(value));
  if (has(["turnstile", "captcha", "recaptcha", "hcaptcha", "cloudflare challenge"])) {
    return JSON.stringify({ outcome: "blocked_captcha", url: location.href, title: document.title, message: "Captcha or challenge marker detected" });
  }
  if (has(["suspicious login", "verify your identity", "account security", "security verification", "unusual activity"])) {
    return JSON.stringify({ outcome: "blocked_suspicious_login", url: location.href, title: document.title, message: "Suspicious login or account security prompt detected" });
  }
  if (href.includes("/login") || href.includes("/signin") || has(["sign in to continue", "log in to continue"])) {
    return JSON.stringify({ outcome: "blocked_suspicious_login", url: location.href, title: document.title, message: "Worker browser was redirected to login during action flow" });
  }
  return JSON.stringify({ outcome: "opened", url: location.href, title: document.title, message: "No hard-stop markers detected" });
})()`

const firefoxAcceptClickExpression = `(() => {
  const selector = __SELECTOR__;
  const visible = (el) => {
    if (!el) return false;
    const style = window.getComputedStyle(el);
    const rect = el.getBoundingClientRect();
    return style.visibility !== "hidden" && style.display !== "none" && rect.width > 0 && rect.height > 0;
  };
  const direct = document.querySelector(selector);
  const candidates = direct ? [direct] : Array.from(document.querySelectorAll("button, a, input[type=button], input[type=submit]"));
  const target = candidates.find((el) => {
    const label = String(el.innerText || el.value || el.getAttribute("aria-label") || "").toLowerCase();
    return visible(el) && (label.includes("accept") || label.includes("start"));
  });
  if (!target) {
    return JSON.stringify({ clicked: false, url: location.href, title: document.title, message: "Accept control not found" });
  }
  target.click();
  return JSON.stringify({ clicked: true, url: location.href, title: document.title, message: "Accept control clicked" });
})()`
