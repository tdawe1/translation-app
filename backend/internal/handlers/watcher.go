package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/watcher"
)

// WatcherHandler handles watcher control endpoints
type WatcherHandler struct {
	manager *watcher.UserWatcherManager
}

// NewWatcherHandler creates a new watcher handler
func NewWatcherHandler(manager *watcher.UserWatcherManager) *WatcherHandler {
	return &WatcherHandler{
		manager: manager,
	}
}

// GetConfig returns the user's watcher configuration
func (h *WatcherHandler) GetConfig(c *fiber.Ctx) error {
	userID, ok := GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
			"code":  "NOT_AUTHENTICATED",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
			"code":  "INVALID_USER_ID",
		})
	}

	var config models.WatcherConfig
	if err := models.DB.Where("user_id = ?", userUUID).First(&config).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Watcher config not found",
			"code":  "CONFIG_NOT_FOUND",
		})
	}

	return c.JSON(configToResponse(&config))
}

// UpdateConfigRequest represents config update input
type UpdateConfigRequest struct {
	RSSFeedURL               string   `json:"rss_feed_url,omitempty"`
	WebSocketEnabled         *bool    `json:"websocket_enabled,omitempty"`
	GengoUserID              string   `json:"gengo_user_id,omitempty"`
	MinReward                *float64 `json:"min_reward,omitempty"`
	MaxReward                *float64 `json:"max_reward,omitempty"`
	IncludedLanguagePairs   []string `json:"included_language_pairs,omitempty"`
	EnableDesktopNotifs      *bool    `json:"enable_desktop_notifications,omitempty"`
	EnableSoundNotifs        *bool    `json:"enable_sound_notifications,omitempty"`
	EnableEmailNotifs        *bool    `json:"enable_email_notifications,omitempty"`
	NotificationEmail        string   `json:"notification_email,omitempty"`
	AutoAcceptEnabled        *bool    `json:"auto_accept_enabled,omitempty"`
	AutoAcceptMinReward      *float64 `json:"auto_accept_min_reward,omitempty"`
	AutoAcceptMaxReward      *float64 `json:"auto_accept_max_reward,omitempty"`
}

// UpdateConfig updates the user's watcher configuration
func (h *WatcherHandler) UpdateConfig(c *fiber.Ctx) error {
	userID, ok := GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
			"code":  "NOT_AUTHENTICATED",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
			"code":  "INVALID_USER_ID",
		})
	}

	var req UpdateConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Load existing config
	var config models.WatcherConfig
	if err := models.DB.Where("user_id = ?", userUUID).First(&config).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Watcher config not found",
			"code":  "CONFIG_NOT_FOUND",
		})
	}

	// Update fields that are provided
	updates := make(map[string]interface{})

	if req.RSSFeedURL != "" {
		updates["rss_feed_url"] = req.RSSFeedURL
	}
	if req.WebSocketEnabled != nil {
		updates["websocket_enabled"] = *req.WebSocketEnabled
	}
	if req.GengoUserID != "" {
		updates["gengo_user_id"] = req.GengoUserID
	}
	if req.MinReward != nil {
		updates["min_reward"] = *req.MinReward
	}
	if req.MaxReward != nil {
		updates["max_reward"] = *req.MaxReward
	}
	if req.IncludedLanguagePairs != nil {
		// Convert to JSON string for storage
		updates["included_language_pairs"] = req.IncludedLanguagePairs
	}
	if req.EnableDesktopNotifs != nil {
		updates["enable_desktop_notifs"] = *req.EnableDesktopNotifs
	}
	if req.EnableSoundNotifs != nil {
		updates["enable_sound_notifs"] = *req.EnableSoundNotifs
	}
	if req.EnableEmailNotifs != nil {
		updates["enable_email_notifs"] = *req.EnableEmailNotifs
	}
	if req.NotificationEmail != "" {
		updates["notification_email"] = req.NotificationEmail
	}
	if req.AutoAcceptEnabled != nil {
		updates["auto_accept_enabled"] = *req.AutoAcceptEnabled
	}
	if req.AutoAcceptMinReward != nil {
		updates["auto_accept_min_reward"] = *req.AutoAcceptMinReward
	}
	if req.AutoAcceptMaxReward != nil {
		updates["auto_accept_max_reward"] = *req.AutoAcceptMaxReward
	}

	// Apply updates
	if err := models.DB.Model(&config).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update config",
			"code":  "UPDATE_ERROR",
		})
	}

	// Reload config
	if err := models.DB.Where("user_id = ?", userUUID).First(&config).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to reload config",
			"code":  "RELOAD_ERROR",
		})
	}

	return c.JSON(configToResponse(&config))
}

// GetState returns the user's watcher state
func (h *WatcherHandler) GetState(c *fiber.Ctx) error {
	userID, ok := GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
			"code":  "NOT_AUTHENTICATED",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
			"code":  "INVALID_USER_ID",
		})
	}

	var state models.WatcherState
	if err := models.DB.Where("user_id = ?", userUUID).First(&state).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Watcher state not found",
			"code":  "STATE_NOT_FOUND",
		})
	}

	// Get live status from manager if available
	status, _ := h.manager.GetStatus(userUUID)
	state.WatcherStatus = status

	return c.JSON(stateToResponse(&state))
}

// StartWatcher starts the user's watcher
func (h *WatcherHandler) StartWatcher(c *fiber.Ctx) error {
	userID, ok := GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
			"code":  "NOT_AUTHENTICATED",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
			"code":  "INVALID_USER_ID",
		})
	}

	if err := h.manager.StartWatcher(userUUID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
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
	userID, ok := GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Not authenticated",
			"code":  "NOT_AUTHENTICATED",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
			"code":  "INVALID_USER_ID",
		})
	}

	if err := h.manager.StopWatcher(userUUID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
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
		"user_id":            state.UserID.String(),
		"watcher_status":     state.WatcherStatus,
		"total_jobs_found":   state.TotalJobsFound,
		"total_jobs_accepted": state.TotalJobsAccepted,
		"total_earnings":     state.TotalEarnings,
		"last_activity":      state.LastActivity,
		"updated_at":         state.UpdatedAt,
	}
}
