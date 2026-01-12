package oauth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisStateStore_SetAndGet(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	store := NewRedisStateStore(client)
	ctx := context.Background()

	state := "test-state-123"
	expiry := time.Minute

	err := store.Set(ctx, state, expiry)
	require.NoError(t, err)

	exists, err := store.Exists(ctx, state)
	require.NoError(t, err)
	assert.True(t, exists, "state should exist immediately after Set")
}

func TestRedisStateStore_Expiration(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	store := NewRedisStateStore(client)
	ctx := context.Background()

	state := "expiring-state"
	err := store.Set(ctx, state, 100*time.Millisecond)
	require.NoError(t, err)

	s.FastForward(200 * time.Millisecond)

	exists, err := store.Exists(ctx, state)
	require.NoError(t, err)
	assert.False(t, exists, "state should be expired")
}

func TestRedisStateStore_Delete(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	store := NewRedisStateStore(client)
	ctx := context.Background()

	state := "delete-me"
	_ = store.Set(ctx, state, time.Minute)

	err := store.Delete(ctx, state)
	require.NoError(t, err)

	exists, _ := store.Exists(ctx, state)
	assert.False(t, exists, "state should be deleted")
}

func TestRedisStateStore_Exists_NotFound(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	store := NewRedisStateStore(client)
	ctx := context.Background()

	exists, err := store.Exists(ctx, "non-existent-state")
	require.NoError(t, err)
	assert.False(t, exists, "non-existent state should return false")
}

func TestRedisStateStore_Delete_NonExistent(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	store := NewRedisStateStore(client)
	ctx := context.Background()

	// Deleting a non-existent key should not error
	err := store.Delete(ctx, "non-existent-state")
	require.NoError(t, err)
}
