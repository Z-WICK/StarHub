package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionCache interface {
	GetUserIDByTokenHash(ctx context.Context, tokenHash string) (int64, bool, error)
	SetUserIDByTokenHash(ctx context.Context, tokenHash string, userID int64, ttl time.Duration) error
}

type RedisSessionCache struct {
	client *redis.Client
}

func NewRedisSessionCache(redisURL string) (*RedisSessionCache, error) {
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(options)
	return &RedisSessionCache{client: client}, nil
}

func (c *RedisSessionCache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}

func (c *RedisSessionCache) Close() error {
	return c.client.Close()
}

func (c *RedisSessionCache) GetUserIDByTokenHash(ctx context.Context, tokenHash string) (int64, bool, error) {
	key := sessionCacheKey(tokenHash)
	value, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("get session cache: %w", err)
	}
	userID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse cached user id: %w", err)
	}
	if userID <= 0 {
		return 0, false, nil
	}
	return userID, true, nil
}

func (c *RedisSessionCache) SetUserIDByTokenHash(ctx context.Context, tokenHash string, userID int64, ttl time.Duration) error {
	key := sessionCacheKey(tokenHash)
	if err := c.client.Set(ctx, key, strconv.FormatInt(userID, 10), ttl).Err(); err != nil {
		return fmt.Errorf("set session cache: %w", err)
	}
	return nil
}

func sessionCacheKey(tokenHash string) string {
	return "session:token_hash:" + tokenHash
}
