package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/Chanadu/better-music/middleware"
	"github.com/lib/pq"
)

func isPostgresError(err error, code pq.ErrorCode) bool {
	var pqError *pq.Error
	return errors.As(err, &pqError) && pqError.Code == code
}

func isPostgresConstraintError(err error, code pq.ErrorCode, constraint string) bool {
	var pqError *pq.Error
	return errors.As(err, &pqError) && pqError.Code == code && pqError.Constraint == constraint
}

type ApiErrorResponse struct {
	Error string `json:"error" validate:"required"`
}

type MessageResponse struct {
	Message string `json:"message" validate:"required"`
}

func encodeJSON(value any) ([]byte, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

func writeEncodedJSON(w http.ResponseWriter, status int, headers map[string]string, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	for key, value := range headers {
		w.Header().Set(key, value)
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
	slog.Info("Status: " + http.StatusText(status))
	slog.Info("Response: " + strings.TrimSuffix(string(body), "\n"))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	body, err := encodeJSON(value)
	if err != nil {
		status = http.StatusInternalServerError
		body, _ = encodeJSON(apiError("failed to encode response"))
	}
	writeEncodedJSON(w, status, nil, body)
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
