package main

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/db"
	"github.com/Chanadu/better-music/handlers"
	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/middleware"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	err = logger.SetupLogger(cfg.Logs.Enabled, cfg.Logs.File)
	if err != nil {
		log.Fatal(err)
	}

	if cfg.Logs.Debug {
		slog.Info("=============================================================")
	}

	slog.Info("Connecting to database")
	database := db.Connect(cfg.DB.Url)

	slog.Info("Running database migrations")
	db.RunMigrations(cfg.DB.Url)

	h := handlers.NewHandler(&cfg, database)
	mux := http.NewServeMux()

	slog.Info("Creating API routes")
	mux.HandleFunc("POST /api/auth/register", h.AuthRegister)
	mux.HandleFunc("POST /api/auth/login", h.AuthLogin)
	mux.HandleFunc("POST /api/auth/refresh", h.AuthRefresh)
	mux.HandleFunc("POST /api/auth/logout", h.AuthLogout)

	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("GET /api/account", h.GetAccount)
	protectedMux.HandleFunc("PUT /api/account/email", h.UpdateAccountEmail)
	protectedMux.HandleFunc("PUT /api/account/password", h.UpdateAccountPassword)
	protectedMux.HandleFunc("DELETE /api/account", h.DeleteAccount)

	protectedMux.HandleFunc("POST /api/sync/{kind}/pull", h.PullLibrary)
	protectedMux.HandleFunc("POST /api/sync/{kind}/push", h.PushLibrary)
	protectedMux.HandleFunc("/api/artists", retiredLibraryAPI)
	protectedMux.HandleFunc("/api/artists/", retiredLibraryAPI)
	protectedMux.HandleFunc("/api/albums", retiredLibraryAPI)
	protectedMux.HandleFunc("/api/albums/", retiredLibraryAPI)

	protectedMux.HandleFunc("GET /api/spotify/search/artists", h.SearchSpotifyArtists)
	protectedMux.HandleFunc("GET /api/spotify/search/albums", h.SearchSpotifyAlbums)
	protectedMux.HandleFunc("GET /api/spotify/artists/{id}", h.GetSpotifyArtist)
	protectedMux.HandleFunc("GET /api/spotify/albums/{id}", h.GetSpotifyAlbum)

	mux.Handle("/api/", middleware.Auth(protectedMux, cfg.JWTSecret))

	serverAddr := cfg.Server.Host + ":" + cfg.Server.Port
	slog.Info("Starting server", "address", serverAddr)
	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatal(err)
	}
}

func retiredLibraryAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusGone)
	_, _ = w.Write([]byte(`{"error":"Library sync has changed. Update the app."}`))
}
