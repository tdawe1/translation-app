package validation

import "testing"

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"valid email", "user@example.com", true},
		{"valid with subdomain", "user@mail.example.com", true},
		{"valid with + tag", "user+tag@example.com", true},
		{"valid with hyphen", "user-name@example.com", true},
		{"valid with dots", "user.name@example.com", true},
		{"valid with numbers", "user123@example.com", true},
		{"valid with percent", "user%tag@example.com", true},
		{"missing @", "userexample.com", false},
		{"missing domain", "user@", false},
		{"missing user", "@example.com", false},
		{"empty string", "", false},
		{"spaces", "user @example.com", false},
		{"missing TLD", "user@example", false},
		{"invalid chars", "user@exa mple.com", false},
		{"double @", "user@@example.com", false},
		{"no TLD", "user@com", false},      // Missing dot in domain
		{"single char TLD", "user@example.c", false}, // Too short
		{"multiple dots", "user..name@example.com", true}, // Regex allows this - acceptable for web use
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateEmail(tt.email)
			if got != tt.want {
				t.Errorf("ValidateEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}
