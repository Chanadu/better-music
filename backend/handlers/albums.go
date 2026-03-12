package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Chanadu/better-music/models"
)

// CreateAlbumRequest represents the request body for creating an album
type CreateAlbumRequest struct {
	ArtistID int    `json:"artist_id" example:"1"`
	Title    string `json:"title" example:"Abbey Road"`
}

// UpdateAlbumRequest represents the request body for updating an album
type UpdateAlbumRequest struct {
	ArtistID   int     `json:"artist_id" example:"1"`
	Title      *string `json:"title,omitempty" example:"Abbey Road"`
	CoverURL   *string `json:"cover_url,omitempty" example:"https://example.com/cover.jpg"`
	Year       *int    `json:"year,omitempty" example:"1969"`
	SpotifyID  *string `json:"spotify_id,omitempty" example:"4oDw9mW4Sro2zN1RHzlvOr"`
	Listened   *bool   `json:"listened,omitempty" example:"true"`
	Rating     *int    `json:"rating,omitempty" example:"5"`
	Comment    *string `json:"comment,omitempty" example:"Classic album"`
	ListenedAt *string `json:"listened_at,omitempty" example:"2024-01-15"`
}

// ArtistIDRequest represents a request body with just an artist ID
type ArtistIDRequest struct {
	ArtistID int `json:"artist_id" example:"1"`
}

// GetAlbums godoc
// @Summary Get all albums
// @Description Get all albums for the authenticated user
// @Tags albums
// @Produce json
// @Security Bearer
// @Success 200 {array} models.Album
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/albums [get]
func GetAlbums(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "GET /api/albums", "method", r.Method, "path", r.URL.Path)
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	albums, err := models.GetAlbumsByUser(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to get albums: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, albums)
}

func checkAlbumExistsByID(w http.ResponseWriter, userID int, artistID int, idStr string) (int, bool) {
	albumID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid album ID"))
		return 0, false
	}

	exists, err := models.AlbumExistsByID(userID, artistID, albumID)
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
// @Param request body ArtistIDRequest true "Artist ID"
// @Success 200 {object} models.Album
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Album or artist not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/albums/{id} [get]
func GetAlbum(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "GET /api/albums/{id}", "method", r.Method, "path", r.URL.Path)

	var body ArtistIDRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	if body.ArtistID <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError("invalid artist ID"))
		return
	}

	exists, err := models.ArtistExistsByID(userID, body.ArtistID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check existing artist: "+err.Error()))
		return
	}

	if !exists {
		writeJSON(w, http.StatusNotFound, apiError("artist not found"))
		return
	}

	albumID, ok := checkAlbumExistsByID(w, userID, body.ArtistID, r.PathValue("id"))
	if !ok {
		return
	}

	album, err := models.GetAlbumByID(userID, body.ArtistID, albumID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to get album: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, album)
}

// CreateAlbum godoc
// @Summary Create a new album
// @Description Create a new album for an artist
// @Tags albums
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateAlbumRequest true "Album data"
// @Success 201 {object} models.Album
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 409 {object} map[string]string "Album already exists"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/albums [post]
func CreateAlbum(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "POST /api/albums", "method", r.Method, "path", r.URL.Path)
	var body CreateAlbumRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	if body.Title == "" {
		writeJSON(w, http.StatusBadRequest, apiError("title is required"))
		return
	}

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	exists, err := models.ArtistExistsByID(userID, body.ArtistID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check existing artist: "+err.Error()))
		return
	}

	if !exists {
		writeJSON(w, http.StatusBadRequest, apiError("artist does not exist"))
		return
	}

	exists, err = models.AlbumExistsByName(userID, body.ArtistID, body.Title)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check existing album: "+err.Error()))
		return
	}

	if exists {
		writeJSON(w, http.StatusConflict, apiError("album with this name and artist already exists"))
		return
	}

	album, err := models.CreateAlbum(userID, body.ArtistID, body.Title)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to create album: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, album)
}

// UpdateAlbum godoc
// @Summary Update an album
// @Description Update an album's information
// @Tags albums
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Album ID"
// @Param request body UpdateAlbumRequest true "Update data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid request or no fields provided"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Album or artist not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/albums/{id} [put]
func UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "PUT /api/albums/{id}", "method", r.Method, "path", r.URL.Path)

	var body UpdateAlbumRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	if body.ArtistID <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError("invalid artist ID"))
		return
	}

	if body.Title == nil && body.CoverURL == nil && body.Year == nil && body.SpotifyID == nil && body.Listened == nil && body.Rating == nil && body.Comment == nil && body.ListenedAt == nil {
		writeJSON(w, http.StatusBadRequest, apiError("at least one field must be provided"))
		return
	}

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	exists, err := models.ArtistExistsByID(userID, body.ArtistID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check existing artist: "+err.Error()))
		return
	}

	if !exists {
		writeJSON(w, http.StatusNotFound, apiError("artist not found"))
		return
	}

	albumID, ok := checkAlbumExistsByID(w, userID, body.ArtistID, r.PathValue("id"))
	if !ok {
		return
	}

	err = models.UpdateAlbum(userID, body.ArtistID, albumID, body.Title, body.CoverURL, body.Year, body.SpotifyID, body.Listened, body.Rating, body.Comment, body.ListenedAt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to update album: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "album updated"})
}

// DeleteAlbum godoc
// @Summary Delete an album
// @Description Delete an album by ID
// @Tags albums
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Album ID"
// @Param request body ArtistIDRequest true "Artist ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Album or artist not found"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/albums/{id} [delete]
func DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "DELETE /api/albums/{id}", "method", r.Method, "path", r.URL.Path)

	var body ArtistIDRequest

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	if body.ArtistID <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError("invalid artist ID"))
		return
	}

	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	exists, err := models.ArtistExistsByID(userID, body.ArtistID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check existing artist: "+err.Error()))
		return
	}

	if !exists {
		writeJSON(w, http.StatusNotFound, apiError("artist not found"))
		return
	}

	albumID, ok := checkAlbumExistsByID(w, userID, body.ArtistID, r.PathValue("id"))
	if !ok {
		return
	}

	err = models.DeleteAlbum(userID, body.ArtistID, albumID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to delete album: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "album deleted"})
}
