package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vhgomes/rate-guard/internal/config"
	"github.com/vhgomes/rate-guard/internal/limiter"
	"github.com/vhgomes/rate-guard/internal/server"
	pkg "github.com/vhgomes/rate-guard/pkg/logging"
)

func main() {
	pkg.Info("Starting rate limit server")
	cfg := config.LoadConfig()

	pkg.Info(fmt.Sprintf("Config carregada: listen_addr=%q, redis.addr=%q", cfg.ListenAddr, cfg.Redis.Addr))

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		pkg.Error("failed to connect to Redis", err)
	}

	lim := limiter.NewRedisFixedWindowLimiter(redisClient, "rateguard")

	rateConfigs := convertConfig(cfg.Tenants)

	rateLimitServer := server.NewRateLimitServer(lim, rateConfigs)

	go func() {
		pkg.Info("Starting rate limit server on " + cfg.ListenAddr)
		if err := rateLimitServer.Start(cfg.ListenAddr); err != nil {
			pkg.Error("server failed", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	pkg.Info("Shutting down server...")
	rateLimitServer.GracefulStop()
	pkg.Info("Server stopped")
}

func convertConfig(tenants map[string]map[string]config.LimiterConfig) map[string]map[string]limiter.LimiterConfig {
	result := make(map[string]map[string]limiter.LimiterConfig)
	for tenant, limits := range tenants {
		result[tenant] = make(map[string]limiter.LimiterConfig)
		for limitID, cfg := range limits {
			result[tenant][limitID] = limiter.LimiterConfig{
				Limit:  cfg.Limit,
				Window: time.Duration(cfg.WindowSeconds) * time.Second,
			}
		}
	}
	return result
}
