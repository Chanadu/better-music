// @title Better Music API
// @version 1.0
// @description A music management API with artists and albums
// @host localhost:8080
// @BasePath /
// @schemes http
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @note Update the @host value above and run 'swag init' if you change the server host/port
package main

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/db"
	_ "github.com/Chanadu/better-music/docs"
	"github.com/Chanadu/better-music/handlers"
	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	logger.SetupLogger(cfg.Logs.File)
	if cfg.Logs.Debug {
		slog.Info("=============================================================")
	}

	slog.Info("Connecting to database")
	database := db.Connect(cfg.DB.Url)

	slog.Info("Running database migrations")
	db.RunMigrations(cfg.DB.Url)

	// Create handler with config and database
	h := handlers.NewHandler(&cfg, database)

	mux := http.NewServeMux()

	// Swagger UI
	swaggerURL := "http://" + cfg.Server.Host + ":" + cfg.Server.Port + "/swagger/doc.json"
	mux.HandleFunc("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL(swaggerURL),
	))

	slog.Info("Creating API routes")
	mux.HandleFunc("POST /api/auth/register", h.AuthRegister)
	mux.HandleFunc("POST /api/auth/login", h.AuthLogin)

	protectedMux := http.NewServeMux()

	protectedMux.HandleFunc("GET /api/artists", h.GetArtists)
	protectedMux.HandleFunc("GET /api/artists/{id}", h.GetArtist)
	protectedMux.HandleFunc("POST /api/artists", h.CreateArtist)
	protectedMux.HandleFunc("DELETE /api/artists/{id}", h.DeleteArtist)
	protectedMux.HandleFunc("PUT /api/artists/{id}", h.UpdateArtist)
	protectedMux.HandleFunc("GET /api/artists/{id}/albums", h.GetArtistAlbums)
	// protectedMux.HandleFunc("PUT /api/artists/{id}/refresh", h.RefreshArtist)
	//
	protectedMux.HandleFunc("GET /api/albums", h.GetAlbums)
	protectedMux.HandleFunc("GET /api/albums/{id}", h.GetAlbum)
	protectedMux.HandleFunc("POST /api/albums", h.CreateAlbum)
	protectedMux.HandleFunc("PUT /api/albums/{id}", h.UpdateAlbum)
	protectedMux.HandleFunc("DELETE /api/albums/{id}", h.DeleteAlbum)

	// Wrap protected routes with Auth middleware
	mux.Handle("/api/", middleware.Auth(protectedMux, cfg.JWTSecret))

	serverAddr := cfg.Server.Host + ":" + cfg.Server.Port
	slog.Info("Starting server", "address", serverAddr)
	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatal(err)
	}
}
