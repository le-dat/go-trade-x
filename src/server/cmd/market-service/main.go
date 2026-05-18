package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/verno/gotradex/internal/market"
	"github.com/verno/gotradex/pkg/config"
	"github.com/verno/gotradex/pkg/kafka"
	"github.com/verno/gotradex/pkg/logger"
)

var symbolRegex = regexp.MustCompile(`^[A-Z]{3,10}(/[A-Z]{3,10})?$`)

// cfg is loaded once at package init to allow origin checker to use it
var cfg = config.Load()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // No origin header, allow (same-origin requests)
		}
		if cfg.AllowedOrigins == "*" {
			return true // Wildcard for development
		}
		allowedOrigins := strings.Split(cfg.AllowedOrigins, ",")
		for _, allowed := range allowedOrigins {
			if strings.TrimSpace(allowed) == origin {
				return true
			}
		}
		return false
	},
}

// activeConnections tracks current WebSocket connections atomically
var activeConnections atomic.Int64

func main() {
	logger.Init()
	log := logger.Get()

	brokers := strings.Split(cfg.KafkaBrokers, ",")

	hub := market.NewHub(log)
	hubCtx, hubCancel := context.WithCancel(context.Background())
	defer hubCancel()

	// Register connection counter decrementer
	market.RegisterConnectionDecrementer(func() {
		activeConnections.Add(-1)
	})

	// Start hub goroutine
	done := make(chan struct{})
	go func() {
		hub.Run(hubCtx.Done())
		close(done)
	}()

	// Start Kafka consumer for trades
	consumer := kafka.NewConsumer(brokers, "market-service", "trades", log)
	go func() {
		consumer.Run(hubCtx, func(_ context.Context, msg kafkago.Message) error {
			var trade market.TradeEvent
			if err := json.Unmarshal(msg.Value, &trade); err != nil {
				log.With(zap.Error(err)).Warn("failed to unmarshal trade")
				return nil
			}
			hub.Broadcast(trade)
			return nil
		})
	}()

	// WebSocket handler
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			http.Error(w, "symbol query parameter required", http.StatusBadRequest)
			return
		}

		// Validate symbol format
		if !symbolRegex.MatchString(symbol) {
			http.Error(w, "invalid symbol format", http.StatusBadRequest)
			return
		}

		// Check connection limit
		current := activeConnections.Load()
		if current >= int64(cfg.MaxConnections) {
			http.Error(w, "connection limit reached", http.StatusServiceUnavailable)
			return
		}
		activeConnections.Add(1)

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			activeConnections.Add(-1)
			log.With(zap.Error(err)).Warn("websocket upgrade failed")
			return
		}

		client := market.NewClient(hub, conn, symbol)
		hub.Subscribe(client)
		go client.Write()
		go client.Read()
		// Note: Read() defer will call hub.Unsubscribe and activeConnections.Dec
	})

	// Start HTTP server with timeouts
	addr := ":" + cfg.AppPort
	log.Info("Starting Market Service", zap.String("addr", addr))

	server := &http.Server{
		Addr: addr,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout: 60 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.With(zap.Error(err)).Error("HTTP server error")
		}
	}()

	// Handle shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down market service...")
	hubCancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.With(zap.Error(err)).Error("server shutdown error")
	}

	<-done
	log.Info("Market service stopped")
}