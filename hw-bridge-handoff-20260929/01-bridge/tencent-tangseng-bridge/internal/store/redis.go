package store

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	Client *redis.Client
	Key    string
}

func NewRedisQueue(rawURL, key string) (*RedisQueue, error) {
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	if key == "" {
		key = "bridge:outbox"
	}
	return &RedisQueue{Client: redis.NewClient(options), Key: key}, nil
}

func (q *RedisQueue) Ping(ctx context.Context) error {
	if q == nil || q.Client == nil {
		return fmt.Errorf("redis queue is not configured")
	}
	return q.Client.Ping(ctx).Err()
}

func (q *RedisQueue) Enqueue(ctx context.Context, jobID string) error {
	return q.Client.RPush(ctx, q.Key, jobID).Err()
}
