// Package password provides password hashing utilities
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost is the cost factor for bcrypt hashing (#011 fix - increased from 10 to 12)
// OWASP recommends minimum 12 for 2024+
const BcryptCost = 12

var (
	ErrCryptoFailed = errors.New("failed to generate secure random value")

	// Password strength validation errors (P2 fix)
	ErrPasswordTooShort     = errors.New("password must be at least 12 characters")
	ErrPasswordMissingUpper  = errors.New("password must contain an uppercase letter")
	ErrPasswordMissingLower  = errors.New("password must contain a lowercase letter")
	ErrPasswordMissingDigit  = errors.New("password must contain a digit")
	ErrPasswordMissingSpecial = errors.New("password must contain a special character")
)

// HashPassword hashes a password using bcrypt with secure cost factor
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// ValidateStrength validates password strength requirements (P2 fix)
// Enforces: minimum 12 chars, uppercase, lowercase, digit, and special character
func ValidateStrength(password string) error {
	if len(password) < 12 {
		return ErrPasswordTooShort
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return ErrPasswordMissingUpper
	}
	if !hasLower {
		return ErrPasswordMissingLower
	}
	if !hasDigit {
		return ErrPasswordMissingDigit
	}
	if !hasSpecial {
		return ErrPasswordMissingSpecial
	}
	return nil
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomPassword generates a cryptographically secure random password of specified length (#018 fix)
func GenerateRandomPassword(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("password length must be positive")
	}
	// Generate enough random bytes to ensure we have enough characters after hex encoding
	// Hex encoding doubles the length, so we need at least (length+1)/2 bytes
	byteLength := (length + 1) / 2
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", ErrCryptoFailed
	}
	// Encode to hex and trim to requested length
	return hex.EncodeToString(b)[:length], nil
}

// SecureCompare performs constant-time comparison of two strings (#019 fix)
// This prevents timing attacks when comparing tokens
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// GenerateSecureToken generates a cryptographically secure random token (#013 fix - shared helper)
func GenerateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", ErrCryptoFailed
	}
	return hex.EncodeToString(b), nil
}
