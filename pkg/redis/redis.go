// Package redis provides the shared Redis client used by realtime features.
package redis

import (
	"context"
	"fmt"
	"time"

	"chatapp/internal/config"

	"github.com/redis/go-redis/v9"
)

// New connects to Redis and verifies that it is reachable before returning.
// Realtime features use this shared client for presence, typing and Pub/Sub.
func New(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect Redis at %s: %w", cfg.RedisAddr, err)
	}

	return client, nil
}
