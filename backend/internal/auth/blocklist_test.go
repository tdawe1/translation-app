package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, mr, func() {
		client.Close()
		mr.Close()
	}
}

func TestTokenBlocklist_Add(t *testing.T) {
	client, _, cleanup := setupTestRedis(t)
	defer cleanup()

	blocklist := NewTokenBlocklist(client)
	ctx := context.Background()

	err := blocklist.Add(ctx, "user-123", "token-abc", 5*time.Minute)
	assert.NoError(t, err)

	// Verify key exists
	exists, err := client.Exists(ctx, "user:user-123:blocklist:token-abc").Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), exists)
}

func TestTokenBlocklist_IsBlocked(t *testing.T) {
	client, _, cleanup := setupTestRedis(t)
	defer cleanup()

	blocklist := NewTokenBlocklist(client)
	ctx := context.Background()

	// Not blocked initially
	blocked, err := blocklist.IsBlocked(ctx, "user-123", "token-xyz")
	assert.NoError(t, err)
	assert.False(t, blocked)

	// Add to blocklist
	err = blocklist.Add(ctx, "user-123", "token-xyz", 5*time.Minute)
	require.NoError(t, err)

	// Now blocked
	blocked, err = blocklist.IsBlocked(ctx, "user-123", "token-xyz")
	assert.NoError(t, err)
	assert.True(t, blocked)
}

func TestTokenBlocklist_Expiry(t *testing.T) {
	client, mr, cleanup := setupTestRedis(t)
	defer cleanup()

	blocklist := NewTokenBlocklist(client)
	ctx := context.Background()

	// Add with very short expiry
	err := blocklist.Add(ctx, "user-123", "token-exp", 10*time.Millisecond)
	require.NoError(t, err)

	// Fast-forward time to expire the key
	mr.FastForward(100 * time.Millisecond)

	// Should no longer be blocked
	blocked, err := blocklist.IsBlocked(ctx, "user-123", "token-exp")
	assert.NoError(t, err)
	assert.False(t, blocked)
}

func TestTokenBlocklist_MultiTenancy(t *testing.T) {
	client, _, cleanup := setupTestRedis(t)
	defer cleanup()

	blocklist := NewTokenBlocklist(client)
	ctx := context.Background()

	// Add token for user A
	err := blocklist.Add(ctx, "user-A", "shared-token", 5*time.Minute)
	require.NoError(t, err)

	// User A's token is blocked
	blocked, err := blocklist.IsBlocked(ctx, "user-A", "shared-token")
	assert.NoError(t, err)
	assert.True(t, blocked)

	// User B with same token ID is NOT blocked (different namespace)
	blocked, err = blocklist.IsBlocked(ctx, "user-B", "shared-token")
	assert.NoError(t, err)
	assert.False(t, blocked)
}
