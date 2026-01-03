// Package password provides password hashing utilities
package password

import (
	"crypto/rand"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrCryptoFailed = errors.New("failed to generate secure random value")
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// VerifyPassword verifies a password against a hash
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomPassword generates a cryptographically secure random password of specified length
func GenerateRandomPassword(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", ErrCryptoFailed
	}
	// Encode to hex and double the length, then trim to requested length
	return hex.EncodeToString(b)[:length], nil
}
