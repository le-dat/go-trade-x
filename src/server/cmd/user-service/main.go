package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/verno/gotradex/internal/user"
	"github.com/verno/gotradex/pkg/auth"
	"github.com/verno/gotradex/pkg/config"
	"github.com/verno/gotradex/pkg/logger"
	"github.com/verno/gotradex/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvStr(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	cfg := config.Load()
	logger.Init()
	log := logger.Get()

	ctx := context.Background()
	if err := run(ctx, cfg, log); err != nil {
		log.With(zap.Error(err)).Fatal("User service failed")
	}
}

func run(ctx context.Context, cfg *config.Config, log *zap.Logger) error {
	// Connect to database
	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/gotradex?sslmode=disable"
	}

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return fmt.Errorf("failed to parse database URL: %w", err)
	}

	poolConfig.MaxConns = int32(getEnvInt("MAX_CONNS", 10))
	poolConfig.MinConns = int32(getEnvInt("MIN_CONNS", 2))

	conn, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close()

	// Verify connection
	if err := conn.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Connected to database")

	// Initialize JWT manager
	jwtExpiry := 24 * time.Hour
	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, jwtExpiry)

	// Initialize repository and service
	repo := user.NewRepository(conn)
	svc := user.NewService(repo, jwtMgr, jwtExpiry)
	handler := user.NewHandler(svc)

	// Create gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		return fmt.Errorf("failed to listen on port 50051: %w", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterUserServiceServer(grpcServer, handler)
	enableReflection := getEnvStr("ENABLE_REFLECTION", "false") == "true"
	if enableReflection {
		reflection.Register(grpcServer)
	}

	// Start server
	go func() {
		log.Info("Starting User Service on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.With(zap.Error(err)).Error("gRPC server error")
		}
	}()

	log.Info("User Service is running")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down User Service...")
	grpcServer.GracefulStop()
	return nil
}
