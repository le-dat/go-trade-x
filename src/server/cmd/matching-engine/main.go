package main

import (
	"github.com/verno/gotradex/pkg/config"
	"github.com/verno/gotradex/pkg/logger"
)

func main() {
	config.Load()
	logger.Init()
	log := logger.Get()

	log.Info("Starting Matching Engine")
	select {}
}
