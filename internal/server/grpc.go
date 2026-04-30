package server

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/vhgomes/rate-guard/internal/limiter"
	pb "github.com/vhgomes/rate-guard/pkg/pb/ratelimit/v1"
)

type RateLimitServer struct {
	grpcServer   *grpc.Server
	healthServer *health.Server
	rateLimiter  limiter.Limiter
	configs      map[string]map[string]limiter.LimiterConfig
	pb.UnimplementedRateLimitServiceServer
}

func NewRateLimitServer(rateLimiter limiter.Limiter, configs map[string]map[string]limiter.LimiterConfig) *RateLimitServer {
	s := &RateLimitServer{
		healthServer: health.NewServer(),
		rateLimiter:  rateLimiter,
		configs:      configs,
	}
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.healthServer.SetServingStatus("ratelimit.v1.RateLimitService", grpc_health_v1.HealthCheckResponse_SERVING)
	return s
}

func (s *RateLimitServer) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.grpcServer = grpc.NewServer()
	pb.RegisterRateLimitServiceServer(s.grpcServer, s)
	grpc_health_v1.RegisterHealthServer(s.grpcServer, s.healthServer)
	reflection.Register(s.grpcServer)

	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	return s.grpcServer.Serve(lis)
}

func (s *RateLimitServer) GracefulStop() {
	s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.grpcServer.GracefulStop()
}

func (s *RateLimitServer) CheckRateLimit(ctx context.Context, req *pb.CheckRateLimitRequest) (*pb.CheckRateLimitResponse, error) {
	tenant := req.GetKey()
	limitID := req.GetLimitId()

	tenantConfig, ok := s.configs[tenant]
	if !ok {
		return &pb.CheckRateLimitResponse{
			Allowed:      false,
			Remaining:    0,
			RetryAfterMs: 0,
		}, status.Error(codes.NotFound, "tenant not configured")
	}

	cfg, ok := tenantConfig[limitID]
	if !ok {
		return &pb.CheckRateLimitResponse{
			Allowed:      false,
			Remaining:    0,
			RetryAfterMs: 0,
		}, status.Error(codes.NotFound, "limit not configured")
	}

	result, err := s.rateLimiter.Allow(ctx, tenant, limitID, cfg.Limit, cfg.Window)
	if err != nil {
		return &pb.CheckRateLimitResponse{
			Allowed:      false,
			Remaining:    0,
			RetryAfterMs: 0,
		}, status.Error(codes.Internal, "failed to check rate limit")
	}

	return &pb.CheckRateLimitResponse{
		Allowed:      result.Allowed,
		Remaining:    int32(result.Remaining),
		RetryAfterMs: result.RetryAfterMs,
	}, nil
}
