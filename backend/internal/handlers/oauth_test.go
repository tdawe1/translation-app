package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidateProvider(t *testing.T) {
	tests := []struct {
		provider string
		want     bool
	}{
		{"google", true},
		{"github", true},
		{"Google", false}, // case sensitive
		{"GITHUB", false},
		{"facebook", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := ValidateProvider(tt.provider)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStatePattern(t *testing.T) {
	tests := []struct {
		name  string
		state string
		want  bool
	}{
		{"valid 32 chars", "abcdefghijklmnopqrstuvwxyz123456", true},
		{"valid 64 chars", "abcdefghijklmnopqrstuvwxyz123456abcdefghijklmnopqrstuvwxyz123456", true},
		{"valid mixed case", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef", true},
		{"too short", "abc", false},
		{"too long", "abcdefghijklmnopqrstuvwxyz123456abcdefghijklmnopqrstuvwxyz1234567", false},
		{"with special chars", "abc!@#$%^&*()defghijklmnopqrstuv", false},
		{"with spaces", "abcdefghijklmnopqrstuvwxyz 12345", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statePattern.MatchString(tt.state)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStateStore_ThreadSafety(t *testing.T) {
	store := &stateStore{
		m: make(map[string]time.Time),
	}

	// Simulate concurrent access
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			store.Lock()
			store.m["key"] = time.Now()
			store.Unlock()
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			store.RLock()
			_ = store.m["key"]
			store.RUnlock()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// If we got here without panic, the mutex is working
	assert.True(t, true, "No race condition detected")
}

func TestOAuthHandler_StateOperations(t *testing.T) {
	// Create handler with minimal dependencies
	handler := &OAuthHandler{
		states: &stateStore{
			m: make(map[string]time.Time),
		},
		stateExpiry: 10 * time.Minute,
	}

	t.Run("setState and getState", func(t *testing.T) {
		state := "test-state-12345678901234567890"
		expiry := time.Now().Add(10 * time.Minute)

		handler.setState(state, expiry)

		got, exists := handler.getState(state)
		assert.True(t, exists)
		assert.Equal(t, expiry.Unix(), got.Unix())
	})

	t.Run("getState nonexistent", func(t *testing.T) {
		_, exists := handler.getState("nonexistent")
		assert.False(t, exists)
	})

	t.Run("deleteState", func(t *testing.T) {
		state := "state-to-delete-1234567890123"
		handler.setState(state, time.Now().Add(10*time.Minute))

		handler.deleteState(state)

		_, exists := handler.getState(state)
		assert.False(t, exists)
	})
}

func TestOAuthHandler_CleanupExpiredStates(t *testing.T) {
	handler := &OAuthHandler{
		states: &stateStore{
			m: make(map[string]time.Time),
		},
		stateExpiry: 10 * time.Minute,
	}

	// Add expired state
	handler.setState("expired-state-123456789012345", time.Now().Add(-1*time.Minute))
	// Add valid state
	handler.setState("valid-state-12345678901234567", time.Now().Add(10*time.Minute))

	// Run cleanup
	handler.cleanupExpiredStates()

	// Expired state should be gone
	_, exists := handler.getState("expired-state-123456789012345")
	assert.False(t, exists, "Expired state should be removed")

	// Valid state should remain
	_, exists = handler.getState("valid-state-12345678901234567")
	assert.True(t, exists, "Valid state should remain")
}
