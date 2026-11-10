package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Chanadu/better-music/models"
	"github.com/lib/pq"
)

func (h *Handler) PushLibrary(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	kind, ok := syncKind(r)
	if !ok {
		writeJSON(w, 404, apiError("unknown collection"))
		return
	}

	var input PushLibraryRequest
	if !decodeSync(w, r, &input) {
		return
	}

	if len(input.Rows) == 0 || len(input.Rows) > 100 || !validUUID(input.Epoch) {
		writeJSON(w, 400, apiError("invalid push batch or epoch"))
		return
	}

	docs, err := validatePushRows(input.Rows, kind)
	if err != nil {
		writeJSON(w, 422, apiError(err.Error()))
		return
	}

	tx, e := h.Database.BeginTx(r.Context(), nil)
	if e != nil {
		syncStorageError(w, e)
		return
	}

	defer tx.Rollback()
	epoch, e := models.LockSyncState(tx, userID)
	if e != nil {
		syncStorageError(w, e)
		return
	}

	if epoch != input.Epoch {
		writeJSON(w, 409, apiError("Sync history changed. Preserve pending edits before resetting."))
		return
	}

	conflicts := []json.RawMessage{}
	failures := []LibraryChangeFailure{}
	retryable := false
	for i, doc := range docs {
		if e = models.CreateSyncSavepoint(tx); e != nil {
			syncStorageError(w, e)
			return
		}

		conflict, failure, err := applySyncRow(tx, userID, kind, doc, input.Rows[i].Assumed)
		if err != nil {
			if e = models.RollbackSyncSavepoint(tx); e != nil {
				syncStorageError(w, e)
				return
			}

			var pe *pq.Error
			if errors.As(err, &pe) && (pe.Code == "23505" || pe.Code == "23503" || pe.Code == "23514") {
				failure = &LibraryChangeFailure{ID: doc.Fields().ID, Error: "A record with this name or ID already exists, or its relationship is invalid."}
			} else {
				syncStorageError(w, err)
				return
			}
		}

		if failure != nil {
			failures = append(failures, *failure)
			retryable = retryable || failure.Retryable
		} else if conflict != nil {
			conflicts = append(conflicts, conflict)
		}

		if e = models.ReleaseSyncSavepoint(tx); e != nil {
			syncStorageError(w, e)
			return
		}
	}

	if e = tx.Commit(); e != nil {
		syncStorageError(w, e)
		return
	}

	if len(failures) > 0 {
		status := 422
		if retryable {
			status = 409
		}

		writeJSON(w, status, PushLibraryErrorResponse{Error: "Some library changes could not sync. They remain saved on this device.", Failures: failures})
		return
	}

	writeJSON(w, 200, conflicts)
}
