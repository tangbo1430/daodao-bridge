package leader

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Lease is a small Redis based leader lease. It is deliberately independent
// from the event store so a single instance can run with the same code path as
// a future active/standby deployment.
type Lease struct {
	Client *redis.Client
	Key    string
	Value  string
	TTL    time.Duration
}

func (l Lease) Acquire(ctx context.Context) (bool, error) {
	if l.Client == nil {
		return false, fmt.Errorf("redis client is required")
	}
	if l.TTL <= 0 {
		l.TTL = 15 * time.Second
	}
	if l.Value == "" {
		return false, fmt.Errorf("lease value is required")
	}
	return l.Client.SetNX(ctx, l.Key, l.Value, l.TTL).Result()
}

func (l Lease) Renew(ctx context.Context) (bool, error) {
	if l.Client == nil {
		return false, fmt.Errorf("redis client is required")
	}
	if l.TTL <= 0 {
		l.TTL = 15 * time.Second
	}
	const script = `if redis.call('get', KEYS[1]) == ARGV[1] then return redis.call('pexpire', KEYS[1], ARGV[2]) else return 0 end`
	value, err := l.Client.Eval(ctx, script, []string{l.Key}, l.Value, l.TTL.Milliseconds()).Int()
	return value == 1, err
}

func (l Lease) Release(ctx context.Context) error {
	if l.Client == nil {
		return fmt.Errorf("redis client is required")
	}
	const script = `if redis.call('get', KEYS[1]) == ARGV[1] then return redis.call('del', KEYS[1]) else return 0 end`
	_, err := l.Client.Eval(ctx, script, []string{l.Key}, l.Value).Result()
	return err
}
