# Canonical Code Patterns

## Database Model
See: `backend/internal/models/user.go`, type `User`

```go
type User struct {
    Base
    Email         string         `gorm:"uniqueIndex;size:255;not null" json:"email"`
    EmailVerified bool           `gorm:"default:false" json:"email_verified"`
    PasswordHash  string         `gorm:"size:255" json:"-"` // Never serialize in JSON
    IsActive      bool           `gorm:"default:true" json:"is_active"`

    // Relationships (lazy-loaded, cascade delete)
    OAuthAccounts []OAuthAccount `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"oauth_accounts,omitempty"`
    APIKeys       []APIKey       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"api_keys,omitempty"`
}

type Base struct {
    ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
    CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
```

## API Route
See: `backend/internal/handlers/auth.go`, `Register()`

```go
type AuthHandler struct {
    userService *auth.UserService
    secureCookie bool
}

func NewAuthHandler(userService *auth.UserService, secureCookie bool) *AuthHandler {
    return &AuthHandler{
        userService:  userService,
        secureCookie: secureCookie,
    }
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
    var req RegisterRequest
    if err := c.BodyParser(&req); err != nil {
        return RespondWithError(c, fiber.StatusBadRequest, apperrors.ErrInvalidRequest, "Invalid request body")
    }

    // Business logic via service layer
    result, apiErr := h.userService.Register(auth.RegisterRequest{
        Email:    req.Email,
        Password: req.Password,
    })

    if apiErr != nil {
        status := h.statusCodeForError(apiErr.(*apperrors.APIError).Code)
        return RespondWithAPIError(c, status, apiErr.(*apperrors.APIError))
    }

    // Set httpOnly cookie (JWT in refresh token pattern)
    SetSessionCookie(c, result.AccessToken, h.secureCookie)

    return c.Status(fiber.StatusCreated).JSON(AuthResponse{
        AccessToken: result.AccessToken,
        User:        UserToResponse(result.User),
    })
}
```

## Test Pattern
See: Create new test files as `*_test.go` next to source

```go
package handlers

import (
    "testing"

    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/assert"
)

func TestRegister_WeakPassword(t *testing.T) {
    app := fiber.New()
    handler := NewAuthHandler(mockUserService, false)

    app.Post("/api/v1/auth/register", handler.Register)

    req := &RegisterRequest{Email: "test@example.com", Password: "short"}
    body, _ := json.Marshal(req)

    resp, err := app.Test(httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body)))
    assert.NoError(t, err)
    assert.Equal(t, 400, resp.StatusCode)
}
```

## Dependency Injection Pattern
See: `backend/cmd/server/main.go`

```go
// Service layer gets interface, not concrete type
userSvc := auth.NewUserService(db, tokenSvc)

// Handler gets service, creates its own dependencies
authHandler := handlers.NewAuthHandler(userSvc, cfg.CookieSecure)
```

## Error Response Format
All errors follow this structure:

```json
{
    "error": "Human-readable message",
    "code": "ERR_CATEGORY_SPECIFIC",
    "details": {}
}
```

Always reference before creating similar.
