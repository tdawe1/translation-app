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
		UserID: user.ID,
	}
	if err := tx.Create(&config).Error; err != nil {
		tx.Rollback()
		return nil, apperrors.New(apperrors.ErrConfigError, "Failed to create watcher config")
	}

	// Create default watcher state
	state := models.WatcherState{
		UserID:        user.ID,
		WatcherStatus: "stopped",
	}
	if err := tx.Create(&state).Error; err != nil {
		tx.Rollback()
		return nil, apperrors.New(apperrors.ErrStateError, "Failed to create watcher state")
	}

	if err := tx.Commit().Error; err != nil {
		return nil, apperrors.New(apperrors.ErrCommitError, "Failed to commit transaction")
	}

	// Generate access token
	accessToken, err := s.tokenSvc.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrTokenError, "Failed to generate token")
	}

	return &AuthResult{
		AccessToken: accessToken,
		User:        &user,
	}, nil
}

// Login authenticates a user with email and password
func (s *UserService) Login(req LoginRequest) (*AuthResult, error) {
	// Find user
	var user models.User
	result := s.db.Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrInactiveUser
	}

	// Generate access token
	accessToken, err := s.tokenSvc.GenerateAccessToken(user.ID)
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
	err := s.db.Where("id = ?", userID).First(&user).Error
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
