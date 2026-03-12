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

	// Swagger UI
	swaggerURL := "http://" + config.Conf.Server.Host + ":" + config.Conf.Server.Port + "/swagger/doc.json"
	mux.HandleFunc("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL(swaggerURL),
	))

	slog.Info("Creating API routes")
	mux.HandleFunc("POST /api/auth/register", handlers.AuthRegister)
	mux.HandleFunc("POST /api/auth/login", handlers.AuthLogin)

	protectedMux := http.NewServeMux()

	protectedMux.HandleFunc("GET /api/artists", handlers.GetArtists)
	protectedMux.HandleFunc("GET /api/artists/{id}", handlers.GetArtist)
	protectedMux.HandleFunc("POST /api/artists", handlers.CreateArtist)
	protectedMux.HandleFunc("DELETE /api/artists/{id}", handlers.DeleteArtist)
	protectedMux.HandleFunc("PUT /api/artists/{id}", handlers.UpdateArtist)
	protectedMux.HandleFunc("GET /api/artists/{id}/albums", handlers.GetArtistAlbums)
	// protectedMux.HandleFunc("PUT /api/artists/{id}/refresh", handlers.RefreshArtist)
	//
	protectedMux.HandleFunc("GET /api/albums", handlers.GetAlbums)
	protectedMux.HandleFunc("GET /api/albums/{id}", handlers.GetAlbum)
	protectedMux.HandleFunc("POST /api/albums", handlers.CreateAlbum)
	protectedMux.HandleFunc("PUT /api/albums/{id}", handlers.UpdateAlbum)
	protectedMux.HandleFunc("DELETE /api/albums/{id}", handlers.DeleteAlbum)

	// Wrap protected routes with Auth middleware
	mux.Handle("/api/", middleware.Auth(protectedMux))

	serverAddr := config.Conf.Server.Host + ":" + config.Conf.Server.Port
	slog.Info("Starting server", "address", serverAddr)
	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatal(err)
	}
}
