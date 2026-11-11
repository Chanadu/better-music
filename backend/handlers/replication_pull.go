package handlers

import (
	"net/http"
	"strconv"

	"github.com/Chanadu/better-music/models"
)

func (h *Handler) PullLibrary(writer http.ResponseWriter, httpRequest *http.Request) {
	userID, ok := getUserID(writer, httpRequest)
	if !ok {
		return
	}

	kind, ok := syncKind(httpRequest)
	if !ok {
		writeJSON(writer, 404, apiError("unknown collection"))
		return
	}

	var input PullLibraryRequest
	if !decodeSync(writer, httpRequest, &input) {
		return
	}

	if input.Limit < 1 || input.Limit > 500 {
		writeJSON(writer, 400, apiError("limit must be between 1 and 500"))
		return
	}

	seq := int64(0)
	if input.Checkpoint != nil {
		var err error
		seq, err = strconv.ParseInt(input.Checkpoint.Sequence, 10, 64)
		if err != nil || seq < 0 {
			writeJSON(writer, 400, apiError("invalid checkpoint"))
			return
		}
	}

	tx, err := h.Database.BeginTx(httpRequest.Context(), nil)
	if err != nil {
		logSyncFailure(httpRequest, "begin pull transaction", err)
		writeJSON(writer, 500, apiError("sync unavailable"))
		return
	}

	defer tx.Rollback()
	epoch, err := models.LockSyncState(tx, userID)
	if err != nil {
		syncStorageError(writer, httpRequest, "lock sync state", err)
		return
	}

	if input.Checkpoint != nil && input.Checkpoint.Epoch != epoch {
		writeJSON(writer, 409, apiError("Sync history changed. Export pending edits before resetting this device."))
		return
	}

	published, err := models.GetSyncSequence(tx, userID)
	if err != nil {
		syncStorageError(writer, httpRequest, "read published sequence", err)
		return
	}

	if seq > published {
		writeJSON(writer, 409, apiError("Checkpoint is ahead of the server. Preserve local edits before resetting."))
		return
	}

	docs, seq, err := models.PullLibraryDocuments(tx, userID, kind, seq, input.Limit)
	if err != nil {
		syncStorageError(writer, httpRequest, "pull library documents", err)
		return
	}

	if err = tx.Commit(); err != nil {
		syncStorageError(writer, httpRequest, "commit pull transaction", err)
		return
	}

	writeJSON(writer, 200, PullLibraryResponse{Documents: docs, Checkpoint: LibraryCheckpoint{Epoch: epoch, Sequence: strconv.FormatInt(seq, 10)}})
}
