package tests

import (
	"testing"

	"github.com/tdawe1/translation-app/internal/watcher"
)

// TestURLValidator_PublicURLs accepts valid public URLs
func TestURLValidator_PublicURLs(t *testing.T) {
	validator := watcher.NewURLValidator()

	validURLs := []string{
		"https://gengo.com/rss",
		"https://www.google.com",
		"https://github.com",
		"https://example.com/feed.xml",
	}

	for _, url := range validURLs {
		t.Run(url, func(t *testing.T) {
			if err := validator.Validate(url); err != nil {
				t.Errorf("Public URL should be valid: %s, error: %v", url, err)
			}
		})
	}
}

// TestURLValidator_RejectsLocalhost rejects localhost URLs
func TestURLValidator_RejectsLocalhost(t *testing.T) {
	validator := watcher.NewURLValidator()

	localhostURLs := []string{
		"http://localhost/feed.xml",
		"http://127.0.0.1/rss",
		"http://127.1.1.1/feed",
		"http://[::1]/rss",
	}

	for _, url := range localhostURLs {
		t.Run(url, func(t *testing.T) {
			if err := validator.Validate(url); err == nil {
				t.Errorf("Localhost URL should be rejected: %s", url)
			}
		})
	}
}

// TestURLValidator_RejectsPrivateIPs rejects private IP ranges
func TestURLValidator_RejectsPrivateIPs(t *testing.T) {
	validator := watcher.NewURLValidator()

	privateURLs := []string{
		"http://10.0.0.1/feed.xml",      // RFC 1918 Class A
		"http://10.255.255.255/rss",      // RFC 1918 Class A
		"http://172.16.0.1/feed.xml",     // RFC 1918 Class B
		"http://172.31.255.255/rss",      // RFC 1918 Class B
		"http://192.168.0.1/feed.xml",    // RFC 1918 Class C
		"http://192.168.255.255/rss",     // RFC 1918 Class C
		"http://169.254.1.1/feed.xml",    // Link-local
		"http://fc00::1/feed.xml",        // IPv6 unique local
		"http://fe80::1/rss",             // IPv6 link-local
	}

	for _, url := range privateURLs {
		t.Run(url, func(t *testing.T) {
			if err := validator.Validate(url); err == nil {
				t.Errorf("Private IP URL should be rejected: %s", url)
			}
		})
	}
}

// TestURLValidator_RejectsInvalidSchemes rejects non-HTTP protocols
func TestURLValidator_RejectsInvalidSchemes(t *testing.T) {
	validator := watcher.NewURLValidator()

	invalidSchemes := []string{
		"file:///etc/passwd",
		"ftp://example.com/feed.xml",
		"gopher://localhost:70/feed",
		"//example.com/feed.xml", // protocol-relative URL
	}

	for _, url := range invalidSchemes {
		t.Run(url, func(t *testing.T) {
			if err := validator.Validate(url); err == nil {
				t.Errorf("Non-HTTP scheme should be rejected: %s", url)
			}
		})
	}
}

// TestURLValidator_PermissiveMode allows private IPs in permissive mode
func TestURLValidator_PermissiveMode(t *testing.T) {
	validator := watcher.NewPermissiveURLValidator()

	// These should pass in permissive mode
	allowedURLs := []string{
		"http://localhost:8000/feed.xml",
		"http://127.0.0.1/rss",
		"http://10.0.0.1/feed.xml",
		"http://192.168.1.1/rss",
	}

	for _, url := range allowedURLs {
		t.Run(url, func(t *testing.T) {
			if err := validator.Validate(url); err != nil {
				t.Errorf("Permissive mode should allow: %s, error: %v", url, err)
			}
		})
	}

	// But file:// should still be blocked
	fileURL := "file:///etc/passwd"
	if err := validator.Validate(fileURL); err == nil {
		t.Errorf("Permissive mode should still reject file:// URLs")
	}
}

// TestContainsSuspiciousPatterns detects SSRF attempt patterns
func TestContainsSuspiciousPatterns(t *testing.T) {
	suspiciousURLs := []struct {
		url      string
		expected bool
	}{
		{"http://user@localhost/feed", true},
		{"http://admin@127.0.0.1/rss", true},
		{"http://test@0.0.0.0/feed", true},
		{"http://user@169.254.1.1/api", true},
		{"file:///etc/passwd", true},
		{"gopher://localhost/feed", true},
		{"ftp://192.168.1.1/feed", true},
		{"http://.example.com@", true}, // relative domain trick
		{"https://gengo.com/rss", false}, // legitimate URL
		{"http://example.com/feed.xml", false},
	}

	for _, tc := range suspiciousURLs {
		t.Run(tc.url, func(t *testing.T) {
			result := watcher.ContainsSuspiciousPatterns(tc.url)
			if result != tc.expected {
				t.Errorf("ContainsSuspiciousPatterns(%s) = %v, want %v",
					tc.url, result, tc.expected)
			}
		})
	}
}

// TestURLValidator_RejectsInvalidURLs rejects malformed URLs
func TestURLValidator_RejectsInvalidURLs(t *testing.T) {
	validator := watcher.NewURLValidator()

	invalidURLs := []string{
		"",
		"not a url",
		"http://",
		"://invalid",
		"ht!tp://example.com",
	}

	for _, url := range invalidURLs {
		t.Run(url, func(t *testing.T) {
			if err := validator.Validate(url); err == nil {
				t.Errorf("Invalid URL should be rejected: %s", url)
			}
		})
	}
}

// TestURLValidator_ValidateAndParse returns parsed URL when valid
func TestURLValidator_ValidateAndParse(t *testing.T) {
	validator := watcher.NewURLValidator()

	testURL := "https://gengo.com/rss"
	parsed, err := validator.ValidateAndParse(testURL)
	if err != nil {
		t.Fatalf("Valid URL should pass ValidateAndParse: %v", err)
	}

	if parsed.Scheme != "https" {
		t.Errorf("Scheme = %q, want 'https'", parsed.Scheme)
	}
	if parsed.Host != "gengo.com" {
		t.Errorf("Host = %q, want 'gengo.com'", parsed.Host)
	}
}
