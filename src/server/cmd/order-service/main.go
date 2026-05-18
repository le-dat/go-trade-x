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
	"github.com/verno/gotradex/internal/order"
	"github.com/verno/gotradex/pkg/config"
	"github.com/verno/gotradex/pkg/kafka"
	"github.com/verno/gotradex/pkg/logger"
	"github.com/verno/gotradex/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
		log.With(zap.Error(err)).Fatal("Order service failed")
	}
}

func run(ctx context.Context, cfg *config.Config, log *zap.Logger) error {
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

	if err := conn.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	log.Info("Connected to database")

	// Kafka producer
	brokers := getEnvStr("KAFKA_BROKERS", "localhost:9092")
	prod := kafka.NewProducer([]string{brokers})
	defer prod.Close()
	log.Info("Kafka producer initialized", zap.String("brokers", brokers))

	// User service gRPC client — with dial timeout to avoid hanging on startup
	userServiceAddr := getEnvStr("USER_SERVICE_ADDR", "localhost:50051")
	connCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	connUser, err := grpc.DialContext(connCtx, userServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to dial user service: %w", err)
	}
	userClient := order.NewUserServiceClientAdapter(proto.NewUserServiceClient(connUser))
	log.Info("User service client connected", zap.String("addr", userServiceAddr))

	// Repository + Service + Handler
	repo := order.NewRepository(conn)
	svc := order.NewService(repo, userClient, log)
	handler := order.NewHandler(svc)

	// Outbox relay — runs in background
	relay := order.NewRelay(repo, prod, log)
	relayCtx, relayCancel := context.WithCancel(context.Background())
	go relay.Start(relayCtx)

	// gRPC server
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		return fmt.Errorf("failed to listen on port 50052: %w", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterOrderServiceServer(grpcServer, handler)
	if getEnvStr("ENABLE_REFLECTION", "false") == "true" {
		reflection.Register(grpcServer)
	}

	// Serve in a goroutine; if it fails, log fatal to exit the process
	go func() {
		log.Info("Starting Order Service on :50052")
		if err := grpcServer.Serve(lis); err != nil {
			log.With(zap.Error(err)).Fatal("gRPC server failed")
		}
	}()

	log.Info("Order Service is running")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down Order Service...")
	relayCancel()
	grpcServer.GracefulStop()
	return nil
}