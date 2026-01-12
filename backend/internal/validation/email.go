package validation

import (
	"regexp"
)

var (
	// emailRegex validates email addresses according to RFC 5322
	// Simplified but practical for web applications
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ValidateEmail checks if the email format is valid
func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}
	return emailRegex.MatchString(email)
}
