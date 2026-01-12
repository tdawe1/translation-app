package oauth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	oauthStateKeyPrefix = "oauth:state:"
)

// StateStore defines the interface for OAuth state storage
type StateStore interface {
	Set(ctx context.Context, state string, expiry time.Duration) error
	Exists(ctx context.Context, state string) (bool, error)
	Delete(ctx context.Context, state string) error
}

// RedisStateStore stores OAuth state tokens in Redis with automatic expiration
type RedisStateStore struct {
	redis *redis.Client
}

// NewRedisStateStore creates a new Redis-backed state store
func NewRedisStateStore(redisClient *redis.Client) *RedisStateStore {
	return &RedisStateStore{
		redis: redisClient,
	}
}

// Set stores an OAuth state token with expiration
func (s *RedisStateStore) Set(ctx context.Context, state string, expiry time.Duration) error {
	key := oauthStateKeyPrefix + state
	return s.redis.Set(ctx, key, "1", expiry).Err()
}

// Exists checks if an OAuth state token exists and is not expired
func (s *RedisStateStore) Exists(ctx context.Context, state string) (bool, error) {
	key := oauthStateKeyPrefix + state
	result, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check state existence: %w", err)
	}
	return result == 1, nil
}

// Delete removes an OAuth state token
func (s *RedisStateStore) Delete(ctx context.Context, state string) error {
	key := oauthStateKeyPrefix + state
	return s.redis.Del(ctx, key).Err()
}
