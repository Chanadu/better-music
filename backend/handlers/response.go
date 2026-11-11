package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Chanadu/better-music/logger"
)

type ApiErrorResponse struct {
	Error string `json:"error" validate:"required"`
}

type MessageResponse struct {
	Message string `json:"message" validate:"required"`
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		slog.Error("encode JSON response failed", "status", status, logger.ErrorDetails(err))
		status = http.StatusInternalServerError
		body, _ = json.Marshal(apiError("failed to encode response"))
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, _ = writer.Write(append(body, '\n'))
	slog.Debug("JSON response sent", "status", status)
}

func apiError(msg string) ApiErrorResponse {
	return ApiErrorResponse{Error: msg}
}

func apiMessage(msg string) MessageResponse {
	return MessageResponse{Message: msg}
}
