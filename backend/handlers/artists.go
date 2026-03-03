package handlers

import (
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
