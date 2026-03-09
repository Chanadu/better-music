package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Chanadu/better-music/middleware"
	"github.com/Chanadu/better-music/models"
)

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

	userID := middleware.GetUserID(r)

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

	userID := middleware.GetUserID(r)

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
