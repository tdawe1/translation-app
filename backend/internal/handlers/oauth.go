// Package handlers provides HTTP handlers for the API
package handlers

import (
	"context"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/oauth"
)

// stateStore provides thread-safe storage for OAuth state tokens
type stateStore struct {
	sync.RWMutex
	m map[string]time.Time
}

// OAuthHandler handles OAuth authentication
type OAuthHandler struct {
	oauthService  *oauth.Service
	db            database.Database
	tokenService  *auth.TokenService
	states        *stateStore
	stateExpiry   time.Duration
	stopCleanup   chan struct{}
	frontendRedirect string // URL to redirect users to after successful OAuth
}

// statePattern validates OAuth state format (alphanumeric, 32-64 chars)
var statePattern = regexp.MustCompile(`^[a-zA-Z0-9]{32,64}$`)

// NewOAuthHandler creates a new OAuth handler
// Receives config to ensure OAuth credentials are loaded from centralized configuration
func NewOAuthHandler(db database.Database, tokenService *auth.TokenService, cfg *config.Config) *OAuthHandler {
	// Backend URL for OAuth callbacks (where GitHub/Google sends the code)
	// This should be the backend URL with the callback endpoint path
	callbackURL := cfg.OAuthRedirectURL
	if callbackURL == "" {
		callbackURL = "http://localhost:8000"
	}

	// Frontend URL for redirecting users after successful login
	// This is where users land after OAuth completes
	frontendURL := cfg.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:3001"
	}

	// Load OAuth config from centralized config
	oauthConfig := &oauth.Config{
		GoogleClientID:     cfg.GoogleOAuthClientID,
		GoogleClientSecret: cfg.GoogleOAuthClientSecret,
		GitHubClientID:     cfg.GitHubOAuthClientID,
		GitHubClientSecret: cfg.GitHubOAuthClientSecret,
		FrontendURL:        callbackURL, // This is used for callback URLs (backend)
	}

	h := &OAuthHandler{
		oauthService:     oauth.NewService(db, oauthConfig),
		db:               db,
		tokenService:     tokenService,
		frontendRedirect: frontendURL,
		states: &stateStore{
			m: make(map[string]time.Time),
		},
		stateExpiry: 10 * time.Minute,
		stopCleanup: make(chan struct{}),
	}

	// Start background cleanup worker (#003 fix)
	go h.startCleanupWorker()

	return h
}

// startCleanupWorker runs a background goroutine to clean up expired states
func (h *OAuthHandler) startCleanupWorker() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.cleanupExpiredStates()
		case <-h.stopCleanup:
			return
		}
	}
}

// Stop stops the cleanup worker
func (h *OAuthHandler) Stop() {
	close(h.stopCleanup)
}

// setState stores a state token with thread safety (#001 fix)
func (h *OAuthHandler) setState(key string, expiry time.Time) {
	h.states.Lock()
	defer h.states.Unlock()
	h.states.m[key] = expiry
}

// getState retrieves a state token with thread safety (#001 fix)
func (h *OAuthHandler) getState(key string) (time.Time, bool) {
	h.states.RLock()
	defer h.states.RUnlock()
	val, ok := h.states.m[key]
	return val, ok
}

// deleteState removes a state token with thread safety (#001 fix)
func (h *OAuthHandler) deleteState(key string) {
	h.states.Lock()
	defer h.states.Unlock()
	delete(h.states.m, key)
}

// cleanupExpiredStates removes expired states from the store (#003 fix)
func (h *OAuthHandler) cleanupExpiredStates() {
	h.states.Lock()
	defer h.states.Unlock()

	now := time.Now()
	for state, expiry := range h.states.m {
		if now.After(expiry) {
			delete(h.states.m, state)
		}
	}
}

// ValidateProvider checks if provider is valid (shared helper for #012)
func ValidateProvider(provider string) bool {
	return provider == "google" || provider == "github"
}

// Authorize starts the OAuth flow
func (h *OAuthHandler) Authorize(c *fiber.Ctx) error {
	provider := c.Query("provider")
	if !ValidateProvider(provider) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "provider must be 'google' or 'github'",
			"code":  "INVALID_PROVIDER",
		})
	}

	// #016 fix - CSRF protection: bind state to session
	// Get or create session ID for CSRF binding
	sessionID := c.Cookies("csrf_session")
	if sessionID == "" {
		rawState, err := oauth.GenerateState()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to generate session",
				"code":  "SESSION_ERROR",
			})
		}
		sessionID = rawState[:16] // Use first 16 chars as session ID
		c.Cookie(&fiber.Cookie{
			Name:     "csrf_session",
			Value:    sessionID,
			HTTPOnly: true,
			Secure:   c.Protocol() == "https",
			SameSite: "lax",
			MaxAge:   600, // 10 minutes
		})
	}

	// Generate state with session binding
	rawState, err := oauth.GenerateState()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate state",
			"code":  "STATE_ERROR",
		})
	}

	// Bind state to session: state = sessionID:randomState
	state := sessionID + ":" + rawState

	// Store state with expiry (thread-safe)
	h.setState(state, time.Now().Add(h.stateExpiry))

	// Get auth URL
	authURL, err := h.oauthService.GetAuthURL(oauth.Provider(provider), state)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "authentication service error",
			"code":  "AUTH_URL_ERROR",
		})
	}

	return c.JSON(fiber.Map{
		"auth_url": authURL,
	})
}

// Callback handles the OAuth callback
func (h *OAuthHandler) Callback(c *fiber.Ctx) error {
	// Extract provider from path since routes are static (/google/callback, /github/callback)
	// Path format: /api/v1/oauth/google/callback or /api/v1/oauth/github/callback
	// Split: ["", "api", "v1", "oauth", "github", "callback"] - provider at index 4
	pathParts := strings.Split(c.Path(), "/")
	var provider string
	if len(pathParts) >= 5 {
		provider = pathParts[4]
	}

	if !ValidateProvider(provider) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid provider",
			"code":  "INVALID_PROVIDER",
		})
	}

	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "code and state are required",
			"code":  "MISSING_PARAMS",
		})
	}

	// #016 fix - CSRF protection: verify session binding
	sessionID := c.Cookies("csrf_session")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "missing session",
			"code":  "INVALID_STATE",
		})
	}

	// State should be in format: sessionID:randomState
	parts := strings.Split(state, ":")
	if len(parts) != 2 || parts[0] != sessionID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid state - session mismatch",
			"code":  "INVALID_STATE",
		})
	}

	// Verify state (thread-safe)
	expiry, exists := h.getState(state)
	if !exists || time.Now().After(expiry) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid or expired state",
			"code":  "INVALID_STATE",
		})
	}
	h.deleteState(state)

	// Clear the csrf session cookie after use
	c.Cookie(&fiber.Cookie{
		Name:     "csrf_session",
		Value:    "",
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "lax",
		MaxAge:   -1, // Delete cookie
	})

	// Exchange code for token
	ctx := context.Background()
	token, err := h.oauthService.ExchangeToken(ctx, oauth.Provider(provider), code)
	if err != nil {
		// Don't leak internal error details (#022 fix)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to exchange token",
			"code":  "TOKEN_EXCHANGE_ERROR",
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
		// Don't leak internal error details (#022 fix)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "failed to fetch user info",
			"code":  "USER_INFO_ERROR",
		})
	}

	// Handle OAuth login (create/link user account)
	user, err := h.oauthService.HandleOAuthLogin(ctx, oauth.Provider(provider), code, userInfo)
	if err != nil {
		// Don't leak internal error details (#022 fix)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to process login",
			"code":  "LOGIN_ERROR",
		})
	}

	// Generate JWT for the user
	accessToken, err := h.tokenService.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate access token",
			"code":  "TOKEN_ERROR",
		})
	}

	// Set httpOnly cookie with correct name (#006 fix) and SameSite Strict (#021 fix)
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    accessToken,
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "strict",
	})

	// Return only user info, not token (#002 fix - prevent token leak in response)
	// But redirect to frontend dashboard for OAuth callback completion
	if h.frontendRedirect != "" {
		log.Printf("OAuth redirect: Redirecting user to %s", h.frontendRedirect+"/dashboard")
		return c.Redirect(h.frontendRedirect + "/dashboard")
	}
	return c.JSON(fiber.Map{
		"user": user,
	})
}

// GetLinkedAccounts returns the user's linked OAuth accounts
func (h *OAuthHandler) GetLinkedAccounts(c *fiber.Ctx) error {
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
			"code":  "UNAUTHORIZED",
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
	if !ValidateProvider(provider) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid provider",
			"code":  "INVALID_PROVIDER",
		})
	}

	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
			"code":  "UNAUTHORIZED",
		})
	}

	// TODO: Unlink account
	return c.SendStatus(fiber.StatusNoContent)
}
