package main

import (
	"log"
	"log/slog"

	"github.com/Chanadu/better-music/envs"
	"github.com/Chanadu/better-music/logger"
)

func main() {
	config, err := envs.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	logger.SetupLogger(config)

	slog.Info(config.DB.Host)
	slog.Debug(config.DB.Password)

	log.Fatal("test")
}
