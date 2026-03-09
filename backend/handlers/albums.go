package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Chanadu/better-music/middleware"
	"github.com/Chanadu/better-music/models"
)

func CreateAlbum(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ArtistID int    `json:"artist_id"`
		Title    string `json:"title"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("invalid JSON"+err.Error()))
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
		writeJSON(w, http.StatusBadRequest, apiError("artist with does not exist"))
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
