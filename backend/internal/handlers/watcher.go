package handlers

import (
	"encoding/json"
	"log"
	"strings"

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
	var config models.WatcherConfig
	if err := h.db.Where("user_id = ?", userUUID).First(&config).Error; err != nil {
		// #017 fix - Lazy initialization of watcher config
		// Explicitly generate ID for composite primary key tables
		config = models.WatcherConfig{
			Base:                  models.Base{ID: uuid.New()},
			UserID:                userUUID,
			IncludedLanguagePairs: "[]", // Valid JSON array for jsonb column
		}
		if createErr := h.db.Create(&config).Error; createErr != nil {
			log.Printf("Failed to create watcher config: %v", createErr)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create watcher configuration",
				"code":  "CONFIG_CREATE_FAILED",
			})
		}
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

// UpdateConfig updates the user's watcher configuration
func (h *WatcherHandler) UpdateConfig(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.updateConfigLogic)(c)
}

func (h *WatcherHandler) IngestJob(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.ingestJobLogic)(c)
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

func (h *WatcherHandler) ensureWatcherResources(userUUID uuid.UUID) error {
	var config models.WatcherConfig
	if err := h.db.Where("user_id = ?", userUUID).First(&config).Error; err != nil {
		config = models.WatcherConfig{
			Base:                  models.Base{ID: uuid.New()},
			UserID:                userUUID,
			IncludedLanguagePairs: "[]",
		}
		if createErr := h.db.Create(&config).Error; createErr != nil {
			return createErr
		}
	}

	var state models.WatcherState
	if err := h.db.Where("user_id = ?", userUUID).First(&state).Error; err != nil {
		state = models.WatcherState{
			UserID:           userUUID,
			WatcherStatus:    "stopped",
			LastSeenJobIDs:   "[]",
			RecentJobHistory: "[]",
		}
		if createErr := h.db.Create(&state).Error; createErr != nil {
			return createErr
		}
	}

	return nil
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

	if err := h.ensureWatcherResources(userUUID); err != nil {
		log.Printf("Failed to initialize watcher resources: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to initialize watcher resources",
			"code":  "WATCHER_INIT_FAILED",
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

// GetState returns the user's watcher state
func (h *WatcherHandler) GetState(c *fiber.Ctx) error {
	return middleware.RequireAuth(h.getStateLogic)(c)
}

// getStateLogic contains the actual GetState logic after auth is verified
func (h *WatcherHandler) getStateLogic(c *fiber.Ctx, userUUID uuid.UUID) error {
	var state models.WatcherState
	if err := h.db.Where("user_id = ?", userUUID).First(&state).Error; err != nil {
		// #017 fix - Lazy initialization of watcher state
		state = models.WatcherState{
			UserID:           userUUID,
			WatcherStatus:    "stopped",
			LastSeenJobIDs:   "[]", // Valid JSON array for jsonb column
			RecentJobHistory: "[]", // Valid JSON array for jsonb column
		}
		if createErr := h.db.Create(&state).Error; createErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create watcher state",
				"code":  "STATE_CREATE_FAILED",
			})
		}
	}

	// Get live status from manager if available
	status, _ := h.manager.GetStatus(userUUID)
	state.WatcherStatus = status

	return c.JSON(stateToResponse(&state))
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
	return map[string]interface{}{
		"user_id":             state.UserID.String(),
		"watcher_status":      state.WatcherStatus,
		"total_jobs_found":    state.TotalJobsFound,
		"total_jobs_accepted": state.TotalJobsAccepted,
		"total_earnings":      state.TotalEarnings,
		"last_activity":       state.LastActivity,
		"updated_at":          state.UpdatedAt,
	}
}
