package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Chanadu/better-music/models"
)

type CreateArtistRequest struct {
	Name      string  `json:"name" example:"The Beatles" validate:"required"`
	CoverURL  *string `json:"cover_url,omitempty" extensions:"x-nullable" example:"https://example.com/artist.jpg"`
	SpotifyID *string `json:"spotify_id,omitempty" extensions:"x-nullable" example:"6ml0jHmy7SNFWckrZblO5B"`
}

type UpdateArtistRequest struct {
	Name      *string `json:"name,omitempty" example:"The Beatles"`
	CoverURL  *string `json:"cover_url,omitempty" extensions:"x-nullable" example:"https://example.com/artist.jpg"`
	SpotifyID *string `json:"spotify_id,omitempty" extensions:"x-nullable" example:"6ml0jHmy7SNFWckrZblO5B"`
}

// GetArtists godoc
// @Summary Get all artists
// @Description Get all artists for the authenticated user
// @Tags artists
// @Produce json
// @Security Bearer
// @Success 200 {array} models.Artist
// @Failure 401 {object} ApiErrorResponse "Unauthorized"
// @Failure 500 {object} ApiErrorResponse "Server error"
// @Router /api/artists [get]
func (h *Handler) GetArtists(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "GET /api/artists", "method", r.Method, "path", r.URL.Path)
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	artists, err := models.GetArtistsByUser(h.Database, userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to get artists: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, artists)
}

// GetArtist godoc
// @Summary Get a specific artist
// @Description Get a specific artist by ID for the authenticated user
// @Tags artists
// @Produce json
// @Security Bearer
// @Param id path int true "Artist ID"
// @Success 200 {object} models.Artist
// @Failure 401 {object} ApiErrorResponse "Unauthorized"
// @Failure 404 {object} ApiErrorResponse "Artist not found"
// @Failure 500 {object} ApiErrorResponse "Server error"
// @Router /api/artists/{id} [get]
func (h *Handler) GetArtist(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "GET /api/artists/{id}", "method", r.Method, "path", r.URL.Path)
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	artistID, ok := h.checkArtistExistsByID(w, userID, r.PathValue("id"))
	if !ok {
		return
	}

	artist, err := models.GetArtistByID(h.Database, userID, artistID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to get artist: "+err.Error()))
		return
	}

	w.Header().Set("ETag", strconv.Quote(strconv.Itoa(artist.Version)))
	writeJSON(w, http.StatusOK, artist)
}

// CreateArtist godoc
// @Summary Create a new artist
// @Description Create a new artist for the authenticated user
// @Tags artists
// @Accept json
// @Produce json
// @Security Bearer
// @Param Idempotency-Key header string false "Account-scoped mutation ID; reuse only for an identical request"
// @Param request body CreateArtistRequest true "Artist data"
// @Success 201 {object} models.Artist
// @Failure 400 {object} ApiErrorResponse "Invalid request"
// @Failure 401 {object} ApiErrorResponse "Unauthorized"
// @Failure 409 {object} ApiErrorResponse "Artist already exists"
// @Failure 500 {object} ApiErrorResponse "Server error"
// @Router /api/artists [post]
func (h *Handler) CreateArtist(w http.ResponseWriter, r *http.Request) {
	h.handleLibraryMutation(w, r, models.ArtistEntity)
}

func (h *Handler) checkArtistExistsByID(w http.ResponseWriter, userID int, idStr string) (int, bool) {
	artistID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid artist ID"))
		return 0, false
	}
	exists, err := models.ArtistExistsByID(h.Database, userID, artistID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check existing artist: "+err.Error()))
		return 0, false
	}
	if !exists {
		writeJSON(w, http.StatusNotFound, apiError("artist not found"))
		return 0, false
	}
	return artistID, true
}

// DeleteArtist godoc
// @Summary Delete an artist
// @Description Delete an artist by ID (must have no albums)
// @Tags artists
// @Produce json
// @Security Bearer
// @Param If-Match header string false "Expected positive record version, bare or quoted"
// @Param Idempotency-Key header string false "Account-scoped mutation ID; reuse only for an identical request"
// @Param id path int true "Artist ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ApiErrorResponse "Artist has albums"
// @Failure 401 {object} ApiErrorResponse "Unauthorized"
// @Failure 412 {object} ArtistConflictResponse "Version conflict with current record"
// @Failure 500 {object} ApiErrorResponse "Server error"
// @Router /api/artists/{id} [delete]
func (h *Handler) DeleteArtist(w http.ResponseWriter, r *http.Request) {
	h.handleLibraryMutation(w, r, models.ArtistEntity)
}

// UpdateArtist godoc
// @Summary Update an artist
// @Description Update an artist's name and/or Spotify ID
// @Tags artists
// @Accept json
// @Produce json
// @Security Bearer
// @Param If-Match header string false "Expected positive record version, bare or quoted"
// @Param Idempotency-Key header string false "Account-scoped mutation ID; reuse only for an identical request"
// @Param id path int true "Artist ID"
// @Param request body UpdateArtistRequest true "Update data"
// @Success 200 {object} models.Artist
// @Failure 400 {object} ApiErrorResponse "Invalid request or no fields provided"
// @Failure 401 {object} ApiErrorResponse "Unauthorized"
// @Failure 404 {object} ApiErrorResponse "Artist not found"
// @Failure 409 {object} ApiErrorResponse "Artist already exists"
// @Failure 412 {object} ArtistConflictResponse "Version conflict with current record"
// @Failure 500 {object} ApiErrorResponse "Server error"
// @Router /api/artists/{id} [put]
func (h *Handler) UpdateArtist(w http.ResponseWriter, r *http.Request) {
	h.handleLibraryMutation(w, r, models.ArtistEntity)
}

// GetArtistAlbums godoc
// @Summary Get all albums by an artist
// @Description Get all albums for a specific artist
// @Tags artists
// @Produce json
// @Security Bearer
// @Param id path int true "Artist ID"
// @Success 200 {array} models.Album
// @Failure 401 {object} ApiErrorResponse "Unauthorized"
// @Failure 404 {object} ApiErrorResponse "Artist not found"
// @Failure 500 {object} ApiErrorResponse "Server error"
// @Router /api/artists/{id}/albums [get]
func (h *Handler) GetArtistAlbums(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "GET /api/artists/{id}/albums", "method", r.Method, "path", r.URL.Path)
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	artistID, ok := h.checkArtistExistsByID(w, userID, r.PathValue("id"))
	if !ok {
		return
	}

	albums, err := models.GetArtistAlbums(h.Database, userID, artistID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to get artist's albums: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, albums)

}
