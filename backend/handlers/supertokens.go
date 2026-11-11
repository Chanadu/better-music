package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Chanadu/better-music/internal/auth"
	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/middleware"
	"github.com/Chanadu/better-music/models"
)

func (h *Handler) SessionInfo(writer http.ResponseWriter, httpRequest *http.Request) {
	id, ok := getUserID(writer, httpRequest)

	if !ok {
		return
	}

	email, epoch, err := models.ActiveAccountSession(httpRequest.Context(), h.Database, id)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.ErrorContext(httpRequest.Context(), "load account session failed", "user_id", id, logger.ErrorDetails(err))
		}
		writeJSON(writer, 503, apiError("account unavailable"))
		return
	}

	writeJSON(writer, 200, map[string]any{"user_id": id, "email": email, "sync_epoch": epoch})
}

func (h *Handler) ChangeSuperTokensAccount(writer http.ResponseWriter, httpRequest *http.Request) {
	id, ok := getUserID(writer, httpRequest)

	if !ok {
		return
	}

	var body struct {
		Email           string `json:"email"`
		Password        string `json:"password"`
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(http.MaxBytesReader(writer, httpRequest.Body, 16384)).Decode(&body); err != nil {
		writeJSON(writer, 400, apiError("invalid account request"))
		return
	}

	kind, password, value := auth.DeleteAccount, body.Password, ""

	switch httpRequest.URL.Path {
	case "/api/account/email":
		kind = auth.ChangeEmail

		var valid bool
		value, valid = normalizeEmail(body.Email)

		if !valid {
			writeJSON(writer, 400, apiError("valid email required"))
			return
		}

	case "/api/account/password":
		kind, password, value = auth.ChangePassword, body.CurrentPassword, body.NewPassword
	}

	if password == "" {
		writeJSON(writer, 400, apiError("current password required"))

		return
	}

	if err := h.Auth.Change(httpRequest.Context(), id, kind, password, value); err != nil {
		status, message := 503, "Account change could not finish. Try signing in again; pending changes will be retried."

		switch {
		case errors.Is(err, auth.ErrInvalidInput), errors.Is(err, auth.ErrCurrentPassword):
			status, message = 400, err.Error()

		case errors.Is(err, auth.ErrEmailExists):
			status, message = 409, "email already in use"

		case errors.Is(err, auth.ErrAccountDisabled):
			status, message = 403, "account is disabled"

		default:
			slog.ErrorContext(httpRequest.Context(), "account change failed", "user_id", id, "operation", kind, logger.ErrorDetails(err))
		}

		writeJSON(writer, status, apiError(message))
		return
	}

	writeJSON(writer, 200, apiMessage("account updated"))
}

func getUserID(writer http.ResponseWriter, httpRequest *http.Request) (string, bool) {
	userID, ok := middleware.GetUserID(httpRequest)

	if !ok {
		writeJSON(writer, http.StatusUnauthorized, apiError("unauthorized"))
		return "", false
	}

	return userID, true
}
