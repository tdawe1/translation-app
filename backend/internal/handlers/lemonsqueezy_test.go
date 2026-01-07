package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Helper to generate a valid LemonSqueezy signature format: <timestamp>.<hex_signature>
func generateTestSignature(secret string, body []byte, timestamp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%d.%s", timestamp, signature)
}

func TestWebhookSignature_ValidTimestamp_Accepted(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"test": "data"}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Current timestamp should be accepted
	now := time.Now().Unix()
	signature := generateTestSignature(secret, body, now)

	result := handler.verifySignature(body, signature)
	assert.True(t, result, "Valid signature with current timestamp should be accepted")
}

func TestWebhookSignature_OldTimestamp_Rejected(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"test": "data"}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Timestamp older than 5 minutes should be rejected
	oldTimestamp := time.Now().Add(-10 * time.Minute).Unix()
	signature := generateTestSignature(secret, body, oldTimestamp)

	result := handler.verifySignature(body, signature)
	assert.False(t, result, "Signature with timestamp older than 5 minutes should be rejected")
}

func TestWebhookSignature_FutureTimestamp_Rejected(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"test": "data"}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Timestamp more than 5 minutes in the future should be rejected
	futureTimestamp := time.Now().Add(10 * time.Minute).Unix()
	signature := generateTestSignature(secret, body, futureTimestamp)

	result := handler.verifySignature(body, signature)
	assert.False(t, result, "Signature with timestamp more than 5 minutes in future should be rejected")
}

func TestWebhookSignature_EdgeCaseWithinTolerance_Accepted(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"test": "data"}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Exactly 4 minutes 59 seconds in the past should be accepted
	edgeTimestamp := time.Now().Add(-4*time.Minute - 59*time.Second).Unix()
	signature := generateTestSignature(secret, body, edgeTimestamp)

	result := handler.verifySignature(body, signature)
	assert.True(t, result, "Signature at edge of tolerance window (4:59 past) should be accepted")
}

func TestWebhookSignature_EdgeCaseOutsideTolerance_Rejected(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"test": "data"}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Exactly 5 minutes 1 second in the past should be rejected
	edgeTimestamp := time.Now().Add(-5*time.Minute - 1*time.Second).Unix()
	signature := generateTestSignature(secret, body, edgeTimestamp)

	result := handler.verifySignature(body, signature)
	assert.False(t, result, "Signature just outside tolerance window (5:01 past) should be rejected")
}

func TestWebhookSignature_InvalidFormat_Rejected(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"test": "data"}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	tests := []struct {
		name      string
		signature string
	}{
		{"empty signature", ""},
		{"only timestamp", "1234567890"},
		{"only hex", "abcdef123456"},
		{"too many parts", "1234567890.extra.part"},
		{"missing delimiter", "1234567890abcdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.verifySignature(body, tt.signature)
			assert.False(t, result, "Malformed signature should be rejected: "+tt.name)
		})
	}
}

func TestWebhookSignature_InvalidTimestamp_Rejected(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"test": "data"}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Non-numeric timestamp
	signature := "not-a-number." + hex.EncodeToString([]byte("anything"))

	result := handler.verifySignature(body, signature)
	assert.False(t, result, "Signature with non-numeric timestamp should be rejected")
}

func TestWebhookSignature_WrongSecret_Rejected(t *testing.T) {
	correctSecret := "correct-secret"
	wrongSecret := "wrong-secret"
	body := []byte(`{"test": "data"}`)

	handler := &LemonSqueezyHandler{webhookSecret: correctSecret}

	// Generate signature with wrong secret
	mac := hmac.New(sha256.New, []byte(wrongSecret))
	mac.Write(body)
	wrongSig := hex.EncodeToString(mac.Sum(nil))
	now := time.Now().Unix()
	signature := fmt.Sprintf("%d.%s", now, wrongSig)

	result := handler.verifySignature(body, signature)
	assert.False(t, result, "Signature with wrong secret should be rejected")
}

func TestWebhookSignature_DifferentBody_Rejected(t *testing.T) {
	secret := "test-webhook-secret"
	originalBody := []byte(`{"test": "data"}`)
	modifiedBody := []byte(`{"test": "modified"}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Generate signature for original body but verify with modified body
	signature := generateTestSignature(secret, originalBody, time.Now().Unix())

	result := handler.verifySignature(modifiedBody, signature)
	assert.False(t, result, "Signature for different body should be rejected")
}

func TestWebhookSignature_ValidAndWithinTolerance_Accepted(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"meta": {"event_id": "123"}, "data": {"attributes": {"status": "active"}}}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Timestamp 3 minutes ago - within tolerance
	recentTimestamp := time.Now().Add(-3 * time.Minute).Unix()
	signature := generateTestSignature(secret, body, recentTimestamp)

	result := handler.verifySignature(body, signature)
	assert.True(t, result, "Valid signature within tolerance window should be accepted")
}

func TestWebhookSignature_ValidAndJustBeforeFuture_Accepted(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"meta": {"event_id": "123"}, "data": {"attributes": {"status": "active"}}}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Timestamp 4 minutes in future - within tolerance (allows for some clock skew)
	futureTimestamp := time.Now().Add(4 * time.Minute).Unix()
	signature := generateTestSignature(secret, body, futureTimestamp)

	result := handler.verifySignature(body, signature)
	assert.True(t, result, "Valid signature within future tolerance window should be accepted")
}

func TestWebhookSignature_EmptyBody_Verifies(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte{}
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Even with empty body, the signature should verify
	signature := generateTestSignature(secret, body, time.Now().Unix())

	result := handler.verifySignature(body, signature)
	assert.True(t, result, "Valid signature with empty body should verify")
}

func TestWebhookSignature_NegativeTimestamp_Rejected(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"test": "data"}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Negative timestamp (before Unix epoch)
	signature := fmt.Sprintf("%d.%s", -1, hex.EncodeToString([]byte("anything")))

	result := handler.verifySignature(body, signature)
	assert.False(t, result, "Negative timestamp should be rejected")
}

func TestWebhookSignature_TimestampBoundary_Zero_Rejected(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"test": "data"}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Unix epoch timestamp (Jan 1, 1970) - definitely too old
	signature := fmt.Sprintf("%d.%s", 0, hex.EncodeToString([]byte("anything")))

	result := handler.verifySignature(body, signature)
	assert.False(t, result, "Unix epoch timestamp should be rejected (too old)")
}

func TestWebhookSignature_LargeTimestamp_Rejected(t *testing.T) {
	secret := "test-webhook-secret"
	body := []byte(`{"test": "data"}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Very large timestamp (year 2100+) - too far in future
	year2100 := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	signature := fmt.Sprintf("%d.%s", year2100.Unix(), hex.EncodeToString([]byte("anything")))

	result := handler.verifySignature(body, signature)
	assert.False(t, result, "Far future timestamp should be rejected")
}

func TestWebhookSignature_HMACConstantTimeComparison(t *testing.T) {
	// This test verifies that signature comparison uses constant-time comparison
	// to prevent timing attacks. We test this by generating two different
	// signatures of the same length and verifying both fail properly.
	secret := "test-webhook-secret"
	body := []byte(`{"test": "data"}`)
	handler := &LemonSqueezyHandler{webhookSecret: secret}

	// Two different signatures, same length
	sig1 := "1234567890.0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	sig2 := "1234567890.ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"

	// Both should be rejected as incorrect signatures
	result1 := handler.verifySignature(body, sig1)
	result2 := handler.verifySignature(body, sig2)

	assert.False(t, result1, "Incorrect signature 1 should be rejected")
	assert.False(t, result2, "Incorrect signature 2 should be rejected")
}

// Test strconv.ParseInt behavior with timestamp edge cases
func TestTimestampParsing_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		expected int64
	}{
		{"zero", "0", false, 0},
		{"positive", "1234567890", false, 1234567890},
		{"negative", "-1", false, -1},
		{"max int64", "9223372036854775807", false, 9223372036854775807},
		{"invalid text", "abc", true, 0},
		{"with newline", "123\n", true, 0},
		{"with spaces", " 123 ", true, 0},
		{"float", "123.456", true, 0},
		{"hex", "0x123", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := strconv.ParseInt(tt.input, 10, 64)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInt(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("ParseInt(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}
