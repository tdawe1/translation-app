// Package auth provides authentication and authorization services
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenService handles JWT token generation and validation
type TokenService struct {
	secret      []byte
	accessTTL   time.Duration
	issuer      string
}

// TokenClaims represents the JWT claims structure
type TokenClaims struct {
	UserID string `json:"sub"`
	Type   string `json:"type"`
	jwt.RegisteredClaims
}

// NewTokenService creates a new token service
func NewTokenService(secret string) *TokenService {
	return &TokenService{
		secret:    []byte(secret),
		accessTTL: 15 * time.Minute,
		issuer:    "gengowatcher-saas",
	}
}

// GenerateAccessToken creates a new access token for a user
func (s *TokenService) GenerateAccessToken(userID uuid.UUID) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": now.Add(s.accessTTL).Unix(),
		"iat": now.Unix(),
		"type": "access",
		"iss": s.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateToken validates a JWT token and returns the claims
func (s *TokenService) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, _ := claims["sub"].(string)
		tokenType, _ := claims["type"].(string)

		return &TokenClaims{
			UserID: userID,
			Type:   tokenType,
		}, nil
	}

	return nil, jwt.ErrTokenInvalidClaims
}

// ExtractUserID extracts the user ID from a token string
func (s *TokenService) ExtractUserID(tokenString string) (uuid.UUID, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return uuid.Nil, err
	}

	return uuid.Parse(claims.UserID)
}
