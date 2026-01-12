// Package handlers provides HTTP handlers for the API
package handlers

import (
	"context"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/oauth"
)

// OAuthHandler handles OAuth authentication
type OAuthHandler struct {
	oauthService     *oauth.Service
	db               database.Database
	tokenService     *auth.TokenService
	stateStore       oauth.StateStore
	stateExpiry      time.Duration
	frontendRedirect string // URL to redirect users to after successful OAuth
}

// statePattern validates OAuth state format (alphanumeric, 32-64 chars)
var statePattern = regexp.MustCompile(`^[a-zA-Z0-9]{32,64}$`)

// NewOAuthHandler creates a new OAuth handler
// Receives config to ensure OAuth credentials are loaded from centralized configuration
// Receives redisClient for Redis-backed OAuth state storage (H-2 fix: DoS resilience)
func NewOAuthHandler(db database.Database, tokenService *auth.TokenService, cfg *config.Config, redisClient *redis.Client) *OAuthHandler {
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

	return &OAuthHandler{
		oauthService:     oauth.NewService(db, oauthConfig),
		db:               db,
		tokenService:     tokenService,
		stateStore:       oauth.NewRedisStateStore(redisClient),
		frontendRedirect: frontendURL,
		stateExpiry:      10 * time.Minute,
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

	// Store state with expiry in Redis (H-2 fix: DoS resilience via auto-expiring keys)
	ctx := context.Background()
	if err := h.stateStore.Set(ctx, state, h.stateExpiry); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to store state",
			"code":  "STATE_STORAGE_ERROR",
		})
	}

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

	// Verify state exists in Redis (H-2 fix: DoS resilience)
	ctx := context.Background()
	exists, err := h.stateStore.Exists(ctx, state)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to verify state",
			"code":  "STATE_VERIFICATION_ERROR",
		})
	}
	if !exists {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid or expired state",
			"code":  "INVALID_STATE",
		})
	}
	// Delete state after successful verification (single-use)
	_ = h.stateStore.Delete(ctx, state)

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
