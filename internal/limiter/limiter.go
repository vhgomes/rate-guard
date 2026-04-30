package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	//"github.com/vhgomes/rate-guard/internal/metrics"
)

type Limiter interface {
	Allow(ctx context.Context, key string, limitID string, limit int, window time.Duration) (Result, error)
}

type Result struct {
	Allowed      bool
	Remaining    int64
	RetryAfterMs int64
}

type redisFixedWindowLimiter struct {
	client *redis.Client
	prefix string
}

type LimiterConfig struct {
	Limit  int
	Window time.Duration
}

func NewRedisFixedWindowLimiter(client *redis.Client, prefix string) Limiter {
	return &redisFixedWindowLimiter{
		client: client,
		prefix: prefix,
	}
}

func (l *redisFixedWindowLimiter) Allow(ctx context.Context, key string, limitID string, limit int, window time.Duration) (Result, error) {
	currentWindow := time.Now().Unix() / int64(window.Seconds())
	keyRedis := fmt.Sprintf("%s:%s:%s:%d", l.prefix, limitID, key, currentWindow)

	cmd := l.client.Incr(ctx, keyRedis)
	count, err := cmd.Result()
	if err != nil {
		return Result{}, err
	}

	if count == 1 {
		p := l.client.Pipeline()
		p.Expire(ctx, keyRedis, window)

		_, _ = p.Exec(ctx)
	}

	allowed := count <= int64(limit)
	remaining := int64(limit) - count
	if remaining < 0 {
		remaining = 0
	}
	var retryAfterMs int64
	if !allowed {
		ttlSeconds := (currentWindow + 1) * int64(window.Seconds())
		retryAfterMs = (ttlSeconds - time.Now().Unix()) * 1000
	}

	return Result{
		Allowed:      allowed,
		Remaining:    remaining,
		RetryAfterMs: retryAfterMs,
	}, nil
}
