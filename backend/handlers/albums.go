package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Chanadu/better-music/models"
)

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

func GetAlbum(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "GET /api/albums/{id}", "method", r.Method, "path", r.URL.Path)

	var body struct {
		ArtistID int `json:"artist_id"`
	}

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

func CreateAlbum(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "POST /api/albums", "method", r.Method, "path", r.URL.Path)
	var body struct {
		ArtistID int    `json:"artist_id"`
		Title    string `json:"title"`
	}

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

func UpdateAlbum(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "PUT /api/albums/{id}", "method", r.Method, "path", r.URL.Path)

	var body struct {
		ArtistID  int     `json:"artist_id"`
		Title     *string `json:"title,omitempty"`
		CoverURL  *string `json:"cover_url,omitempty"`
		Year      *int    `json:"year,omitempty"`
		SpotifyID *string `json:"spotify_id,omitempty"`
		Listened  *bool   `json:"listened,omitempty"`
		Rating    *int    `json:"rating,omitempty"`
		Comment   *string `json:"comment,omitempty"`
		ListenedAt *string `json:"listened_at,omitempty"`
	}

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

func DeleteAlbum(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "DELETE /api/albums/{id}", "method", r.Method, "path", r.URL.Path)

	var body struct {
		ArtistID int `json:"artist_id"`
	}

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
