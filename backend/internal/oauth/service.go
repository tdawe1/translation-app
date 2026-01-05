// Package oauth provides OAuth2 authentication for Google and GitHub
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/password"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"gorm.io/gorm"
)

// Provider represents an OAuth provider
type Provider string

const (
	ProviderGoogle Provider = "google"
	ProviderGitHub Provider = "github"
)

// Service handles OAuth operations
type Service struct {
	db     database.Database
	config *Config
}

// Config holds OAuth configuration
type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	FrontendURL        string
}

// UserInfo represents user info from OAuth provider
type UserInfo struct {
	ID       string
	Email    string
	Name     string
	Verified bool
}

// NewService creates a new OAuth service
func NewService(db database.Database, config *Config) *Service {
	return &Service{
		db:     db,
		config: config,
	}
}

// GoogleConfig returns the OAuth2 config for Google
func (s *Service) GoogleConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.config.GoogleClientID,
		ClientSecret: s.config.GoogleClientSecret,
		RedirectURL:  fmt.Sprintf("%s/api/v1/oauth/google/callback", s.config.FrontendURL),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

// GitHubConfig returns the OAuth2 config for GitHub
func (s *Service) GitHubConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.config.GitHubClientID,
		ClientSecret: s.config.GitHubClientSecret,
		RedirectURL:  fmt.Sprintf("%s/api/v1/oauth/github/callback", s.config.FrontendURL),
		Scopes: []string{
			"user:email",
			"read:user",
		},
		Endpoint: github.Endpoint,
	}
}

// GenerateState generates a random state parameter for OAuth
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetAuthURL returns the OAuth authorization URL
func (s *Service) GetAuthURL(provider Provider, state string) (string, error) {
	var config *oauth2.Config

	switch provider {
	case ProviderGoogle:
		config = s.GoogleConfig()
	case ProviderGitHub:
		config = s.GitHubConfig()
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	return config.AuthCodeURL(state), nil
}

// ExchangeToken exchanges the authorization code for an access token
func (s *Service) ExchangeToken(ctx context.Context, provider Provider, code string) (*oauth2.Token, error) {
	var config *oauth2.Config

	switch provider {
	case ProviderGoogle:
		config = s.GoogleConfig()
	case ProviderGitHub:
		config = s.GitHubConfig()
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	return config.Exchange(ctx, code)
}

// HandleOAuthLogin handles the complete OAuth flow
func (s *Service) HandleOAuthLogin(ctx context.Context, provider Provider, code string, userInfo *UserInfo) (*models.User, error) {
	// Check if OAuth account exists
	var oauthAccount models.OAuthAccount
	err := s.db.Where("provider = ? AND provider_user_id = ?", string(provider), userInfo.ID).
		First(&oauthAccount).Error

	if err == nil {
		// OAuth account exists, fetch the user
		var user models.User
		err = s.db.Where("id = ?", oauthAccount.UserID).First(&user).Error
		if err != nil {
			return nil, fmt.Errorf("failed to fetch user: %w", err)
		}
		return &user, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Check if user exists with this email
	var user models.User
	err = s.db.Where("email = ?", userInfo.Email).First(&user).Error

	if err == nil {
		// User exists, link OAuth account
		oauthAccount = models.OAuthAccount{
			UserID:         user.ID,
			Provider:       string(provider),
			ProviderUserID: userInfo.ID,
		}
		if err := s.db.Create(&oauthAccount).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Create new user (#015 fix - handle race condition with unique constraint)
	user = models.User{
		Email:         userInfo.Email,
		EmailVerified: userInfo.Verified,
		IsActive:      true,
	}

	// Generate a random password for OAuth users
	randomPassword, err := password.GenerateRandomPassword(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}
	hashedPassword, err := password.HashPassword(randomPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = hashedPassword

	// Start transaction
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	// Create user - handle race condition with unique email constraint
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		// Check if it's a duplicate key error (race condition)
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "UNIQUE constraint") {
			// Another request created the user - fetch and return it
			if fetchErr := s.db.Where("email = ?", userInfo.Email).First(&user).Error; fetchErr != nil {
				return nil, fmt.Errorf("failed to fetch user after race: %w", fetchErr)
			}
			// Link OAuth account to existing user
			oauthAccount = models.OAuthAccount{
				UserID:         user.ID,
				Provider:       string(provider),
				ProviderUserID: userInfo.ID,
			}
			if createErr := s.db.Create(&oauthAccount).Error; createErr != nil {
				// Ignore if OAuth account also already exists
				if !strings.Contains(createErr.Error(), "duplicate") && !strings.Contains(createErr.Error(), "unique") {
					return nil, fmt.Errorf("failed to link oauth account: %w", createErr)
				}
			}
			return &user, nil
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create OAuth account
	oauthAccount = models.OAuthAccount{
		UserID:         user.ID,
		Provider:       string(provider),
		ProviderUserID: userInfo.ID,
	}
	if err := tx.Create(&oauthAccount).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create oauth account: %w", err)
	}

	// #017 fix - Don't create watcher config in auth flow
	// Watcher resources will be created lazily on first access

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &user, nil
}

// GetLinkedAccounts returns all linked OAuth accounts for a user
func (s *Service) GetLinkedAccounts(userID uuid.UUID) ([]models.OAuthAccount, error) {
	var accounts []models.OAuthAccount
	result := s.db.Where("user_id = ?", userID).Find(&accounts)
	if result.Error != nil {
		return nil, result.Error
	}
	return accounts, nil
}

// UnlinkOAuthAccount unlinks an OAuth account from a user
func (s *Service) UnlinkOAuthAccount(userID uuid.UUID, provider Provider) error {
	return s.db.Where("user_id = ? AND provider = ?", userID, string(provider)).
		Delete(&models.OAuthAccount{}).Error
}
