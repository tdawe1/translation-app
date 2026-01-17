// Package auth provides authentication and authorization services
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenService handles JWT token generation and validation
type TokenService struct {
	secret    []byte
	accessTTL time.Duration
	issuer    string
}

type TokenClaims struct {
	UserID           string
	JTI              string
	Role             string
	Type             string
	ExpiresAt        jwt.NumericDate
	RegisteredClaims jwt.RegisteredClaims
}

// NewTokenService creates a new token service
func NewTokenService(secret string) *TokenService {
	return &TokenService{
		secret:    []byte(secret),
		accessTTL: 15 * time.Minute,
		issuer:    "gengowatcher-saas",
	}
}

func (s *TokenService) GenerateAccessToken(userID uuid.UUID, role string) (string, error) {
	now := time.Now()
	jti := uuid.New().String()

	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"jti":  jti,
		"role": role,
		"exp":  now.Add(s.accessTTL).Unix(),
		"iat":  now.Unix(),
		"type": "access",
		"iss":  s.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *TokenService) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
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
		jti, _ := claims["jti"].(string)
		role, _ := claims["role"].(string)
		tokenType, _ := claims["type"].(string)
		var exp *jwt.NumericDate
		if expFloat, ok := claims["exp"].(float64); ok {
			expUnix := int64(expFloat)
			exp = &jwt.NumericDate{Time: time.Unix(expUnix, 0)}
		}

		return &TokenClaims{
			UserID:    userID,
			JTI:       jti,
			Role:      role,
			Type:      tokenType,
			ExpiresAt: *exp,
		}, nil
	}

	return nil, jwt.ErrTokenInvalidClaims
}

func (s *TokenService) ExtractUserID(tokenString string) (uuid.UUID, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return uuid.Nil, err
	}

	return uuid.Parse(claims.UserID)
}
