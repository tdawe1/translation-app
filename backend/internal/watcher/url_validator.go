package watcher

import (
	"net"
	"net/url"
	"strings"
)

// URLValidator validates URLs to prevent SSRF (Server-Side Request Forgery) attacks.
// P0-5 FIX: Blocks requests to internal networks, localhost, and non-HTTP(S) protocols.
type URLValidator struct {
	// Allowed schemes (protocols)
	allowedSchemes map[string]bool

	// Whether to allow private IP ranges
	allowPrivateIPs bool

	// Whether to allow localhost
	allowLocalhost bool
}

// NewURLValidator creates a new URL validator with secure defaults.
// By default, only allows HTTP/HTTPS to public IPs.
func NewURLValidator() *URLValidator {
	return &URLValidator{
		allowedSchemes: map[string]bool{
			"http":  true,
			"https": true,
		},
		allowPrivateIPs: false,
		allowLocalhost:   false,
	}
}

// NewPermissiveURLValidator creates a validator that allows more URLs (for testing).
func NewPermissiveURLValidator() *URLValidator {
	return &URLValidator{
		allowedSchemes: map[string]bool{
			"http":  true,
			"https": true,
		},
		allowPrivateIPs: true,
		allowLocalhost:   true,
	}
}

// Validate checks if a URL is safe to fetch.
// Returns an error if the URL is invalid or points to a blocked location.
func (v *URLValidator) Validate(rawURL string) error {
	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return &ValidationError{URL: rawURL, Reason: "invalid URL syntax: " + err.Error()}
	}

	// Check scheme (protocol)
	if !v.allowedSchemes[parsedURL.Scheme] {
		return &ValidationError{
			URL:    rawURL,
			Reason: "protocol not allowed (only http/https allowed)",
		}
	}

	// Check for hostname
	if parsedURL.Hostname() == "" {
		return &ValidationError{URL: rawURL, Reason: "missing hostname"}
	}

	// Resolve hostname to IP addresses
	hosts, err := net.LookupHost(parsedURL.Hostname())
	if err != nil {
		return &ValidationError{URL: rawURL, Reason: "DNS resolution failed: " + err.Error()}
	}

	// Check each resolved IP
	for _, ip := range hosts {
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			// Might be a hostname that couldn't be resolved to IP yet
			// Continue checking other IPs
			continue
		}

		// Block localhost equivalents
		if v.isLocalhost(parsedIP) && !v.allowLocalhost {
			return &ValidationError{URL: rawURL, Reason: "localhost IP not allowed"}
		}

		// Block private IP ranges
		if v.isPrivateIP(parsedIP) && !v.allowPrivateIPs {
			return &ValidationError{URL: rawURL, Reason: "private IP range not allowed"}
		}
	}

	return nil
}

// isLocalhost checks if an IP is a localhost address (127.0.0.0/8 or ::1)
func (v *URLValidator) isLocalhost(ip net.IP) bool {
	return ip.IsLoopback()
}

// isPrivateIP checks if an IP is in a private range (RFC 1918, RFC 4193, link-local)
func (v *URLValidator) isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",        // RFC 1918 - Private Class A
		"172.16.0.0/12",     // RFC 1918 - Private Class B
		"192.168.0.0/16",    // RFC 1918 - Private Class C
		"169.254.0.0/16",    // RFC 3927 - Link-local
		"fc00::/7",          // RFC 4193 - Unique local addresses (IPv6)
		"fe80::/10",         // RFC 4291 - Link-local addresses (IPv6)
		"::1/128",           // IPv6 loopback (covered by IsLoopback but checking explicitly)
	}

	for _, cidr := range privateRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// ValidateAndParse validates a URL and returns the parsed URL if valid.
func (v *URLValidator) ValidateAndParse(rawURL string) (*url.URL, error) {
	if err := v.Validate(rawURL); err != nil {
		return nil, err
	}
	return url.Parse(rawURL)
}

// ValidationError represents a URL validation error.
type ValidationError struct {
	URL    string
	Reason string
}

func (e *ValidationError) Error() string {
	return "URL validation failed for '" + e.URL + "': " + e.Reason
}

// IsPublicURL checks if a URL points to a public internet resource.
// This is a convenience function for common use cases.
func IsPublicURL(rawURL string) bool {
	validator := NewURLValidator()
	return validator.Validate(rawURL) == nil
}

// ContainsSuspiciousPatterns checks for URL patterns that may indicate SSRF attempts.
// Returns true if the URL contains suspicious patterns.
func ContainsSuspiciousPatterns(rawURL string) bool {
	suspicious := []string{
		"@localhost",             // User@localhost
		"@127.",                  // User@127.x.x.x
		"@0.",                    // User@0.0.0.0
		"@169.254",               // Link-local
		"file://",                // Local file access
		"gopher://",              // Alternative protocol
		"ftp://",                 // FTP (often unintended)
	}

	lowerURL := strings.ToLower(rawURL)
	for _, pattern := range suspicious {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}

	// Check for hostname starting with dot (relative domain trick)
	parsed, err := url.Parse(rawURL)
	if err == nil {
		host := parsed.Hostname()
		if strings.HasPrefix(host, ".") {
			return true
		}
	}

	// Check for username in URL (potential credential injection)
	if parsed != nil && parsed.User != nil {
		// Unless it's a well-known public feed that uses auth
		if !isKnownPublicFeed(parsed.Host) {
			return true
		}
	}

	return false
}

// isKnownPublicFeed checks if a host is a known public RSS feed provider
// that uses authentication (rare, but some do)
func isKnownPublicFeed(host string) bool {
	knownFeeds := []string{
		".gengo.com",
		".upwork.com",
		".fiverr.com",
	}

	lowerHost := strings.ToLower(host)
	for _, feed := range knownFeeds {
		if strings.Contains(lowerHost, feed) {
			return true
		}
	}
	return false
}
