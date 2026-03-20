package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Chanadu/better-music/models"
)

// CreateArtistRequest represents the request body for creating an artist
type CreateArtistRequest struct {
	Name      string  `json:"name" example:"The Beatles"`
	CoverURL  *string `json:"cover_url,omitempty" example:"https://example.com/artist.jpg"`
	SpotifyID *string `json:"spotify_id,omitempty" example:"6ml0jHmy7SNFWckrZblO5B"`
}

// UpdateArtistRequest represents the request body for updating an artist
type UpdateArtistRequest struct {
	Name      *string `json:"name,omitempty" example:"The Beatles"`
	CoverURL  *string `json:"cover_url,omitempty" example:"https://example.com/artist.jpg"`
	SpotifyID *string `json:"spotify_id,omitempty" example:"6ml0jHmy7SNFWckrZblO5B"`
}

// GetArtists godoc
// @Summary Get all artists
// @Description Get all artists for the authenticated user
// @Tags artists
// @Produce json
// @Security Bearer
// @Success 200 {array} models.Artist
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
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

	// writeJSON(w, http.StatusOK, fmt.Sprintf("Get artists for user %d", userID))
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
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Artist not found"
// @Failure 500 {object} map[string]string "Server error"
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

	writeJSON(w, http.StatusOK, artist)
}

// CreateArtist godoc
// @Summary Create a new artist
// @Description Create a new artist for the authenticated user
// @Tags artists
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateArtistRequest true "Artist data"
// @Success 201 {object} models.Artist
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 409 {object} map[string]string "Artist already exists"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/artists [post]
func (h *Handler) CreateArtist(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "POST /api/artists", "method", r.Method, "path", r.URL.Path)
	var body CreateArtistRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	if body.Name == "" {
		writeJSON(w, http.StatusBadRequest, apiError("name is required"))
		return
	}

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	exists, err := models.ArtistExistsByName(h.Database, userID, body.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check existing artist: "+err.Error()))
		return
	}

	if exists {
		writeJSON(w, http.StatusConflict, apiError("artist with this name already exists"))
		return
	}

	artist, err := models.CreateArtist(h.Database, userID, body.Name, body.CoverURL, body.SpotifyID)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to create artist: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, artist)
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
// @Param id path int true "Artist ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Artist has albums"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Artist not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/artists/{id} [delete]
func (h *Handler) DeleteArtist(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "DELETE /api/artists/{id}", "method", r.Method, "path", r.URL.Path)
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
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check artist's albums: "+err.Error()))
		return
	}

	if len(albums) > 0 {
		writeJSON(w, http.StatusBadRequest, apiError("artist cannot be deleted because it has albums"))
		return
	}

	err = models.DeleteArtist(h.Database, userID, artistID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to delete artist: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "artist deleted"})
}

// UpdateArtist godoc
// @Summary Update an artist
// @Description Update an artist's name and/or Spotify ID
// @Tags artists
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Artist ID"
// @Param request body UpdateArtistRequest true "Update data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid request or no fields provided"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Artist not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/artists/{id} [put]
func (h *Handler) UpdateArtist(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "PUT /api/artists/{id}", "method", r.Method, "path", r.URL.Path)
	var body UpdateArtistRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	if isEmpty(body.Name) && isEmpty(body.CoverURL) && isEmpty(body.SpotifyID) {
		writeJSON(w, http.StatusBadRequest, apiError("at least one field must be provided"))
		return
	}

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	artistID, ok := h.checkArtistExistsByID(w, userID, r.PathValue("id"))
	if !ok {
		return
	}

	err := models.UpdateArtist(h.Database, userID, artistID, body.Name, body.CoverURL, body.SpotifyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to update artist: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "artist updated"})
}

// GetArtistAlbums godoc
// @Summary Get all albums by an artist
// @Description Get all albums for a specific artist
// @Tags artists
// @Produce json
// @Security Bearer
// @Param id path int true "Artist ID"
// @Success 200 {array} models.Album
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Artist not found"
// @Failure 500 {object} map[string]string "Server error"
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
