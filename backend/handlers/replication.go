package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

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

func decodeSync(w http.ResponseWriter, r *http.Request, out any) bool {
	if err := decodeJSON(http.MaxBytesReader(w, r.Body, 2<<20), out); err != nil {
		writeJSON(w, 400, apiError("invalid or oversized sync request"))
		return false
	}

	return true
}

func syncKind(r *http.Request) (models.LibraryKind, bool) {
	switch r.PathValue("kind") {
	case "artists":
		return models.ArtistKind, true
	case "albums":
		return models.AlbumKind, true
	default:
		return models.UnknownLibraryKind, false
	}
}

func syncStorageError(w http.ResponseWriter, e error) {
	if errors.Is(e, sql.ErrNoRows) {
		writeJSON(w, 401, apiError("account no longer exists"))
		return
	}

	writeJSON(w, 500, apiError("library storage is unavailable"))
}
