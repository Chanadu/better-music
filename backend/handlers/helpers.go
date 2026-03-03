package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
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
