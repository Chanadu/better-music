package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/Chanadu/better-music/middleware"
)

type ApiErrorResponse struct {
	Error string `json:"error" validate:"required"`
}

type MessageResponse struct {
	Message string `json:"message" validate:"required"`
}

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

func apiError(msg string) ApiErrorResponse {
	return ApiErrorResponse{Error: msg}
}

func apiMessage(msg string) MessageResponse {
	return MessageResponse{Message: msg}
}

func normalizeEmail(email string) (string, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", false
	}

	return email, true
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
