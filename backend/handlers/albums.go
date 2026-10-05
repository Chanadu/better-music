package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Chanadu/better-music/models"
)

type CreateAlbumRequest struct {
	ArtistID   int     `json:"artist_id" example:"1" validate:"required"`
	Title      string  `json:"title" example:"Abbey Road" validate:"required"`
	SpotifyID  *string `json:"spotify_id,omitempty" extensions:"x-nullable" example:"4oDw9mW4Sro2zN1RHzlvOr"`
	CoverURL   *string `json:"cover_url,omitempty" extensions:"x-nullable"`
	Year       *int    `json:"year,omitempty" extensions:"x-nullable"`
	Listened   *bool   `json:"listened,omitempty"`
	Rating     *int    `json:"rating,omitempty" extensions:"x-nullable"`
	Comment    *string `json:"comment,omitempty" extensions:"x-nullable"`
	ListenedAt *string `json:"listened_at,omitempty" extensions:"x-nullable"`
}

type UpdateAlbumRequest struct {
	ArtistID   int     `json:"artist_id,omitempty" example:"1"`
	Title      *string `json:"title,omitempty" example:"Abbey Road"`
	CoverURL   *string `json:"cover_url,omitempty" extensions:"x-nullable" example:"https://example.com/cover.jpg"`
	Year       *int    `json:"year,omitempty" extensions:"x-nullable" example:"1969"`
	SpotifyID  *string `json:"spotify_id,omitempty" extensions:"x-nullable" example:"4oDw9mW4Sro2zN1RHzlvOr"`
	Listened   *bool   `json:"listened,omitempty" example:"true"`
	Rating     *int    `json:"rating,omitempty" extensions:"x-nullable" example:"8"`
	Comment    *string `json:"comment,omitempty" extensions:"x-nullable" example:"Classic album"`
	ListenedAt *string `json:"listened_at,omitempty" extensions:"x-nullable" example:"2024-01-15"`
}

type ArtistIDRequest struct {
	ArtistID int `json:"artist_id" example:"1" validate:"required"`
}

// GetAlbums godoc
// @Summary Get all albums
// @Description Get all albums for the authenticated user
// @Tags albums
// @Produce json
// @Security Bearer
// @Success 200 {array} models.Album
// @Failure 401 {object} ApiErrorResponse "Unauthorized"
// @Failure 500 {object} ApiErrorResponse "Server error"
// @Router /api/albums [get]
func (h *Handler) GetAlbums(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "GET /api/albums", "method", r.Method, "path", r.URL.Path)
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	albums, err := models.GetAlbumsByUser(h.Database, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to get albums: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, albums)
}

func (h *Handler) checkAlbumExistsByID(w http.ResponseWriter, userID int, artistID int, idStr string) (int, bool) {
	albumID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid album ID"))
		return 0, false
	}

	exists, err := models.AlbumExistsByID(h.Database, userID, artistID, albumID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check existing album: "+err.Error()))
		return 0, false
	}

	if !exists {
		writeJSON(w, http.StatusNotFound, apiError("album not found"))
		return 0, false
	}

	return albumID, true
}

// GetAlbum godoc
// @Summary Get a specific album
// @Description Get a specific album by ID
// @Tags albums
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Album ID"
// @Param artist_id query int true "Artist ID"
// @Success 200 {object} models.Album
// @Failure 400 {object} ApiErrorResponse "Invalid request"
// @Failure 401 {object} ApiErrorResponse "Unauthorized"
// @Failure 404 {object} ApiErrorResponse "Album or artist not found"
// @Failure 500 {object} ApiErrorResponse "Server error"
// @Router /api/albums/{id} [get]
func (h *Handler) GetAlbum(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "GET /api/albums/{id}", "method", r.Method, "path", r.URL.Path)

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	artistID, err := strconv.Atoi(r.URL.Query().Get("artist_id"))
	if err != nil {
		var body ArtistIDRequest

		if decodeErr := json.NewDecoder(r.Body).Decode(&body); decodeErr != nil {
			writeJSON(w, http.StatusBadRequest, apiError("invalid artist ID"))
			return
		}

		artistID = body.ArtistID
	}

	if artistID <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError("invalid artist ID"))
		return
	}

	exists, err := models.ArtistExistsByID(h.Database, userID, artistID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check existing artist: "+err.Error()))
		return
	}

	if !exists {
		writeJSON(w, http.StatusNotFound, apiError("artist not found"))
		return
	}

	albumID, ok := h.checkAlbumExistsByID(w, userID, artistID, r.PathValue("id"))
	if !ok {
		return
	}

	album, err := models.GetAlbumByID(h.Database, userID, artistID, albumID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to get album: "+err.Error()))
		return
	}

	w.Header().Set("ETag", strconv.Quote(strconv.Itoa(album.Version)))
	writeJSON(w, http.StatusOK, album)
}

// CreateAlbum godoc
// @Summary Create a new album
// @Description Create a new album for an artist
// @Tags albums
// @Accept json
// @Produce json
// @Security Bearer
// @Param Idempotency-Key header string false "Account-scoped mutation ID; reuse only for an identical request"
// @Param request body CreateAlbumRequest true "Album data"
// @Success 201 {object} models.Album
// @Failure 400 {object} ApiErrorResponse "Invalid request"
// @Failure 401 {object} ApiErrorResponse "Unauthorized"
// @Failure 409 {object} ApiErrorResponse "Album already exists"
// @Failure 500 {object} ApiErrorResponse "Server error"
// @Router /api/albums [post]
func (h *Handler) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	h.handleLibraryMutation(w, r, models.AlbumEntity)
}

// UpdateAlbum godoc
// @Summary Update an album
// @Description Update an album's information
// @Tags albums
// @Accept json
// @Produce json
// @Security Bearer
// @Param If-Match header string false "Expected positive record version, bare or quoted"
// @Param Idempotency-Key header string false "Account-scoped mutation ID; reuse only for an identical request"
// @Param id path int true "Album ID"
// @Param request body UpdateAlbumRequest true "Update data"
// @Success 200 {object} models.Album
// @Failure 400 {object} ApiErrorResponse "Invalid request or no fields provided"
// @Failure 401 {object} ApiErrorResponse "Unauthorized"
// @Failure 404 {object} ApiErrorResponse "Album or artist not found"
// @Failure 409 {object} ApiErrorResponse "Album already exists"
// @Failure 412 {object} AlbumConflictResponse "Version conflict with current record"
// @Failure 500 {object} ApiErrorResponse "Server error"
// @Router /api/albums/{id} [put]
func (h *Handler) UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	h.handleLibraryMutation(w, r, models.AlbumEntity)
}

// DeleteAlbum godoc
// @Summary Delete an album
// @Description Delete an album by ID
// @Tags albums
// @Accept json
// @Produce json
// @Security Bearer
// @Param If-Match header string false "Expected positive record version, bare or quoted"
// @Param Idempotency-Key header string false "Account-scoped mutation ID; reuse only for an identical request"
// @Param id path int true "Album ID"
// @Param request body ArtistIDRequest false "Optional artist ID; must match the album's artist"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ApiErrorResponse "Invalid request"
// @Failure 401 {object} ApiErrorResponse "Unauthorized"
// @Failure 404 {object} ApiErrorResponse "Album belongs to another artist"
// @Failure 412 {object} AlbumConflictResponse "Version conflict with current record"
// @Failure 500 {object} ApiErrorResponse "Server error"
// @Router /api/albums/{id} [delete]
func (h *Handler) DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	h.handleLibraryMutation(w, r, models.AlbumEntity)
}
