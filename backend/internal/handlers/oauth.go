// Package handlers provides HTTP handlers for the API
package handlers

import (
	"context"
	"time"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/oauth"
	"github.com/gofiber/fiber/v2"
)

// OAuthHandler handles OAuth authentication
type OAuthHandler struct {
	oauthService *oauth.Service
	db           database.Database
	tokenService *auth.TokenService
	stateStore   map[string]time.Time
	stateExpiry  time.Duration
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(db database.Database, tokenService *auth.TokenService) *OAuthHandler {
	// OAuth config will be loaded from environment
	config := &oauth.Config{
		GoogleClientID:     "", // Loaded from env
		GoogleClientSecret: "", // Loaded from env
		GitHubClientID:     "", // Loaded from env
		GitHubClientSecret: "", // Loaded from env
		FrontendURL:        "", // Loaded from env
	}

	return &OAuthHandler{
		oauthService: oauth.NewService(db, config),
		db:           db,
		tokenService: tokenService,
		stateStore:   make(map[string]time.Time),
		stateExpiry:  10 * time.Minute,
	}
}

// Authorize starts the OAuth flow
func (h *OAuthHandler) Authorize(c *fiber.Ctx) error {
	provider := c.Query("provider")
	if provider != "google" && provider != "github" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "provider must be 'google' or 'github'",
			"code":   "INVALID_PROVIDER",
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

	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "code and state are required",
			"code":   "MISSING_PARAMS",
		})
	}

	// Verify state
	expiry, exists := h.stateStore[state]
	if !exists || time.Now().After(expiry) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid or expired state",
			"code":   "INVALID_STATE",
		})
	}
	delete(h.stateStore, state)

	// Exchange code for token
	ctx := context.Background()
	token, err := h.oauthService.ExchangeToken(ctx, oauth.Provider(provider), code)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to exchange token: " + err.Error(),
			"code":   "TOKEN_EXCHANGE_ERROR",
		})
	}

	// Fetch user info from provider
	var userInfo *oauth.UserInfo
	if provider == "google" {
		userInfo, err = oauth.FetchGoogleUserInfo(token)
	} else {
		userInfo, err = oauth.FetchGitHubUserInfo(token)
	}

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to fetch user info: " + err.Error(),
			"code":   "USER_INFO_ERROR",
		})
	}

	// Handle OAuth login (create/link user account)
	user, err := h.oauthService.HandleOAuthLogin(ctx, oauth.Provider(provider), code, userInfo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to process login: " + err.Error(),
			"code":   "LOGIN_ERROR",
		})
	}

	// Generate JWT for the user
	accessToken, err := h.tokenService.GenerateAccessToken(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate access token",
			"code":   "TOKEN_ERROR",
		})
	}

	// Set httpOnly cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    accessToken,
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "lax",
	})

	// Return user info and token
	return c.JSON(fiber.Map{
		"access_token": accessToken,
		"user":         user,
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

	// TODO: Fetch linked accounts
	return c.JSON(fiber.Map{
		"linked_accounts": []string{},
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

	// TODO: Unlink account
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
