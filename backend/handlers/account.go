package handlers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/models"
)

type AccountResponse struct {
	Email string `json:"email" validate:"required"`
}

func (h *Handler) GetAccount(writer http.ResponseWriter, httpRequest *http.Request) {
	id, ok := getUserID(writer, httpRequest)
	if !ok {
		return
	}
	email, err := models.ActiveAccountEmail(httpRequest.Context(), h.Database, id)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.ErrorContext(httpRequest.Context(), "load account email failed", "user_id", id, logger.ErrorDetails(err))
		}
		writeJSON(writer, 503, apiError("account unavailable"))
		return
	}

	writeJSON(writer, 200, AccountResponse{Email: email})
}

func normalizeEmail(email string) (string, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", false
	}

	return email, true
}
