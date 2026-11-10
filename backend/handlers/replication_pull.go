package handlers

import (
	"net/http"
	"strconv"

	"github.com/Chanadu/better-music/models"
)

func (h *Handler) PullLibrary(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	kind, ok := syncKind(r)
	if !ok {
		writeJSON(w, 404, apiError("unknown collection"))
		return
	}

	var input PullLibraryRequest
	if !decodeSync(w, r, &input) {
		return
	}

	if input.Limit < 1 || input.Limit > 500 {
		writeJSON(w, 400, apiError("limit must be between 1 and 500"))
		return
	}

	seq := int64(0)
	if input.Checkpoint != nil {
		var e error
		seq, e = strconv.ParseInt(input.Checkpoint.Sequence, 10, 64)
		if e != nil || seq < 0 {
			writeJSON(w, 400, apiError("invalid checkpoint"))
			return
		}
	}

	tx, e := h.Database.BeginTx(r.Context(), nil)
	if e != nil {
		writeJSON(w, 500, apiError("sync unavailable"))
		return
	}

	defer tx.Rollback()
	epoch, e := models.LockSyncState(tx, userID)
	if e != nil {
		syncStorageError(w, e)
		return
	}

	if input.Checkpoint != nil && input.Checkpoint.Epoch != epoch {
		writeJSON(w, 409, apiError("Sync history changed. Export pending edits before resetting this device."))
		return
	}

	published, e := models.GetSyncSequence(tx, userID)
	if e != nil {
		syncStorageError(w, e)
		return
	}

	if seq > published {
		writeJSON(w, 409, apiError("Checkpoint is ahead of the server. Preserve local edits before resetting."))
		return
	}

	docs, seq, e := models.PullLibraryDocuments(tx, userID, kind, seq, input.Limit)
	if e != nil {
		syncStorageError(w, e)
		return
	}

	if e = tx.Commit(); e != nil {
		syncStorageError(w, e)
		return
	}

	writeJSON(w, 200, PullLibraryResponse{Documents: docs, Checkpoint: LibraryCheckpoint{Epoch: epoch, Sequence: strconv.FormatInt(seq, 10)}})
}
