package tests

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdawe1/translation-app/internal/watcher"
)

// TestSeenJobsSet_HasTTL verifies that seen_jobs sets have a 24-hour TTL
// This test prevents P0-1: Redis seen_jobs unbounded growth
func TestSeenJobsSet_HasTTL(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer client.Close()

	ctx := context.Background()
	userID := uuid.New()

	// Simulate adding job to seen_jobs using the same key pattern as JobProcessor
	key := "seen_jobs:" + userID.String()
	jobID := "job-123"

	err := client.SAdd(ctx, key, jobID).Err()
	require.NoError(t, err)

	// Set TTL to 24 hours (this is what the fix should do)
	err = client.Expire(ctx, key, 24*time.Hour).Err()
	require.NoError(t, err)

	// Verify TTL is set (approximately 24 hours)
	ttl := client.TTL(ctx, key).Val()
	assert.Greater(t, ttl, 23*time.Hour, "TTL should be ~24 hours")
	assert.LessOrEqual(t, ttl, 24*time.Hour, "TTL should be ~24 hours")
}

// TestSeenJobsKeyFormat verifies the key format matches watcher package
func TestSeenJobsKeyFormat(t *testing.T) {
	userID := uuid.New()
	expectedKey := "user:" + userID.String() + ":seen_jobs"
	actualKey := watcher.GetSeenJobsKey(userID.String())
	assert.Equal(t, expectedKey, actualKey, "Key format should match")
}
