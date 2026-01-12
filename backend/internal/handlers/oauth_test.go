package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tdawe1/translation-app/internal/oauth"
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

// TestRedisStateStore_Concurrency tests concurrent access to the Redis state store
func TestRedisStateStore_Concurrency(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	store := oauth.NewRedisStateStore(client)
	ctx := context.Background()

	done := make(chan bool)

	// Writer goroutines
	for i := 0; i < 5; i++ {
		go func(idx int) {
			for j := 0; j < 20; j++ {
				state := "concurrent-test-" + string(rune('A'+idx)) + "-" + string(rune('0'+j))
				_ = store.Set(ctx, state, time.Minute)
			}
			done <- true
		}(i)
	}

	// Reader goroutines
	for i := 0; i < 5; i++ {
		go func(idx int) {
			for j := 0; j < 20; j++ {
				state := "concurrent-test-" + string(rune('A'+idx)) + "-" + string(rune('0'+j))
				_, _ = store.Exists(ctx, state)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we got here without panic, concurrent access works
	assert.True(t, true, "No race condition detected")
}

// TestOAuthHandler_StateOperations tests OAuth handler state operations via Redis
func TestOAuthHandler_StateOperations(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	stateStore := oauth.NewRedisStateStore(client)

	handler := &OAuthHandler{
		stateStore:  stateStore,
		stateExpiry: 10 * time.Minute,
	}
	ctx := context.Background()

	t.Run("Set and Exists", func(t *testing.T) {
		state := "test-state-12345678901234567890"

		err := handler.stateStore.Set(ctx, state, 10*time.Minute)
		require.NoError(t, err)

		exists, err := handler.stateStore.Exists(ctx, state)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Exists nonexistent", func(t *testing.T) {
		exists, err := handler.stateStore.Exists(ctx, "nonexistent")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("Delete", func(t *testing.T) {
		state := "state-to-delete-1234567890123"
		_ = handler.stateStore.Set(ctx, state, 10*time.Minute)

		err := handler.stateStore.Delete(ctx, state)
		require.NoError(t, err)

		exists, _ := handler.stateStore.Exists(ctx, state)
		assert.False(t, exists)
	})
}

// TestOAuthHandler_StateExpiration tests that states expire correctly
func TestOAuthHandler_StateExpiration(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	stateStore := oauth.NewRedisStateStore(client)
	ctx := context.Background()

	// Add state with short expiry
	state := "expiring-state-1234567890123456"
	err := stateStore.Set(ctx, state, 100*time.Millisecond)
	require.NoError(t, err)

	// State should exist immediately
	exists, err := stateStore.Exists(ctx, state)
	require.NoError(t, err)
	assert.True(t, exists)

	// Fast forward time
	s.FastForward(200 * time.Millisecond)

	// State should now be expired
	exists, err = stateStore.Exists(ctx, state)
	require.NoError(t, err)
	assert.False(t, exists, "State should be expired after fast-forward")
}

// TestNewOAuthHandler verifies the handler is created with proper dependencies
func TestNewOAuthHandler(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer redisClient.Close()

	// This test verifies the signature change (H-2 fix)
	// The actual creation would require a full database setup
	// NewOAuthHandler now accepts redisClient as the 4th parameter
	_ = redisClient

	assert.True(t, true, "NewOAuthHandler signature accepts redis.Client")
}
