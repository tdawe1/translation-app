package password

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"normal password", "MySecureP@ssw0rd!", false},
		{"short password", "short", false},
		{"empty password", "", false},
		{"long password", strings.Repeat("a", 72), false}, // bcrypt max
		{"unicode password", "пароль密码🔐", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && hash == "" {
				t.Error("HashPassword() returned empty hash")
			}
			// Verify it's a valid bcrypt hash
			if !tt.wantErr {
				cost, err := bcrypt.Cost([]byte(hash))
				if err != nil {
					t.Errorf("Invalid bcrypt hash: %v", err)
				}
				if cost != BcryptCost {
					t.Errorf("bcrypt cost = %d, want %d", cost, BcryptCost)
				}
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "TestPassword123!"
	hash, _ := HashPassword(password)

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{"correct password", password, hash, true},
		{"wrong password", "WrongPassword", hash, false},
		{"empty password", "", hash, false},
		{"invalid hash", password, "invalid", false},
		{"empty hash", password, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyPassword(tt.password, tt.hash); got != tt.want {
				t.Errorf("VerifyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"length 8", 8, false},
		{"length 16", 16, false},
		{"length 32", 32, false},
		{"length 0", 0, true},
		{"negative length", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateRandomPassword(tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRandomPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.length {
				t.Errorf("GenerateRandomPassword() length = %d, want %d", len(got), tt.length)
			}
		})
	}

	// Test randomness - generate multiple and ensure they're different
	t.Run("randomness", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			pw, _ := GenerateRandomPassword(16)
			if seen[pw] {
				t.Error("GenerateRandomPassword() produced duplicate")
			}
			seen[pw] = true
		}
	})
}

func TestSecureCompare(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"equal strings", "abc123", "abc123", true},
		{"different strings", "abc123", "xyz789", false},
		{"different lengths", "short", "longer", false},
		{"empty strings", "", "", true},
		{"one empty", "abc", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SecureCompare(tt.a, tt.b); got != tt.want {
				t.Errorf("SecureCompare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateSecureToken(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"length 16", 16},
		{"length 32", 32},
		{"length 64", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateSecureToken(tt.length)
			if err != nil {
				t.Errorf("GenerateSecureToken() error = %v", err)
				return
			}
			// Hex encoding doubles the length
			expectedLen := tt.length * 2
			if len(token) != expectedLen {
				t.Errorf("GenerateSecureToken() length = %d, want %d", len(token), expectedLen)
			}
		})
	}

	// Test uniqueness
	t.Run("uniqueness", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 100; i++ {
			token, _ := GenerateSecureToken(32)
			if seen[token] {
				t.Error("GenerateSecureToken() produced duplicate")
			}
			seen[token] = true
		}
	})
}

func TestValidateStrength_AllRequirements_Passes(t *testing.T) {
	// Password with 12+ chars, upper, lower, digit, special should pass
	password := "SecureP@ssw0rd123"
	err := ValidateStrength(password)
	if err != nil {
		t.Errorf("ValidateStrength() valid password failed: %v", err)
	}
}

func TestValidateStrength_TooShort_ReturnsError(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"empty", ""},
		{"1 char", "a"},
		{"5 chars", "abcDE"},
		{"11 chars", "abcDEFG1234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStrength(tt.password)
			if err != ErrPasswordTooShort {
				t.Errorf("ValidateStrength() expected ErrPasswordTooShort, got %v", err)
			}
		})
	}
}

func TestValidateStrength_MissingUpper_ReturnsError(t *testing.T) {
	// 12+ chars, no uppercase
	password := "securep@ssw0rd"
	err := ValidateStrength(password)
	if err != ErrPasswordMissingUpper {
		t.Errorf("ValidateStrength() expected ErrPasswordMissingUpper, got %v", err)
	}
}

func TestValidateStrength_MissingLower_ReturnsError(t *testing.T) {
	// 12+ chars, no lowercase
	password := "SECUREP@SSW0RD"
	err := ValidateStrength(password)
	if err != ErrPasswordMissingLower {
		t.Errorf("ValidateStrength() expected ErrPasswordMissingLower, got %v", err)
	}
}

func TestValidateStrength_MissingDigit_ReturnsError(t *testing.T) {
	// 12+ chars, no digit
	password := "SecurePassword!"
	err := ValidateStrength(password)
	if err != ErrPasswordMissingDigit {
		t.Errorf("ValidateStrength() expected ErrPasswordMissingDigit, got %v", err)
	}
}

func TestValidateStrength_MissingSpecial_ReturnsError(t *testing.T) {
	// 12+ chars, no special character
	password := "SecurePassword123"
	err := ValidateStrength(password)
	if err != ErrPasswordMissingSpecial {
		t.Errorf("ValidateStrength() expected ErrPasswordMissingSpecial, got %v", err)
	}
}

func TestValidateStrength_UnicodeSpecial_Passes(t *testing.T) {
	// Unicode symbols should count as special characters
	password := "SecurePassw0rd•" // Bullet symbol
	err := ValidateStrength(password)
	if err != nil {
		t.Errorf("ValidateStrength() unicode special char failed: %v", err)
	}
}

func TestBcryptCost(t *testing.T) {
	// Verify bcrypt cost is at least 12 (OWASP recommendation)
	if BcryptCost < 12 {
		t.Errorf("BcryptCost = %d, want >= 12", BcryptCost)
	}
}

// Benchmark to ensure hashing time is reasonable
func BenchmarkHashPassword(b *testing.B) {
	password := "BenchmarkPassword123!"
	for i := 0; i < b.N; i++ {
		HashPassword(password)
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := "BenchmarkPassword123!"
	hash, _ := HashPassword(password)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyPassword(password, hash)
	}
}
