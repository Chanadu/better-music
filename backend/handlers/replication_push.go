package handlers

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Chanadu/better-music/models"
	"github.com/danielgtaylor/huma/v2"
	"github.com/lib/pq"
)

func (h *Handler) pushLibrary(ctx context.Context, userID string, kind models.LibraryKind, input PushLibraryRequest) (*jsonOutput[[]json.RawMessage], error) {
	if len(input.Rows) == 0 || len(input.Rows) > 100 || !validUUID(input.Epoch) {
		return nil, huma.Error400BadRequest("invalid push batch or epoch")
	}

	docs, err := validatePushRows(input.Rows, kind)
	if err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}

	tx, storageErr := h.Database.BeginTx(ctx, nil)
	if storageErr != nil {
		return nil, syncStorageError(ctx, userID, kind, "begin push transaction", storageErr)
	}

	defer tx.Rollback()
	epoch, storageErr := models.LockSyncState(tx, userID)
	if storageErr != nil {
		return nil, syncStorageError(ctx, userID, kind, "lock sync state", storageErr)
	}

	if epoch != input.Epoch {
		return nil, huma.Error409Conflict("Sync history changed. Preserve pending edits before resetting.")
	}

	conflicts := []json.RawMessage{}
	failures := []LibraryChangeFailure{}
	retryable := false

	for index, doc := range docs {
		if storageErr = models.CreateSyncSavepoint(tx); storageErr != nil {
			return nil, syncStorageError(ctx, userID, kind, "create sync savepoint", storageErr)
		}

		conflict, failure, err := applySyncRow(tx, userID, kind, doc, input.Rows[index].Assumed)
		if err != nil {
			if storageErr = models.RollbackSyncSavepoint(tx); storageErr != nil {
				return nil, syncStorageError(ctx, userID, kind, "roll back sync savepoint", storageErr)
			}

			var pe *pq.Error
			if errors.As(err, &pe) && (pe.Code == "23505" || pe.Code == "23503" || pe.Code == "23514") {
				failure = &LibraryChangeFailure{ID: doc.Fields().ID, Error: "A record with this name or ID already exists, or its relationship is invalid."}
			} else {
				return nil, syncStorageError(ctx, userID, kind, "apply library change", err)
			}
		}

		if failure != nil {
			failures = append(failures, *failure)
			retryable = retryable || failure.Retryable
		} else if conflict != nil {
			conflicts = append(conflicts, conflict)
		}

		if storageErr = models.ReleaseSyncSavepoint(tx); storageErr != nil {
			return nil, syncStorageError(ctx, userID, kind, "release sync savepoint", storageErr)
		}
	}

	if storageErr = tx.Commit(); storageErr != nil {
		return nil, syncStorageError(ctx, userID, kind, "commit push transaction", storageErr)
	}

	if len(failures) > 0 {
		status := 422
		if retryable {
			status = 409
		}

		return nil, pushLibraryFailure{Status: status, Message: "Some library changes could not sync. They remain saved on this device.", Failures: failures}
	}

	return &jsonOutput[[]json.RawMessage]{Body: conflicts}, nil
}
