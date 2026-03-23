package main

import (
	"fmt"
	"net/http"

	"github.com/verno/gotradex/pkg/config"
	"github.com/verno/gotradex/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	logger.Init()
	log := logger.Get()

	log.Info("Starting API Gateway", 
		zap.String("port", cfg.AppPort))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("API Gateway failed", zap.Error(err))
	}
}
