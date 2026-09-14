package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	client *redis.Client
}

func Connect(url string) (*Redis, error) {
	if url == "" {
		return &Redis{}, nil
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	client := redis.NewClient(opt)
	return &Redis{client: client}, nil
}

func (r *Redis) Enabled() bool {
	return r != nil && r.client != nil
}

func (r *Redis) Ping(ctx context.Context) error {
	if !r.Enabled() {
		return fmt.Errorf("redis is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return r.client.Ping(ctx).Err()
}

func (r *Redis) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}
