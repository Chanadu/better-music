package handlers

import (
	"database/sql"
	"net/http"
	"sync"
	"time"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/internal/auth"
	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

type Handler struct {
	Auth              *auth.Service
	Config            *config.Config
	Database          *sql.DB
	spotifyHTTPClient *http.Client
	spotifyClient     *spotify.Client
	spotifyToken      *oauth2.Token
	spotifyMutex      sync.Mutex
}

func NewHandler(cfg *config.Config, db *sql.DB) *Handler {
	handler := &Handler{
		Config:   cfg,
		Database: db,
		spotifyHTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	handler.spotifyClient = spotify.New(&http.Client{
		Transport: spotifyTransport{handler: handler},
		Timeout:   handler.spotifyHTTPClient.Timeout,
	})
	return handler
}
