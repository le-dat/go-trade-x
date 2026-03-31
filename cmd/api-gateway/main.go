package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/verno/gotradex/cmd/api-gateway/clients"
	"github.com/verno/gotradex/cmd/api-gateway/handlers"
	"github.com/verno/gotradex/cmd/api-gateway/middleware"
	"github.com/verno/gotradex/pkg/auth"
	"github.com/verno/gotradex/pkg/config"
	"github.com/verno/gotradex/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func main() {
	cfg := config.Load()
	logger.Init()
	log := logger.Get()

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, 24*time.Hour)

	// Initialize gRPC clients (mock for now)
	grpcFactory := clients.NewGRPCClientFactoryWithTimeout(cfg.JWTSecret, 5*time.Second)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(grpcFactory.UserService)
	orderHandler := handlers.NewOrderHandler(grpcFactory.OrderService)

	// Initialize middleware
	recoveryMW := middleware.Recovery()
	loggerMW := middleware.Logger()
	rateLimiter := middleware.NewRateLimiter(rate.Limit(100), 50) // 100 rps, burst 50
	authMW := middleware.NewAuthMiddleware(jwtManager)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Apply middleware stack
	router.Use(recoveryMW)
	router.Use(loggerMW)
	router.Use(rateLimiter.Middleware())

	// Health check endpoint (no auth required)
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Order routes (auth required)
		orders := v1.Group("/orders")
		orders.Use(authMW.RequireAuth())
		{
			orders.POST("", orderHandler.PlaceOrder)
			orders.GET("/:id", orderHandler.GetOrder)
		}
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.AppPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info("Starting API Gateway",
			zap.String("port", cfg.AppPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("API Gateway failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down API Gateway...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", zap.Error(err))
	}

	log.Info("API Gateway stopped")
}
