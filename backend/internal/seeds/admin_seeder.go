package seeds

import (
	"fmt"
	"log"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// AdminSeeder handles creating and updating admin users
type AdminSeeder struct {
	db       database.Database
	tokenSvc *auth.TokenService
}

// NewAdminSeeder creates a new admin seeder
func NewAdminSeeder(db database.Database, tokenSvc *auth.TokenService) *AdminSeeder {
	return &AdminSeeder{
		db:       db,
		tokenSvc: tokenSvc,
	}
}

// EnsureAdminUser creates or updates an admin user with the given credentials.
// Returns the user and a valid JWT access token.
func (s *AdminSeeder) EnsureAdminUser(email, password string) (*models.User, string, error) {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Check if user exists
	var user models.User
	result := s.db.Where("email = ?", email).First(&user)

	if result.Error != nil {
		// User doesn't exist - create new admin user
		user = models.User{
			Email:        email,
			PasswordHash: string(hashedPassword),
			IsActive:     true,
			Role:         models.RoleAdmin,
		}

		// Create within transaction for dependent records
		tx := s.db.Begin()
		if tx.Error != nil {
			return nil, "", fmt.Errorf("failed to begin transaction: %w", tx.Error)
		}

		// Create user
		if err := tx.Create(&user).Error; err != nil {
			tx.Rollback()
			return nil, "", fmt.Errorf("failed to create user: %w", err)
		}

		// Create WatcherConfig
		config := models.WatcherConfig{
			UserID:                user.ID,
			IncludedLanguagePairs: "[]",
		}
		if err := tx.Create(&config).Error; err != nil {
			tx.Rollback()
			return nil, "", fmt.Errorf("failed to create watcher config: %w", err)
		}

		// Create WatcherState
		state := models.WatcherState{
			UserID:           user.ID,
			WatcherStatus:    "stopped",
			LastSeenJobIDs:    "[]",
			RecentJobHistory:  "[]",
		}
		if err := tx.Create(&state).Error; err != nil {
			tx.Rollback()
			return nil, "", fmt.Errorf("failed to create watcher state: %w", err)
		}

		if err := tx.Commit().Error; err != nil {
			return nil, "", fmt.Errorf("failed to commit transaction: %w", err)
		}

		log.Printf("[AdminSeeder] Created new admin user: %s (ID: %s)", email, user.ID)
	} else {
		// User exists - update dev credentials and admin access.
		needsSave := false
		if user.PasswordHash != string(hashedPassword) {
			user.PasswordHash = string(hashedPassword)
			needsSave = true
		}
		if !user.IsActive {
			user.IsActive = true
			needsSave = true
		}
		if user.Role != models.RoleAdmin {
			user.Role = models.RoleAdmin
			needsSave = true
		}
		if needsSave {
			if err := s.db.Save(&user).Error; err != nil {
				return nil, "", fmt.Errorf("failed to update admin user: %w", err)
			}
			log.Printf("[AdminSeeder] Updated user to admin: %s (ID: %s)", email, user.ID)
		} else {
			log.Printf("[AdminSeeder] Admin user already exists: %s (ID: %s)", email, user.ID)
		}
	}

	// Generate valid JWT token with admin role
	token, err := s.tokenSvc.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return &user, token, nil
}
