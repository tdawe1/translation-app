package auth

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBlocklist struct {
	redis *redis.Client
}

func NewTokenBlocklist(redisClient *redis.Client) *TokenBlocklist {
	return &TokenBlocklist{redis: redisClient}
}

func (b *TokenBlocklist) Add(ctx context.Context, userID, tokenID string, expiry time.Duration) error {
	key := "user:" + userID + ":blocklist:" + tokenID
	return b.redis.Set(ctx, key, "1", expiry).Err()
}

func (b *TokenBlocklist) IsBlocked(ctx context.Context, userID, tokenID string) (bool, error) {
	key := "user:" + userID + ":blocklist:" + tokenID
	exists, err := b.redis.Exists(ctx, key).Result()
	return exists > 0, err
}
