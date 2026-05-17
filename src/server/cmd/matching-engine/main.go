package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.uber.org/zap"

	"github.com/verno/gotradex/internal/matching"
	"github.com/verno/gotradex/pkg/config"
	"github.com/verno/gotradex/pkg/logger"
)

func main() {
	cfg := config.Load()
	logger.Init()
	log := logger.Get()

	brokers := strings.Split(cfg.KafkaBrokers, ",")
	engine := matching.NewEngine(brokers, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Info("Shutting down matching engine...")
		cancel()
	}()

	log.Info("Starting Matching Engine",
		zap.Strings("brokers", brokers))

	if err := engine.Start(ctx); err != nil && err != context.Canceled {
		log.With(zap.Error(err)).Error("Matching engine error")
	}

	_ = engine.Stop()
	log.Info("Matching engine stopped")
}