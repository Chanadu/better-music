package main

import (
	"log"
	"log/slog"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/db"
	"github.com/Chanadu/better-music/logger"
)

func main() {
	config, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	logger.SetupLogger(config)
	if config.Logs.Debug {
		slog.Info("=============================================================")
	}

	db.Connect(config)
	db.RunMigrations(config)
}
