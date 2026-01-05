// Package email provides email sending functionality using Resend
package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Service handles email operations
type Service struct {
	apiKey     string
	fromEmail  string
	fromName   string
	baseURL    string // Frontend URL for verification links
	httpClient *http.Client
}

// Config holds email configuration
type Config struct {
	APIKey    string
	FromEmail string
	FromName  string
	BaseURL   string
}

// NewService creates a new email service with connection pooling (#005 fix)
func NewService(cfg *Config) *Service {
	return &Service{
		apiKey:    cfg.APIKey,
		fromEmail: cfg.FromEmail,
		fromName:  cfg.FromName,
		baseURL:   cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// IsEnabled returns true if the email service is configured
func (s *Service) IsEnabled() bool {
	return s.apiKey != ""
}

// SendMagicLink sends a magic link email (alias for SendMagicLinkEmail)
func (s *Service) SendMagicLink(email, token string) error {
	return s.SendMagicLinkEmail(email, token)
}

// NewTestService creates a test email service that logs instead of sending
func NewTestService(cfg *Config) *Service {
	return &Service{
		apiKey:    "", // Empty key = logging only
		fromEmail: cfg.FromEmail,
		fromName:  cfg.FromName,
		baseURL:   cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// resendRequest represents a Resend API request
type resendRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// resendResponse represents a Resend API response
type resendResponse struct {
	ID string `json:"id"`
}

// SendEmail sends an email via Resend API
func (s *Service) SendEmail(to, subject, htmlContent string) error {
	if s.apiKey == "" {
		// In development without API key, log and return success
		fmt.Printf("[EMAIL] Would send to %s: %s\n", to, subject)
		return nil
	}

	reqBody := resendRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{to},
		Subject: subject,
		HTML:    htmlContent,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("email API returned status %d", resp.StatusCode)
	}

	return nil
}

// SendVerificationEmail sends an email verification link
func (s *Service) SendVerificationEmail(email, token string) error {
	subject := "Verify your email address"
	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", s.baseURL, token)

	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.button { display: inline-block; padding: 12px 24px; background: #171717; color: #fff; text-decoration: none; border-radius: 4px; }
		.footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #e5e5e5; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h2>Verify your email address</h2>
		<p>Thanks for signing up for GengoWatcher SaaS. Please click the button below to verify your email address:</p>
		<p><a href="%s" class="button">Verify Email</a></p>
		<p>Or copy and paste this link into your browser:</p>
		<p><code style="background: #f5f5f5; padding: 2px 6px; border-radius: 3px;">%s</code></p>
		<p>This link will expire in 24 hours.</p>
		<div class="footer">
			<p>If you didn't create an account, you can safely ignore this email.</p>
		</div>
	</div>
</body>
</html>`, verifyURL, verifyURL)

	return s.SendEmail(email, subject, htmlContent)
}

// SendMagicLinkEmail sends a magic link for passwordless authentication
func (s *Service) SendMagicLinkEmail(email, token string) error {
	subject := "Sign in to GengoWatcher SaaS"
	loginURL := fmt.Sprintf("%s/auth/magic-login?token=%s", s.baseURL, token)

	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.button { display: inline-block; padding: 12px 24px; background: #171717; color: #fff; text-decoration: none; border-radius: 4px; }
		.footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #e5e5e5; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h2>Sign in to GengoWatcher SaaS</h2>
		<p>Click the button below to sign in to your account:</p>
		<p><a href="%s" class="button">Sign In</a></p>
		<p>Or copy and paste this link into your browser:</p>
		<p><code style="background: #f5f5f5; padding: 2px 6px; border-radius: 3px;">%s</code></p>
		<p>This link will expire in 15 minutes.</p>
		<div class="footer">
			<p>If you didn't request this email, you can safely ignore it.</p>
		</div>
	</div>
</body>
</html>`, loginURL, loginURL)

	return s.SendEmail(email, subject, htmlContent)
}

// SendPasswordResetEmail sends a password reset link
func (s *Service) SendPasswordResetEmail(email, token string) error {
	subject := "Reset your password"
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", s.baseURL, token)

	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
		.container { max-width: 600px; margin: 0 auto; padding: 20px; }
		.button { display: inline-block; padding: 12px 24px; background: #171717; color: #fff; text-decoration: none; border-radius: 4px; }
		.footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #e5e5e5; font-size: 12px; color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h2>Reset your password</h2>
		<p>Click the button below to reset your password:</p>
		<p><a href="%s" class="button">Reset Password</a></p>
		<p>Or copy and paste this link into your browser:</p>
		<p><code style="background: #f5f5f5; padding: 2px 6px; border-radius: 3px;">%s</code></p>
		<p>This link will expire in 1 hour.</p>
		<div class="footer">
			<p>If you didn't request a password reset, you can safely ignore this email.</p>
		</div>
	</div>
</body>
</html>`, resetURL, resetURL)

	return s.SendEmail(email, subject, htmlContent)
}
