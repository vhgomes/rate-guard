package limiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vhgomes/rate-guard/internal/limiter"
)

func TestRedisFixedWindowLimiter(t *testing.T) {
	t.Parallel()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	t.Cleanup(func() { rdb.Close() })

	ctx := context.Background()
	l := limiter.NewRedisFixedWindowLimiter(rdb, "test")

	t.Run("should allow first request", func(t *testing.T) {
		mr.FlushAll()

		result, err := l.Allow(ctx, "user1", "login", 10, time.Minute)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, int64(9), result.Remaining)
		assert.Equal(t, int64(0), result.RetryAfterMs)
	})

	t.Run("should reject request after limit is reached", func(t *testing.T) {
		mr.FlushAll()
		for range 10 {
			_, err := l.Allow(ctx, "user2", "login", 10, time.Minute)
			require.NoError(t, err)
		}

		result, err := l.Allow(ctx, "user2", "login", 10, time.Minute)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Equal(t, int64(0), result.Remaining)
		assert.Greater(t, result.RetryAfterMs, int64(0))
	})

	t.Run("should allow requests after window reset", func(t *testing.T) {
		mr.FlushAll()

		for range 10 {
			_, err := l.Allow(ctx, "user3", "login", 10, time.Minute)
			require.NoError(t, err)
		}

		result, err := l.Allow(ctx, "user3", "login", 10, time.Minute)
		require.NoError(t, err)
		assert.False(t, result.Allowed)

		mr.FastForward(time.Minute)

		result, err = l.Allow(ctx, "user3", "login", 10, time.Minute)
		require.NoError(t, err)
		assert.True(t, result.Allowed)
		assert.Equal(t, int64(9), result.Remaining)
	})

	t.Run("should block request when limit exceeded", func(t *testing.T) {
		mr.FlushAll()

		for range 5 {
			_, err := l.Allow(ctx, "user4", "login", 5, time.Minute)
			require.NoError(t, err)
		}

		result, err := l.Allow(ctx, "user4", "login", 5, time.Minute)
		require.NoError(t, err)
		assert.False(t, result.Allowed)
		assert.Equal(t, int64(0), result.Remaining)
		assert.True(t, result.RetryAfterMs > 0)
	})

}

// BenchmarkFixedWindowConcurrency testa a perfomance sob acesso concorrente
func BenchmarkFixedWindowConcurrency(b *testing.B) {
	mr, err := miniredis.Run()
	require.NoError(b, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	lim := limiter.NewRedisFixedWindowLimiter(rdb, "bench")
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := lim.Allow(ctx, "user", "login", 1000, time.Minute)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
