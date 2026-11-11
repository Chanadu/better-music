package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/db"
	"github.com/Chanadu/better-music/handlers"
	"github.com/Chanadu/better-music/internal/auth"
	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/middleware"
	"github.com/supertokens/supertokens-golang/recipe/session/sessmodels"
	"github.com/supertokens/supertokens-golang/supertokens"
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

	root, service, err := newAPIHandler(&cfg, database)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			if err := service.ReconcilePending(ctx); err != nil {
				slog.Warn("account workflow retry failed", "error", err)
			}
			cancel()
			<-ticker.C
		}
	}()
	serverAddr := cfg.Server.Host + ":" + cfg.Server.Port
	slog.Info("Starting server", "address", serverAddr)
	if err := http.ListenAndServe(serverAddr, root); err != nil {
		log.Fatal(err)
	}
}

func newAPIHandler(cfg *config.Config, database *sql.DB) (http.Handler, *auth.Service, error) {
	handler := handlers.NewHandler(cfg, database)
	mux := http.NewServeMux()

	slog.Info("Creating API routes")
	service, err := auth.Init(cfg.Auth, database)
	if err != nil {
		return nil, nil, err
	}
	handler.Auth = service

	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("GET /api/account", handler.GetAccount)
	protectedMux.HandleFunc("PUT /api/account/email", handler.ChangeSuperTokensAccount)
	protectedMux.HandleFunc("PUT /api/account/password", handler.ChangeSuperTokensAccount)
	protectedMux.HandleFunc("DELETE /api/account", handler.ChangeSuperTokensAccount)

	protectedMux.HandleFunc("POST /api/sync/{kind}/pull", handler.PullLibrary)
	protectedMux.HandleFunc("POST /api/sync/{kind}/push", handler.PushLibrary)
	protectedMux.HandleFunc("/api/artists", retiredLibraryAPI)
	protectedMux.HandleFunc("/api/artists/", retiredLibraryAPI)
	protectedMux.HandleFunc("/api/albums", retiredLibraryAPI)
	protectedMux.HandleFunc("/api/albums/", retiredLibraryAPI)

	protectedMux.HandleFunc("GET /api/spotify/search/artists", handler.SearchSpotifyArtists)
	protectedMux.HandleFunc("GET /api/spotify/search/albums", handler.SearchSpotifyAlbums)
	protectedMux.HandleFunc("GET /api/spotify/artists/{id}", handler.GetSpotifyArtist)
	protectedMux.HandleFunc("GET /api/spotify/albums/{id}", handler.GetSpotifyAlbum)

	checkCore := true
	mux.Handle("GET /api/session", middleware.SuperTokensAuth(http.HandlerFunc(handler.SessionInfo), database, &sessmodels.VerifySessionOptions{
		CheckDatabase: &checkCore,
	}))
	mux.Handle("/api/", middleware.SuperTokensAuth(protectedMux, database, nil))
	root := supertokens.Middleware(mux)

	return root, handler.Auth, nil
}

func retiredLibraryAPI(writer http.ResponseWriter, httpRequest *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusGone)
	_, _ = writer.Write([]byte(`{"error":"Library sync has changed. Update the app."}`))
}
