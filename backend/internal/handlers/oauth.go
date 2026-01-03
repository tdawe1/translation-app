// Package handlers provides HTTP handlers for the API
package handlers

import (
	"time"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/oauth"
	"github.com/gofiber/fiber/v2"
)

// OAuthHandler handles OAuth authentication
type OAuthHandler struct {
	oauthService   *oauth.Service
	userService    *auth.UserService
	stateStore     map[string]time.Time // Simple state store (in-memory, not production-ready)
	stateExpiry    time.Duration
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(oauthService *oauth.Service, userService *auth.UserService) *OAuthHandler {
	return &OAuthHandler{
		oauthService: oauthService,
		userService:  userService,
		stateStore:   make(map[string]time.Time),
		stateExpiry:  10 * time.Minute,
	}
}

// OAuthAuthorizeRequest represents the authorize request
type OAuthAuthorizeRequest struct {
	Provider string `query:"provider" validate:"required,oneof=google github"`
}

// OAuthCallbackRequest represents the callback request
type OAuthCallbackRequest struct {
	Code  string `query:"code" validate:"required"`
	State string `query:"state" validate:"required"`
}

// Authorize starts the OAuth flow
func (h *OAuthHandler) Authorize(c *fiber.Ctx) error {
	provider := c.Query("provider")
	if provider == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "provider is required",
			"code":   "MISSING_PROVIDER",
		})
	}

	// Generate state
	state, err := oauth.GenerateState()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate state",
			"code":   "STATE_ERROR",
		})
	}

	// Store state with expiry
	h.stateStore[state] = time.Now().Add(h.stateExpiry)

	// Get auth URL
	authURL, err := h.oauthService.GetAuthURL(oauth.Provider(provider), state)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
			"code":   "AUTH_URL_ERROR",
		})
	}

	return c.JSON(fiber.Map{
		"auth_url": authURL,
		"state":    state,
	})
}

// Callback handles the OAuth callback
func (h *OAuthHandler) Callback(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider != "google" && provider != "github" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid provider",
			"code":   "INVALID_PROVIDER",
		})
	}

	var req OAuthCallbackRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid callback parameters",
			"code":   "INVALID_PARAMS",
		})
	}

	// Verify state
	expiry, exists := h.stateStore[req.State]
	if !exists || time.Now().After(expiry) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid or expired state",
			"code":   "INVALID_STATE",
		})
	}
	delete(h.stateStore, req.State)

	// TODO: Exchange code for token and get user info
	// This requires implementing actual HTTP calls to Google/GitHub APIs
	// For now, return a placeholder response
	return c.JSON(fiber.Map{
		"message": "OAuth callback endpoint - requires implementation of token exchange and user info fetching",
		"code":    req.Code,
	})
}

// GetLinkedAccounts returns the user's linked OAuth accounts
func (h *OAuthHandler) GetLinkedAccounts(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
			"code":   "UNAUTHORIZED",
		})
	}

	// TODO: Implement fetching linked accounts
	return c.JSON(fiber.Map{
		"linked_accounts": []interface{}{},
	})
}

// UnlinkAccount unlinks an OAuth account
func (h *OAuthHandler) UnlinkAccount(c *fiber.Ctx) error {
	provider := c.Params("provider")
	if provider != "google" && provider != "github" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid provider",
			"code":   "INVALID_PROVIDER",
		})
	}

	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
			"code":   "UNAUTHORIZED",
		})
	}

	// TODO: Implement unlinking
	return c.SendStatus(fiber.StatusNoContent)
}

// CleanupExpiredStates removes expired states from the store
func (h *OAuthHandler) CleanupExpiredStates() {
	now := time.Now()
	for state, expiry := range h.stateStore {
		if now.After(expiry) {
			delete(h.stateStore, state)
		}
	}
}
