package auth

import (
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/tdawe1/translation-app/internal/database"
	apperrors "github.com/tdawe1/translation-app/internal/errors"
	"github.com/tdawe1/translation-app/internal/models"
)

// UserService handles user business logic
type UserService struct {
	db          database.Database
	tokenSvc    *TokenService
}

// NewUserService creates a new user service
func NewUserService(db database.Database, tokenSvc *TokenService) *UserService {
	return &UserService{
		db:       db,
		tokenSvc: tokenSvc,
	}
}

// RegisterRequest represents user registration input
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents user login input
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResult represents a successful authentication
type AuthResult struct {
	AccessToken string     `json:"access_token"`
	User        *models.User `json:"user"`
}

// Minimum password length
const minPasswordLength = 8

var (
	ErrUserExists    = apperrors.New(apperrors.ErrUserExists, "User with this email already exists")
	ErrInvalidCredentials = apperrors.New(apperrors.ErrInvalidCredentials, "Invalid email or password")
	ErrInactiveUser = apperrors.New(apperrors.ErrInactiveUser, "User account is inactive")
	ErrWeakPassword = apperrors.New(apperrors.ErrWeakPassword, "Password must be at least 8 characters")
)

// Register creates a new user account with default watcher config and state
func (s *UserService) Register(req RegisterRequest) (*AuthResult, error) {
	// Validate password
	if len(req.Password) < minPasswordLength {
		return nil, ErrWeakPassword
	}

	// Check if user exists
	var existingUser models.User
	result := s.db.Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		return nil, ErrUserExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrPasswordError, "Failed to process password")
	}

	// Create user within transaction
	user := models.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		IsActive:     true,
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, apperrors.New(apperrors.ErrDatabase, "Database error")
	}

	// Create user
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return nil, apperrors.New(apperrors.ErrCreateError, "Failed to create user")
	}

	// Create default watcher config
	config := models.WatcherConfig{
		UserID:                user.ID,
		IncludedLanguagePairs: "[]", // Valid JSON array for jsonb field
	}
	if err := tx.Create(&config).Error; err != nil {
		tx.Rollback()
		return nil, apperrors.New(apperrors.ErrConfigError, "Failed to create watcher config")
	}

	// Create default watcher state
	state := models.WatcherState{
		UserID:           user.ID,
		WatcherStatus:    "stopped",
		LastSeenJobIDs:   "[]", // Valid JSON array for jsonb field
		RecentJobHistory: "[]", // Valid JSON array for jsonb field
	}
	if err := tx.Create(&state).Error; err != nil {
		tx.Rollback()
		return nil, apperrors.New(apperrors.ErrStateError, "Failed to create watcher state")
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.New(apperrors.ErrCommitError, "Failed to commit transaction")
	}

	// Generate access token with user role
	accessToken, err := s.tokenSvc.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrTokenError, "Failed to generate token")
	}

	return &AuthResult{
		AccessToken: accessToken,
		User:        &user,
	}, nil
}

// Login authenticates a user with email and password
//
// Timing-safe user enumeration prevention: Always performs bcrypt comparison
// using a dummy hash for non-existent users to normalize timing between
// "user not found" and "wrong password" cases.
func (s *UserService) Login(req LoginRequest) (*AuthResult, error) {
	// Fast-path validation for empty inputs (still returns generic error)
	if req.Email == "" || req.Password == "" {
		return nil, ErrInvalidCredentials
	}

	// Find user
	var user models.User
	result := s.db.Where("email = ?", req.Email).First(&user)

	// Dummy bcrypt hash for non-existent users (same cost factor as real hashes)
	// Using bcrypt.DefaultCost (currently 10) ensures timing normalization
	// Format: $2a$[cost]$[22 characters of salt][31 characters of hash]
	dummyHash := []byte("$2a$10$dummy.hash.bcrypt.cost.f")

	var err error
	if result.Error != nil {
		// User doesn't exist - still perform bcrypt to normalize timing
		// This prevents timing-based user enumeration attacks
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(req.Password))
		// Always return generic error regardless of bcrypt result
		return nil, ErrInvalidCredentials
	}

	// User exists - verify password with their actual hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrInactiveUser
	}

	// Generate access token with user role
	accessToken, err := s.tokenSvc.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrTokenError, "Failed to generate token")
	}

	return &AuthResult{
		AccessToken: accessToken,
		User:        &user,
	}, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.Where("id = ?", userID).Preload("OAuthAccounts").First(&user).Error
	if err != nil {
		return nil, apperrors.New(apperrors.ErrUserNotFound, "User not found")
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := s.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, apperrors.New(apperrors.ErrUserNotFound, "User not found")
	}
	return &user, nil
}

// IsPasswordValid checks if the provided password matches the hash
func (s *UserService) IsPasswordValid(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// HashPassword creates a bcrypt hash of the password
func (s *UserService) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// ChangePasswordRequest represents password change input
type ChangePasswordRequest struct {
	UserID      uuid.UUID
	OldPassword string
	NewPassword string
}

// ChangePassword updates a user's password after verifying the old password
func (s *UserService) ChangePassword(req ChangePasswordRequest) error {
	// Fetch user with password hash
	var user models.User
	err := s.db.Where("id = ?", req.UserID).First(&user).Error
	if err != nil {
		return apperrors.New(apperrors.ErrUserNotFound, "User not found")
	}

	// Check if user has a password (OAuth users might not)
	if user.PasswordHash == "" {
		return apperrors.New(apperrors.ErrInvalidRequest, "Account uses OAuth sign-in. Set a password first.")
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return apperrors.New(apperrors.ErrInvalidCredentials, "Current password is incorrect")
	}

	// Validate new password
	if len(req.NewPassword) < minPasswordLength {
		return ErrWeakPassword
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperrors.New(apperrors.ErrPasswordError, "Failed to process password")
	}

	// Update password
	user.PasswordHash = string(hashedPassword)
	if err := s.db.Save(&user).Error; err != nil {
		return apperrors.New(apperrors.ErrUpdateError, "Failed to update password")
	}

	return nil
}

// OAuthUserInfo represents user info from OAuth providers
type OAuthUserInfo struct {
	Name    string
	Email   string
	Picture string
}

// FindOrCreateByOAuth finds a user by OAuth provider info or creates a new one
func (s *UserService) FindOrCreateByOAuth(provider, providerID, email string, info OAuthUserInfo) (*models.User, error) {
	// First, try to find existing user by provider + provider_id
	var existingUser models.User
	result := s.db.Where("provider = ? AND provider_id = ?", provider, providerID).First(&existingUser)
	if result.Error == nil {
		// User exists, update email if changed
		if existingUser.Email != email {
			existingUser.Email = email
			s.db.Save(&existingUser)
		}
		return &existingUser, nil
	}

	// Check if email is already used by a different auth method
	result = s.db.Where("email = ?", email).First(&existingUser)
	if result.Error == nil {
		// Email exists but with different provider - link the OAuth account
		existingUser.Provider = provider
		existingUser.ProviderID = providerID
		s.db.Save(&existingUser)
		return &existingUser, nil
	}

	// Create new user from OAuth
	user := models.User{
		Email:        email,
		EmailVerified: true, // OAuth providers verify emails
		IsActive:     true,
		Provider:     provider,
		ProviderID:   providerID,
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, apperrors.New(apperrors.ErrDatabase, "Database error")
	}

	// Create user
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return nil, apperrors.New(apperrors.ErrCreateError, "Failed to create user")
	}

	// Create default watcher config
	config := models.WatcherConfig{
		UserID:                user.ID,
		IncludedLanguagePairs: "[]", // Valid JSON array for jsonb field
	}
	if err := tx.Create(&config).Error; err != nil {
		tx.Rollback()
		return nil, apperrors.New(apperrors.ErrConfigError, "Failed to create watcher config")
	}

	// Create default watcher state
	state := models.WatcherState{
		UserID:           user.ID,
		WatcherStatus:    "stopped",
		LastSeenJobIDs:   "[]", // Valid JSON array for jsonb field
		RecentJobHistory: "[]", // Valid JSON array for jsonb field
	}
	if err := tx.Create(&state).Error; err != nil {
		tx.Rollback()
		return nil, apperrors.New(apperrors.ErrStateError, "Failed to create watcher state")
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.New(apperrors.ErrCommitError, "Failed to commit transaction")
	}

	return &user, nil
}
