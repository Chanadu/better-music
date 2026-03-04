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
		Name      string  `json:"name"`
		SpotifyID *string `json:"spotify_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON"+err.Error()))
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

	artist, err := models.CreateArtist(userID, body.Name, body.SpotifyID)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("failed to create artist: "+err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, artist)
}
