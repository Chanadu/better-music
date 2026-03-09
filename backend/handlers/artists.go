package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Chanadu/better-music/middleware"
	"github.com/Chanadu/better-music/models"
)

func GetArtists(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "GET /api/artists", "method", r.Method, "path", r.URL.Path)
	userID := middleware.GetUserID(r)

	artists, err := models.GetArtistsByUser(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to get artists: "+err.Error()))
		return
	}

	// writeJSON(w, http.StatusOK, fmt.Sprintf("Get artists for user %d", userID))
	writeJSON(w, http.StatusOK, artists)
}

func CreateArtist(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "POST /api/artists", "method", r.Method, "path", r.URL.Path)
	var body struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	if body.Name == "" {
		writeJSON(w, http.StatusBadRequest, apiError("name is required"))
		return
	}

	userID := middleware.GetUserID(r)

	exists, err := models.ArtistExistsByName(userID, body.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check existing artist: "+err.Error()))
		return
	}

	if exists {
		writeJSON(w, http.StatusConflict, apiError("artist with this name already exists"))
		return
	}

	artist, err := models.CreateArtist(userID, body.Name)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to create artist: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, artist)
}

func checkArtistExistsByID(w http.ResponseWriter, userID int, idStr string) (int, bool) {
	artistID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid artist ID"))
		return 0, false
	}
	exists, err := models.ArtistExistsByID(userID, artistID)
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

func DeleteArtist(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "DELETE /api/artists/{id}", "method", r.Method, "path", r.URL.Path)
	userID := middleware.GetUserID(r)

	artistID, ok := checkArtistExistsByID(w, userID, r.PathValue("id"))
	if !ok {
		return
	}

	albums, err := models.GetArtistAlbums(userID, artistID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check artist's albums: "+err.Error()))
		return
	}

	if len(albums) > 0 {
		writeJSON(w, http.StatusBadRequest, apiError("artist cannot be deleted because it has albums"))
		return
	}

	err = models.DeleteArtist(userID, artistID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to delete artist: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "artist deleted"})
}

func UpdateArtist(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "PUT /api/artists/{id}", "method", r.Method, "path", r.URL.Path)
	var body struct {
		Name      *string `json:"name,omitempty"`
		SpotifyID *string `json:"spotify_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON: "+err.Error()))
		return
	}

	if isEmpty(body.Name) && isEmpty(body.SpotifyID) {
		writeJSON(w, http.StatusBadRequest, apiError("at least one field must be provided"))
		return
	}

	userID := middleware.GetUserID(r)

	artistID, ok := checkArtistExistsByID(w, userID, r.PathValue("id"))
	if !ok {
		return
	}

	err := models.UpdateArtist(userID, artistID, body.Name, body.SpotifyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to update artist: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "artist updated"})
}

func GetArtistAlbums(w http.ResponseWriter, r *http.Request) {
	slog.Debug("route hit", "route", "GET /api/artists/{id}/albums", "method", r.Method, "path", r.URL.Path)
	userID := middleware.GetUserID(r)

	artistID, ok := checkArtistExistsByID(w, userID, r.PathValue("id"))
	if !ok {
		return
	}

	albums, err := models.GetArtistAlbums(userID, artistID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to get artist's albums: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, albums)

}
