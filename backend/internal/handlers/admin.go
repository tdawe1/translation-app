package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/tdawe1/translation-app/internal/errors"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
)

// AdminHandler handles admin-only operations
type AdminHandler struct {
	db database.Database
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(db database.Database) *AdminHandler {
	return &AdminHandler{
		db: db,
	}
}

// ListUsersRequest represents query parameters for listing users
type ListUsersRequest struct {
	Page     int    `query:"page" default:"1"`
	PageSize int    `query:"page_size" default:"20"`
	Search   string `query:"search"`
	Role     string `query:"role"`
}

// ListUsersResponse represents the paginated users response
type ListUsersResponse struct {
	Users      []models.User `json:"users"`
	TotalCount int64          `json:"total_count"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// ListUsers returns a paginated list of users with optional filtering
func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	// Parse query parameters
	req := ListUsersRequest{
		Page:     c.QueryInt("page", 1),
		PageSize: c.QueryInt("page_size", 20),
		Search:   c.Query("search", ""),
		Role:     c.Query("role", ""),
	}

	// Validate page size
	if req.PageSize < 1 || req.PageSize > 100 {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "page_size must be between 1 and 100")
	}
	if req.Page < 1 {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "page must be at least 1")
	}

	// Build query
	query := h.db.Model(&models.User{})

	// Apply role filter
	if req.Role != "" {
		if req.Role != models.RoleAdmin && req.Role != models.RoleUser {
			return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "role must be 'admin' or 'user'")
		}
		query = query.Where("role = ?", req.Role)
	}

	// Apply search filter (email or name)
	if req.Search != "" {
		searchPattern := "%" + req.Search + "%"
		query = query.Where("email ILIKE ?", searchPattern)
	}

	// Count total
	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		log.Printf("[Admin] Error counting users: %v", err)
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to count users")
	}

	// Calculate pagination
	totalPages := int(totalCount) / req.PageSize
	if int(totalCount)%req.PageSize != 0 {
		totalPages++
	}
	if req.Page > totalPages && totalPages > 0 {
		req.Page = totalPages
	}

	// Fetch users
	offset := (req.Page - 1) * req.PageSize
	var users []models.User
	if err := query.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		log.Printf("[Admin] Error fetching users: %v", err)
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDatabase, "Failed to fetch users")
	}

	// Clear sensitive data before returning
	for i := range users {
		users[i].PasswordHash = ""
	}

	return c.JSON(ListUsersResponse{
		Users:      users,
		TotalCount: totalCount,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	})
}

// UpdateUserRoleRequest represents the request to update a user's role
type UpdateUserRoleRequest struct {
	Role string `json:"role"`
}

// UpdateUserRole updates a user's role (admin/user)
func (h *AdminHandler) UpdateUserRole(c *fiber.Ctx) error {
	userID, err := ParseUserID(c.Params("id"))
	if err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidUserID, "Invalid user ID")
	}

	var req UpdateUserRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid request body")
	}

	// Validate role
	if req.Role != models.RoleAdmin && req.Role != models.RoleUser {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "role must be 'admin' or 'user'")
	}

	// Get the requesting user's ID (from JWT claims)
	claims := c.Locals("user")
	if claims == nil {
		return RespondWithError(c, fiber.StatusUnauthorized, apperrors.ErrNotAuthenticated, "Not authenticated")
	}
	claimMap, _ := claims.(map[string]interface{})
	requestingUserID, _ := claimMap["sub"].(string)

	// Prevent users from changing their own role
	if requestingUserID == userID.String() {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Cannot change your own role")
	}

	// Update user role
	result := h.db.Model(&models.User{}).Where("id = ?", userID).Update("role", req.Role)
	if result.Error != nil {
		log.Printf("[Admin] Error updating user role: %v", result.Error)
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrUpdateError, "Failed to update user role")
	}

	if result.RowsAffected == 0 {
		return RespondWithError(c, fiber.StatusNotFound, apperrors.ErrUserNotFound, "User not found")
	}

	// Fetch updated user
	var user models.User
	if err := h.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return RespondWithError(c, fiber.StatusNotFound, apperrors.ErrUserNotFound, "User not found")
	}

	user.PasswordHash = "" // Clear sensitive data

	log.Printf("[Admin] User %s updated role of user %s to %s", requestingUserID, userID, req.Role)

	return c.JSON(fiber.Map{
		"message": "User role updated",
		"user":    user,
	})
}

// DeleteUser deletes a user account
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	userID, err := ParseUserID(c.Params("id"))
	if err != nil {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidUserID, "Invalid user ID")
	}

	// Get the requesting user's ID (from JWT claims)
	claims := c.Locals("user")
	if claims == nil {
		return RespondWithError(c, fiber.StatusUnauthorized, apperrors.ErrNotAuthenticated, "Not authenticated")
	}
	claimMap, _ := claims.(map[string]interface{})
	requestingUserID, _ := claimMap["sub"].(string)

	// Prevent users from deleting themselves
	if requestingUserID == userID.String() {
		return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Cannot delete your own account")
	}

	// Check if user exists
	var user models.User
	if err := h.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return RespondWithError(c, fiber.StatusNotFound, apperrors.ErrUserNotFound, "User not found")
	}

	// Delete user (cascade will handle related records)
	if err := h.db.Delete(&user).Error; err != nil {
		log.Printf("[Admin] Error deleting user: %v", err)
		return RespondWithError(c, fiber.StatusInternalServerError, apperrors.ErrDeleteError, "Failed to delete user")
	}

	log.Printf("[Admin] User %s deleted user %s (%s)", requestingUserID, userID, user.Email)

	return c.SendStatus(fiber.StatusNoContent)
}
