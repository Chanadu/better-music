package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Chanadu/better-music/logger"
	"github.com/Chanadu/better-music/middleware"
	"github.com/Chanadu/better-music/models"
)

type LibraryCheckpoint struct {
	Epoch    string `json:"epoch"`
	Sequence string `json:"sequence"`
}

type PullLibraryRequest struct {
	Checkpoint *LibraryCheckpoint `json:"checkpoint"`
	Limit      int                `json:"limit"`
}

type LibraryChange struct {
	Assumed json.RawMessage `json:"assumedMasterState"`
	New     json.RawMessage `json:"newDocumentState"`
}

type PushLibraryRequest struct {
	Epoch string          `json:"epoch"`
	Rows  []LibraryChange `json:"rows"`
}

type LibraryChangeFailure struct {
	ID        string `json:"id"`
	Retryable bool   `json:"retryable"`
	Error     string `json:"error"`
}

type PullLibraryResponse struct {
	Documents  []json.RawMessage `json:"documents"`
	Checkpoint LibraryCheckpoint `json:"checkpoint"`
}

type PushLibraryErrorResponse struct {
	Error    string                 `json:"error"`
	Failures []LibraryChangeFailure `json:"failures"`
}

func decodeSync(writer http.ResponseWriter, httpRequest *http.Request, out any) bool {
	if err := decodeJSON(http.MaxBytesReader(writer, httpRequest.Body, 2<<20), out); err != nil {
		writeJSON(writer, 400, apiError("invalid or oversized sync request"))
		return false
	}

	return true
}

func syncKind(httpRequest *http.Request) (models.LibraryKind, bool) {
	switch httpRequest.PathValue("kind") {
	case "artists":
		return models.ArtistKind, true
	case "albums":
		return models.AlbumKind, true
	default:
		return models.UnknownLibraryKind, false
	}
}

func syncStorageError(writer http.ResponseWriter, httpRequest *http.Request, operation string, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(writer, 401, apiError("account no longer exists"))
		return
	}

	logSyncFailure(httpRequest, operation, err)
	writeJSON(writer, 500, apiError("library storage is unavailable"))
}

func logSyncFailure(httpRequest *http.Request, operation string, err error) {
	userID, _ := middleware.GetUserID(httpRequest)
	slog.ErrorContext(httpRequest.Context(), "library sync failed", "operation", operation, "collection", httpRequest.PathValue("kind"), "user_id", userID, logger.ErrorDetails(err))
}
