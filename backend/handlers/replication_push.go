package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Chanadu/better-music/models"
	"github.com/lib/pq"
)

func (h *Handler) PushLibrary(writer http.ResponseWriter, httpRequest *http.Request) {
	userID, ok := getUserID(writer, httpRequest)
	if !ok {
		return
	}

	kind, ok := syncKind(httpRequest)
	if !ok {
		writeJSON(writer, 404, apiError("unknown collection"))
		return
	}

	var input PushLibraryRequest
	if !decodeSync(writer, httpRequest, &input) {
		return
	}

	if len(input.Rows) == 0 || len(input.Rows) > 100 || !validUUID(input.Epoch) {
		writeJSON(writer, 400, apiError("invalid push batch or epoch"))
		return
	}

	docs, err := validatePushRows(input.Rows, kind)
	if err != nil {
		writeJSON(writer, 422, apiError(err.Error()))
		return
	}

	tx, storageErr := h.Database.BeginTx(httpRequest.Context(), nil)
	if storageErr != nil {
		syncStorageError(writer, httpRequest, "begin push transaction", storageErr)
		return
	}

	defer tx.Rollback()
	epoch, storageErr := models.LockSyncState(tx, userID)
	if storageErr != nil {
		syncStorageError(writer, httpRequest, "lock sync state", storageErr)
		return
	}

	if epoch != input.Epoch {
		writeJSON(writer, 409, apiError("Sync history changed. Preserve pending edits before resetting."))
		return
	}

	conflicts := []json.RawMessage{}
	failures := []LibraryChangeFailure{}
	retryable := false
	for index, doc := range docs {
		if storageErr = models.CreateSyncSavepoint(tx); storageErr != nil {
			syncStorageError(writer, httpRequest, "create sync savepoint", storageErr)
			return
		}

		conflict, failure, err := applySyncRow(tx, userID, kind, doc, input.Rows[index].Assumed)
		if err != nil {
			if storageErr = models.RollbackSyncSavepoint(tx); storageErr != nil {
				syncStorageError(writer, httpRequest, "roll back sync savepoint", storageErr)
				return
			}

			var pe *pq.Error
			if errors.As(err, &pe) && (pe.Code == "23505" || pe.Code == "23503" || pe.Code == "23514") {
				failure = &LibraryChangeFailure{ID: doc.Fields().ID, Error: "A record with this name or ID already exists, or its relationship is invalid."}
			} else {
				syncStorageError(writer, httpRequest, "apply library change", err)
				return
			}
		}

		if failure != nil {
			failures = append(failures, *failure)
			retryable = retryable || failure.Retryable
		} else if conflict != nil {
			conflicts = append(conflicts, conflict)
		}

		if storageErr = models.ReleaseSyncSavepoint(tx); storageErr != nil {
			syncStorageError(writer, httpRequest, "release sync savepoint", storageErr)
			return
		}
	}

	if storageErr = tx.Commit(); storageErr != nil {
		syncStorageError(writer, httpRequest, "commit push transaction", storageErr)
		return
	}

	if len(failures) > 0 {
		status := 422
		if retryable {
			status = 409
		}

		writeJSON(writer, status, PushLibraryErrorResponse{Error: "Some library changes could not sync. They remain saved on this device.", Failures: failures})
		return
	}

	writeJSON(writer, 200, conflicts)
}
