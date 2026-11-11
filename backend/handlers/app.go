package handlers

import (
	"database/sql"
	"net/http"
	"sync"
	"time"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/internal/auth"
)

type Handler struct {
	Auth                  *auth.Service
	Config                *config.Config
	Database              *sql.DB
	spotifyHTTPClient     *http.Client
	spotifyAccessToken    string
	spotifyTokenExpiresAt time.Time
	spotifyMutex          sync.Mutex
}

func NewHandler(cfg *config.Config, db *sql.DB) *Handler {
	return &Handler{
		Config:   cfg,
		Database: db,
		spotifyHTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}
