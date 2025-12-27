package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/tdawe1/translation-app/internal/models"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	jwtSecret []byte
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(jwtSecret string) *AuthHandler {
	return &AuthHandler{
		jwtSecret: []byte(jwtSecret),
	}
}

// RegisterRequest represents registration input
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents login input
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents successful authentication
type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	User        UserResponse  `json:"user"`
}

// UserResponse represents user data
type UserResponse struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	IsActive     bool   `json:"is_active"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Validate input
	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 8 characters",
			"code":  "WEAK_PASSWORD",
		})
	}

	// Check if user exists
	var existingUser models.User
	result := models.DB.Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "User with this email already exists",
			"code":  "USER_EXISTS",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process password",
			"code":  "PASSWORD_ERROR",
		})
	}

	// Create user
	user := models.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		IsActive:     true,
	}

	// Start transaction
	tx := models.DB.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Database error",
			"code":  "DATABASE_ERROR",
		})
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
			"code":  "CREATE_ERROR",
		})
	}

	// Create default watcher config
	config := models.WatcherConfig{
		UserID: user.ID,
	}
	if err := tx.Create(&config).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create watcher config",
			"code":  "CONFIG_ERROR",
		})
	}

	// Create default watcher state
	state := models.WatcherState{
		UserID:       user.ID,
		WatcherStatus: "stopped",
	}
	if err := tx.Create(&state).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create watcher state",
			"code":  "STATE_ERROR",
		})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to commit transaction",
			"code":  "COMMIT_ERROR",
		})
	}

	// Generate tokens
	accessToken, err := h.generateAccessToken(user.ID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
			"code":  "TOKEN_ERROR",
		})
	}

	// Set httpOnly cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    accessToken,
		HTTPOnly: true,
		Secure:   false, // Set true in production with HTTPS
		SameSite: "Lax",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	return c.Status(fiber.StatusCreated).JSON(AuthResponse{
		AccessToken: accessToken,
		User:        userToResponse(&user),
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
	}

	// Find user
	var user models.User
	result := models.DB.Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
			"code":  "INVALID_CREDENTIALS",
		})
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid email or password",
			"code":  "INVALID_CREDENTIALS",
		})
	}

	if !user.IsActive {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "User account is inactive",
			"code":  "INACTIVE_USER",
		})
	}

	// Generate token
	accessToken, err := h.generateAccessToken(user.ID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
			"code":  "TOKEN_ERROR",
		})
	}

	// Set httpOnly cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    accessToken,
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	return c.JSON(AuthResponse{
		AccessToken: accessToken,
		User:        userToResponse(&user),
	})
}

// GetMe returns current user info
func (h *AuthHandler) GetMe(c *fiber.Ctx) error {
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

	var user models.User
	if err := models.DB.Where("id = ?", userUUID).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
			"code":  "USER_NOT_FOUND",
		})
	}

	return c.JSON(userToResponse(&user))
}

// Logout handles logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	c.ClearCookie("session_token")
	return c.SendStatus(fiber.StatusNoContent)
}

// generateAccessToken generates a JWT access token
func (h *AuthHandler) generateAccessToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(15 * time.Minute).Unix(),
		"iat": time.Now().Unix(),
		"type": "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

// userToResponse converts User model to UserResponse
func userToResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		IsActive:      user.IsActive,
	}
}

// GetUserID is exported from middleware
var GetUserID = func(c *fiber.Ctx) (string, bool) {
	claims := c.Locals("user")
	if claims == nil {
		return "", false
	}

	userClaims, ok := claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}

	if sub, ok := userClaims["sub"].(string); ok {
		return sub, true
	}

	return "", false
}
