package handlers

import (
	"database/sql"
	"net/http"
	"sync"
	"time"

	"github.com/Chanadu/better-music/config"
)

// Handler contains all request handlers and their dependencies
type Handler struct {
	Config                *config.Config
	Database              *sql.DB
	spotifyHTTPClient     *http.Client
	spotifyAccessToken    string
	spotifyTokenExpiresAt time.Time
	spotifyMutex          sync.Mutex
}

// NewHandler creates a new Handler with the given config and database
func NewHandler(cfg *config.Config, db *sql.DB) *Handler {
	return &Handler{
		Config:   cfg,
		Database: db,
		spotifyHTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}
