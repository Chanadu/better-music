package main

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/db"
	"github.com/Chanadu/better-music/handlers"
	"github.com/Chanadu/better-music/logger"
)

func main() {
	err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	logger.SetupLogger()
	if config.Conf.Logs.Debug {
		slog.Info("=============================================================")
	}

	slog.Info("Connecting to database")
	db.Connect()

	slog.Info("Running database migrations")
	db.RunMigrations()

	mux := http.NewServeMux()

	slog.Info("Creating API routes")
	mux.HandleFunc("POST /api/auth/register", handlers.AuthRegister)
	mux.HandleFunc("POST /api/auth/login", handlers.AuthLogin)

	slog.Info("Starting server on :8080")
	http.ListenAndServe(":8080", mux)
}
