package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Chanadu/better-music/middleware"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
	slog.Info("Status: " + http.StatusText(status))
	s, err := json.Marshal(v)
	if err == nil {
		slog.Info("Response: " + string(s))
	}
}

func apiError(msg string) map[string]string {
	return map[string]string{"error": msg}
}

func isEmpty(s *string) bool {
	return s == nil || *s == ""
}

func getUserID(w http.ResponseWriter, r *http.Request) (int, bool) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized"))
		return 0, false
	}

	return userID, true
}
