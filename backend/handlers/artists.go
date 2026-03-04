package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Chanadu/better-music/middleware"
	"github.com/Chanadu/better-music/models"
)

func GetArtists(w http.ResponseWriter, r *http.Request) {
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
	var body struct {
		SpotifyID string `json:"spotify_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON"+err.Error()))
		return
	}

	if body.SpotifyID == "" {
		writeJSON(w, http.StatusBadRequest, apiError("spotify_id is required"))
		return
	}

	if len(body.SpotifyID) > 255 {
		writeJSON(w, http.StatusBadRequest, apiError("spotify_id is too long"))
		return
	}

	userID := middleware.GetUserID(r)

	exists, err := models.ArtistExistsBySpotifyID(userID, body.SpotifyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to check existing artist: "+err.Error()))
		return
	}

	if exists {
		writeJSON(w, http.StatusConflict, apiError("artist with this spotify_id already exists"))
		return
	}

	artist, err := models.CreateArtist(userID, "filler name", body.SpotifyID)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to create artist: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, artist)
}
