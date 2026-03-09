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

	protectedMux := http.NewServeMux()

	protectedMux.HandleFunc("GET /api/artists", handlers.GetArtists)
	protectedMux.HandleFunc("POST /api/artists", handlers.CreateArtist)
	protectedMux.HandleFunc("DELETE /api/artists/{id}", handlers.DeleteArtist)
	protectedMux.HandleFunc("PUT /api/artists/{id}", handlers.UpdateArtist)
	// protectedMux.HandleFunc("GET /api/artists/{id}/albums", handlers.GetArtistAlbums)
	// protectedMux.HandleFunc("PUT /api/artists/{id}/refresh", handlers.RefreshArtist)
	//
	// protectedMux.HandleFunc("GET /api/albums", handlers.GetAlbums)
	// protectedMux.HandleFunc("GET /api/albums/{id}", handlers.GetAlbum)
	// protectedMux.HandleFunc("POST /api/albums", handlers.CreateAlbum)
	// protectedMux.HandleFunc("PUT /api/albums/{id}", handlers.UpdateAlbum)
	// protectedMux.HandleFunc("DELETE /api/albums/{id}", handlers.DeleteAlbum)
	// protectedMux.HandleFunc("PUT /api/albums/{id}/listen", handlers.ListenAlbum)
	//
	// protectedMux.HandleFunc("GET /api/spotify/search/artists", handlers.SearchSpotifyArtists)
	// protectedMux.HandleFunc("GET /api/spotify/search/albums", handlers.SearchSpotifyAlbums)

	// Wrap protected routes with Auth middleware
	mux.Handle("/api/", middleware.Auth(protectedMux))

	slog.Info("Starting server on :8080")
	http.ListenAndServe(":8080", mux)
}
