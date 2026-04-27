package handlers

import (
	"encoding/json"
	"log"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/watcher"
)

// WatcherHandler handles watcher control endpoints
type WatcherHandler struct {
	manager *watcher.UserWatcherManager
	db      database.Database
}

// NewWatcherHandler creates a new watcher handler
func NewWatcherHandler(manager *watcher.UserWatcherManager, db database.Database) *WatcherHandler {
	return &WatcherHandler{
		manager: manager,
		db:      db,
	}
}

// GetConfig returns the user's watcher configuration
func (h *WatcherHandler) GetConfig(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.getConfigLogic)(c)
}

// getConfigLogic contains the actual GetConfig logic after auth is verified
func (h *WatcherHandler) getConfigLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	if err := watcher.EnsureUserResources(h.db, userUUID); err != nil {
		log.Printf("Failed to prepare watcher config for %s: %v", userUUID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to prepare watcher configuration",
			"code":  "CONFIG_CREATE_FAILED",
		})
	}

	var config models.WatcherConfig
	if err := h.db.Where("user_id = ?", userUUID).First(&config).Error; err != nil {
		log.Printf("Failed to load watcher config: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load watcher configuration",
			"code":  "CONFIG_LOAD_FAILED",
		})
	}

	return c.JSON(configToResponse(&config))
}

// UpdateConfigRequest represents config update input
type UpdateConfigRequest struct {
	RSSFeedURL            string   `json:"rss_feed_url,omitempty"`
	WebSocketEnabled      *bool    `json:"websocket_enabled,omitempty"`
	GengoUserID           string   `json:"gengo_user_id,omitempty"`
	MinReward             *float64 `json:"min_reward,omitempty"`
	MaxReward             *float64 `json:"max_reward,omitempty"`
	IncludedLanguagePairs []string `json:"included_language_pairs,omitempty"`
	EnableDesktopNotifs   *bool    `json:"enable_desktop_notifications,omitempty"`
	EnableSoundNotifs     *bool    `json:"enable_sound_notifications,omitempty"`
	EnableEmailNotifs     *bool    `json:"enable_email_notifications,omitempty"`
	NotificationEmail     string   `json:"notification_email,omitempty"`
	AutoAcceptEnabled     *bool    `json:"auto_accept_enabled,omitempty"`
	AutoAcceptMinReward   *float64 `json:"auto_accept_min_reward,omitempty"`
	AutoAcceptMaxReward   *float64 `json:"auto_accept_max_reward,omitempty"`
}

type IngestJobRequest struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Reward    float64 `json:"reward"`
	URL       string  `json:"url"`
	Source    string  `json:"source,omitempty"`
	Currency  string  `json:"currency,omitempty"`
	Timestamp float64 `json:"timestamp,omitempty"`
	LangPair  string  `json:"lang_pair,omitempty"`
	WordCount int     `json:"word_count,omitempty"`
}

type SyncBrowserStateRequest struct {
	CurrentURL          *string `json:"current_url,omitempty"`
	CurrentTitle        *string `json:"current_title,omitempty"`
	CurrentActionStep   *string `json:"current_action_step,omitempty"`
	CurrentJobID        *string `json:"current_job_id,omitempty"`
	FrontendURL         *string `json:"frontend_url,omitempty"`
	FrontendTitle       *string `json:"frontend_title,omitempty"`
	LoggedInState       *string `json:"logged_in_state,omitempty"`
	BrowserProcessAlive *bool   `json:"browser_process_alive,omitempty"`
	DevToolsConnected   *bool   `json:"devtools_connected,omitempty"`
}

func normalizeIngestJobRequest(req *IngestJobRequest) string {
	if req == nil {
		return "Invalid request body"
	}
	if strings.TrimSpace(req.ID) == "" {
		return "id is required"
	}
	if strings.TrimSpace(req.Title) == "" {
		return "title is required"
	}
	if strings.TrimSpace(req.URL) == "" {
		return "url is required"
	}
	if req.Reward < 0 {
		return "reward must be >= 0"
	}
	if strings.TrimSpace(req.Source) == "" {
		req.Source = "external"
	}
	if strings.TrimSpace(req.Currency) == "" {
		req.Currency = "USD"
	}
	return ""
}

func trimMax(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func validatePublicBrowserURL(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}

	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" ||
		host == "localhost" ||
		host == "::1" ||
		strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") {
		return false
	}

	if ip := net.ParseIP(host); ip != nil {
		return !(ip.IsLoopback() ||
			ip.IsPrivate() ||
			ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() ||
			ip.IsUnspecified())
	}

	return true
}

func validateFrontendURL(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	if parsed.User != nil {
		return false
	}
	return parsed.Scheme == "https" || parsed.Scheme == "http"
}

func buildBrowserStateUpdates(req SyncBrowserStateRequest, now time.Time) (map[string]interface{}, string) {
	updates := map[string]interface{}{
		"last_activity": now.UTC(),
	}

	hasBrowserState := req.CurrentURL != nil ||
		req.CurrentTitle != nil ||
		req.CurrentActionStep != nil ||
		req.CurrentJobID != nil ||
		req.LoggedInState != nil ||
		req.BrowserProcessAlive != nil ||
		req.DevToolsConnected != nil
	if hasBrowserState {
		updates["browser_status"] = watcher.BrowserStatusDashboard
		updates["action_status"] = watcher.ActionStatusIdle
		updates["last_browser_heartbeat_at"] = now.UTC()
	}

	if req.CurrentURL != nil {
		currentURL := strings.TrimSpace(*req.CurrentURL)
		if currentURL == "" {
			updates["current_url"] = ""
		} else {
			if !validatePublicBrowserURL(currentURL) {
				return nil, "current_url must be a safe public HTTP(S) URL"
			}
			updates["current_url"] = currentURL
		}
	}

	if req.CurrentTitle != nil {
		updates["current_title"] = trimMax(*req.CurrentTitle, 512)
	}
	if req.CurrentActionStep != nil {
		updates["current_action_step"] = trimMax(*req.CurrentActionStep, 120)
	}
	if req.CurrentJobID != nil {
		updates["current_job_id"] = trimMax(*req.CurrentJobID, 120)
	}
	if req.FrontendURL != nil {
		frontendURL := strings.TrimSpace(*req.FrontendURL)
		if frontendURL == "" {
			updates["frontend_url"] = ""
		} else {
			if !validateFrontendURL(frontendURL) {
				return nil, "frontend_url must be an HTTP(S) URL without credentials"
			}
			updates["frontend_url"] = frontendURL
		}
		updates["frontend_last_seen_at"] = now.UTC()
	}
	if req.FrontendTitle != nil {
		updates["frontend_title"] = trimMax(*req.FrontendTitle, 512)
		if _, ok := updates["frontend_last_seen_at"]; !ok {
			updates["frontend_last_seen_at"] = now.UTC()
		}
	}
	if req.LoggedInState != nil {
		loggedInState := trimMax(*req.LoggedInState, 32)
		switch loggedInState {
		case "", "unknown", "logged_in", "logged_out":
			if loggedInState == "" {
				loggedInState = "unknown"
			}
			updates["logged_in_state"] = loggedInState
		default:
			return nil, "logged_in_state must be unknown, logged_in, or logged_out"
		}
	}
	if req.BrowserProcessAlive != nil {
		updates["browser_process_alive"] = *req.BrowserProcessAlive
	}
	if req.DevToolsConnected != nil {
		updates["dev_tools_connected"] = *req.DevToolsConnected
	}

	return updates, ""
}

// UpdateConfig updates the user's watcher configuration
func (h *WatcherHandler) UpdateConfig(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.updateConfigLogic)(c)
}

func (h *WatcherHandler) IngestJob(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.ingestJobLogic)(c)
}

func (h *WatcherHandler) SyncBrowserState(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.syncBrowserStateLogic)(c)
}

func (h *WatcherHandler) RestartBrowser(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.restartBrowserLogic)(c)
}

func (h *WatcherHandler) CaptureBrowserScreenshot(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.captureBrowserScreenshotLogic)(c)
}

func (h *WatcherHandler) GetArtifact(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.getArtifactLogic)(c)
}

// updateConfigLogic contains the actual UpdateConfig logic after auth is verified
func (h *WatcherHandler) updateConfigLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	var req UpdateConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Load existing config
	var config models.WatcherConfig
	if err := h.db.Where("user_id = ?", userUUID).First(&config).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Watcher config not found",
			"code":  "CONFIG_NOT_FOUND",
		})
	}

	// Apply partial updates using helper
	updates := ApplyPartialUpdate(req)

	// Special handling for IncludedLanguagePairs (JSON field)
	if req.IncludedLanguagePairs != nil {
		jsonPairs, _ := json.Marshal(req.IncludedLanguagePairs)
		updates["included_language_pairs"] = string(jsonPairs)
	}

	// Apply updates to database
	if err := h.db.Model(&config).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update config",
			"code":  "UPDATE_ERROR",
		})
	}

	// Reload config
	if err := h.db.Where("user_id = ?", userUUID).First(&config).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reload config",
			"code":  "RELOAD_ERROR",
		})
	}

	return c.JSON(configToResponse(&config))
}

func (h *WatcherHandler) ingestJobLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	if h.manager == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Watcher manager unavailable",
			"code":  "WATCHER_UNAVAILABLE",
		})
	}

	var req IngestJobRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	if msg := normalizeIngestJobRequest(&req); msg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": msg,
			"code":  "INVALID_REQUEST",
		})
	}

	job := watcher.Job{
		ID:        req.ID,
		Title:     req.Title,
		Reward:    req.Reward,
		URL:       req.URL,
		Source:    req.Source,
		Currency:  req.Currency,
		Timestamp: req.Timestamp,
		LangPair:  req.LangPair,
		WordCount: req.WordCount,
	}

	if err := h.manager.ProcessExternalJob(c.Context(), userUUID, job); err != nil {
		log.Printf("Failed to process external watcher job: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process watcher job",
			"code":  "INGEST_FAILED",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"status": "accepted",
		"job_id": req.ID,
	})
}

func (h *WatcherHandler) syncBrowserStateLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	if h.manager == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Watcher manager unavailable",
			"code":  "WATCHER_UNAVAILABLE",
		})
	}

	var req SyncBrowserStateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	updates, msg := buildBrowserStateUpdates(req, time.Now())
	if msg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": msg,
			"code":  "INVALID_REQUEST",
		})
	}

	if err := h.manager.UpdateBrowserState(c.Context(), userUUID, updates); err != nil {
		log.Printf("Failed to sync browser state: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to sync browser state",
			"code":  "BROWSER_STATE_SYNC_FAILED",
		})
	}

	return c.JSON(fiber.Map{
		"status": "synced",
	})
}

func (h *WatcherHandler) restartBrowserLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	if h.manager == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Watcher manager unavailable",
			"code":  "WATCHER_UNAVAILABLE",
		})
	}
	if err := h.manager.RestartBrowser(c.Context(), userUUID); err != nil {
		log.Printf("Failed to restart worker browser: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to restart worker browser",
			"code":  "BROWSER_RESTART_FAILED",
		})
	}
	return c.JSON(fiber.Map{
		"status": "restarted",
	})
}

func (h *WatcherHandler) captureBrowserScreenshotLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	if h.manager == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "Watcher manager unavailable",
			"code":  "WATCHER_UNAVAILABLE",
		})
	}
	result, err := h.manager.CaptureBrowserScreenshot(c.Context(), userUUID)
	if err != nil {
		log.Printf("Failed to capture worker browser screenshot: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to capture worker browser screenshot",
			"code":  "BROWSER_SCREENSHOT_FAILED",
		})
	}
	response := fiber.Map{
		"status": "captured",
	}
	if result != nil {
		response["url"] = result.URL
		response["title"] = result.Title
		response["screenshot_artifact_id"] = result.ScreenshotArtifactID
		if result.ScreenshotArtifactID != "" {
			response["screenshot_url"] = "/api/v1/watcher/artifacts/" + result.ScreenshotArtifactID
		}
	}
	return c.JSON(response)
}

// GetState returns the user's watcher state
func (h *WatcherHandler) GetState(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.getStateLogic)(c)
}

// getStateLogic contains the actual GetState logic after auth is verified
func (h *WatcherHandler) getStateLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	if err := watcher.EnsureUserResources(h.db, userUUID); err != nil {
		log.Printf("Failed to prepare watcher state for %s: %v", userUUID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to prepare watcher state",
			"code":  "STATE_CREATE_FAILED",
		})
	}

	var state models.WatcherState
	if err := h.db.Where("user_id = ?", userUUID).First(&state).Error; err != nil {
		log.Printf("Failed to load watcher state: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load watcher state",
			"code":  "STATE_LOAD_FAILED",
		})
	}

	// Get live status from manager if available
	status, _ := h.manager.GetStatus(userUUID)
	state.WatcherStatus = status

	return c.JSON(stateToResponse(&state))
}

// GetEvents returns the user's recent watcher runtime events.
func (h *WatcherHandler) GetEvents(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.getEventsLogic)(c)
}

func (h *WatcherHandler) getEventsLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	limit := c.QueryInt("limit", 50)

	events, err := h.manager.GetEvents(userUUID, limit)
	if err != nil {
		log.Printf("Failed to load watcher events: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load watcher events",
			"code":  "EVENTS_LOAD_FAILED",
		})
	}

	response := make([]map[string]interface{}, 0, len(events))
	for _, event := range events {
		response = append(response, eventToResponse(&event))
	}

	return c.JSON(fiber.Map{
		"events": response,
	})
}

func (h *WatcherHandler) getArtifactLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	artifactID := strings.TrimSpace(c.Params("artifactID"))
	path, err := watcher.ArtifactPathForUser(userUUID, artifactID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid artifact id",
			"code":  "INVALID_ARTIFACT",
		})
	}
	c.Type("png")
	return c.SendFile(path, false)
}

// StartWatcher starts the user's watcher
func (h *WatcherHandler) StartWatcher(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.startWatcherLogic)(c)
}

// startWatcherLogic contains the actual StartWatcher logic after auth is verified
func (h *WatcherHandler) startWatcherLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	if err := h.manager.StartWatcher(userUUID); err != nil {
		log.Printf("Failed to start watcher: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to start watcher",
			"code":  "START_ERROR",
		})
	}

	// Get current status
	status, _ := h.manager.GetStatus(userUUID)

	return c.JSON(fiber.Map{
		"status": status,
	})
}

// StopWatcher stops the user's watcher
func (h *WatcherHandler) StopWatcher(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.stopWatcherLogic)(c)
}

// stopWatcherLogic contains the actual StopWatcher logic after auth is verified
func (h *WatcherHandler) stopWatcherLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	if err := h.manager.StopWatcher(userUUID); err != nil {
		log.Printf("Failed to stop watcher: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to stop watcher",
			"code":  "STOP_ERROR",
		})
	}

	// Get current status
	status, _ := h.manager.GetStatus(userUUID)

	return c.JSON(fiber.Map{
		"status": status,
	})
}

// configToResponse converts WatcherConfig model to API response
func configToResponse(config *models.WatcherConfig) map[string]interface{} {
	// Parse JSON fields for response
	var languagePairs []string
	_ = json.Unmarshal([]byte(config.IncludedLanguagePairs), &languagePairs)

	return map[string]interface{}{
		"user_id":                      config.UserID.String(),
		"rss_feed_url":                 config.RSSFeedURL,
		"websocket_enabled":            config.WebSocketEnabled,
		"gengo_user_id":                config.GengoUserID,
		"min_reward":                   config.MinReward,
		"max_reward":                   config.MaxReward,
		"included_language_pairs":      languagePairs,
		"enable_desktop_notifications": config.EnableDesktopNotifs,
		"enable_sound_notifications":   config.EnableSoundNotifs,
		"enable_email_notifications":   config.EnableEmailNotifs,
		"notification_email":           config.NotificationEmail,
		"auto_accept_enabled":          config.AutoAcceptEnabled,
		"auto_accept_min_reward":       config.AutoAcceptMinReward,
		"auto_accept_max_reward":       config.AutoAcceptMaxReward,
		"created_at":                   config.CreatedAt,
		"updated_at":                   config.UpdatedAt,
	}
}

// stateToResponse converts WatcherState model to API response
func stateToResponse(state *models.WatcherState) map[string]interface{} {
	response := map[string]interface{}{
		"worker_id":                     state.UserID.String(),
		"user_id":                       state.UserID.String(),
		"watcher_status":                state.WatcherStatus,
		"overall_status":                state.OverallStatus,
		"feed_status":                   state.FeedStatus,
		"browser_status":                state.BrowserStatus,
		"action_status":                 state.ActionStatus,
		"alert_status":                  state.AlertStatus,
		"profile_status":                state.ProfileStatus,
		"current_job_id":                state.CurrentJobID,
		"current_action_step":           state.CurrentActionStep,
		"current_url":                   state.CurrentURL,
		"current_title":                 state.CurrentTitle,
		"frontend_url":                  state.FrontendURL,
		"frontend_title":                state.FrontendTitle,
		"frontend_last_seen_at":         state.FrontendLastSeenAt,
		"logged_in_state":               state.LoggedInState,
		"browser_process_alive":         state.BrowserProcessAlive,
		"devtools_connected":            state.DevToolsConnected,
		"last_rss_poll_started_at":      state.LastRSSPollStartedAt,
		"last_rss_poll_ok_at":           state.LastRSSPollOKAt,
		"rss_consecutive_failures":      state.RSSConsecutiveFailures,
		"last_ws_connect_at":            state.LastWSConnectAt,
		"last_ws_message_at":            state.LastWSMessageAt,
		"last_ws_pong_at":               state.LastWSPongAt,
		"last_ws_close_code":            state.LastWSCloseCode,
		"last_ws_close_reason":          state.LastWSCloseReason,
		"ws_reconnect_count":            state.WSReconnectCount,
		"last_browser_heartbeat_at":     state.LastBrowserHeartbeatAt,
		"last_error":                    state.LastError,
		"last_critical_alert":           state.LastCriticalAlert,
		"latest_screenshot_artifact_id": state.LatestScreenshotArtifactID,
		"total_jobs_found":              state.TotalJobsFound,
		"total_jobs_accepted":           state.TotalJobsAccepted,
		"total_earnings":                state.TotalEarnings,
		"last_activity":                 state.LastActivity,
		"updated_at":                    state.UpdatedAt,
	}
	if state.LatestScreenshotArtifactID != "" {
		response["latest_screenshot_url"] = "/api/v1/watcher/artifacts/" + state.LatestScreenshotArtifactID
	}
	return response
}

func eventToResponse(event *models.WatcherEvent) map[string]interface{} {
	data := map[string]interface{}{}
	if strings.TrimSpace(event.Data) != "" {
		_ = json.Unmarshal([]byte(event.Data), &data)
	}
	return map[string]interface{}{
		"id":          event.ID.String(),
		"worker_id":   event.WorkerID.String(),
		"user_id":     event.UserID.String(),
		"level":       event.Level,
		"source":      event.Source,
		"type":        event.Type,
		"job_id":      event.JobID,
		"message":     event.Message,
		"data":        data,
		"occurred_at": event.OccurredAt,
	}
}
