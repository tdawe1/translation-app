package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRequest is a mock request struct for testing ApplyPartialUpdate
type TestRequest struct {
	StringField string
	IntField    *int
	BoolField   *bool
	FloatField  *float64
	IgnoreField string // Should not update if empty
}

// TestModel is a mock model struct for testing
type TestModel struct {
	StringField string
	IntField    int
	BoolField   bool
	FloatField  float64
	IgnoreField string
	UpdatedAt   time.Time
}

func TestApplyPartialUpdate_AllFields(t *testing.T) {
	req := TestRequest{
		StringField: "new value",
		IntField:    intPtr(42),
		BoolField:   boolPtr(true),
		FloatField:  float64Ptr(3.14),
	}

	updates := ApplyPartialUpdate(req)

	// Verify all fields are included
	assert.Equal(t, "new value", updates["string_field"], "StringField should be included")
	assert.Equal(t, 42, updates["int_field"], "IntField should be included")
	assert.Equal(t, true, updates["bool_field"], "BoolField should be included")
	assert.Equal(t, 3.14, updates["float_field"], "FloatField should be included")
}

func TestApplyPartialUpdate_OnlyProvidedFields(t *testing.T) {
	req := TestRequest{
		StringField: "updated",
		// IntField, BoolField, FloatField not provided (nil/zero)
	}

	updates := ApplyPartialUpdate(req)

	// Only StringField should be in updates
	require.Equal(t, 1, len(updates), "Expected exactly 1 update")
	assert.Equal(t, "updated", updates["string_field"], "StringField should be updated")
}

func TestApplyPartialUpdate_SkipEmptyString(t *testing.T) {
	req := TestRequest{
		StringField: "", // Empty string should be skipped
		IntField:    intPtr(10),
	}

	updates := ApplyPartialUpdate(req)

	// Empty StringField should not be in updates
	_, exists := updates["string_field"]
	assert.False(t, exists, "Empty string should be skipped")
	assert.Equal(t, 10, updates["int_field"], "IntField should be included")
}

func TestApplyPartialUpdate_SkipNilPointers(t *testing.T) {
	req := TestRequest{
		StringField: "value",
		IntField:    nil, // nil pointer should be skipped
		BoolField:   nil,
		FloatField:  nil,
	}

	updates := ApplyPartialUpdate(req)

	// Only StringField should be in updates
	assert.Equal(t, "value", updates["string_field"])
	_, hasInt := updates["int_field"]
	_, hasBool := updates["bool_field"]
	_, hasFloat := updates["float_field"]
	assert.False(t, hasInt, "nil IntField should be skipped")
	assert.False(t, hasBool, "nil BoolField should be skipped")
	assert.False(t, hasFloat, "nil FloatField should be skipped")
}

func TestApplyPartialUpdate_WithFalseBool(t *testing.T) {
	// Edge case: false bool should still be included if explicitly set
	req := TestRequest{
		BoolField: boolPtr(false),
	}

	updates := ApplyPartialUpdate(req)

	// false bool should still be included (it's a valid value)
	assert.Equal(t, false, updates["bool_field"], "false bool should be included")
}

func TestApplyPartialUpdate_WithZeroNumeric(t *testing.T) {
	req := struct {
		IntField   *int
		FloatField *float64
	}{
		IntField:   intPtr(0),
		FloatField: float64Ptr(0.0),
	}

	updates := ApplyPartialUpdate(req)

	// Zero values should be included since they're explicitly set via pointers
	assert.Equal(t, 0, updates["int_field"], "zero int should be included when explicitly set")
	assert.Equal(t, 0.0, updates["float_field"], "zero float should be included when explicitly set")
}

// Helper functions for creating pointers to primitives
func intPtr(i int) *int     { return &i }
func boolPtr(b bool) *bool   { return &b }
func float64Ptr(f float64) *float64 { return &f }

// TestCamelToSnake verifies the CamelCase to snake_case conversion
func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"RSSFeedURL", "rss_feed_url"},
		{"WebSocketEnabled", "web_socket_enabled"},
		{"GengoUserID", "gengo_user_id"},
		{"ID", "id"},
		{"MinReward", "min_reward"},
		{"AutoAcceptEnabled", "auto_accept_enabled"},
		{"EnableDesktopNotifs", "enable_desktop_notifs"},
		{"Single", "single"},
		{"AlreadySnake", "already_snake"},
		{"HTTP", "http"},
		{"OAuthAccount", "o_auth_account"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := camelToSnake(tt.input); got != tt.expected {
				t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestApplyPartialUpdate_WithRealWatcherConfig tests with actual watcher config fields
func TestApplyPartialUpdate_WithRealWatcherConfig(t *testing.T) {
	// Simulate a real UpdateConfigRequest from watcher.go
	req := struct {
		RSSFeedURL         string
		MinReward          *float64
		AutoAcceptEnabled  *bool
		MaxReward          *float64
		WebSocketEnabled   *bool
		NotificationEmail  string
	}{
		RSSFeedURL:        "https://example.com/feed",
		MinReward:         float64Ptr(5.50),
		AutoAcceptEnabled: boolPtr(true),
		// MaxReward not provided (nil)
		// WebSocketEnabled not provided (nil)
		// NotificationEmail empty string
	}

	updates := ApplyPartialUpdate(req)

	// Verify expected fields are present
	expectedFields := map[string]interface{}{
		"rss_feed_url":         "https://example.com/feed",
		"min_reward":           5.50,
		"auto_accept_enabled":  true,
	}

	// Should have exactly 3 fields (empty string and nil pointers skipped)
	if len(updates) != 3 {
		t.Errorf("Expected 3 fields, got %d: %v", len(updates), updates)
	}

	for k, v := range expectedFields {
		if updates[k] != v {
			t.Errorf("Field %s: expected %v, got %v", k, v, updates[k])
		}
	}

	// Verify max_reward is not present (nil pointer)
	if _, exists := updates["max_reward"]; exists {
		t.Error("max_reward should not be present (nil pointer)")
	}

	// Verify websocket_enabled is not present (nil pointer)
	if _, exists := updates["websocket_enabled"]; exists {
		t.Error("websocket_enabled should not be present (nil pointer)")
	}

	// Verify notification_email is not present (empty string)
	if _, exists := updates["notification_email"]; exists {
		t.Error("notification_email should not be present (empty string)")
	}
}
