package handlers

import (
	"context"
	"strconv"

	"github.com/Chanadu/better-music/models"
	"github.com/danielgtaylor/huma/v2"
)

func (h *Handler) pullLibrary(ctx context.Context, userID string, kind models.LibraryKind, input PullLibraryRequest) (*jsonOutput[PullLibraryResponse], error) {
	if input.Limit < 1 || input.Limit > 500 {
		return nil, huma.Error400BadRequest("limit must be between 1 and 500")
	}

	seq := int64(0)
	if input.Checkpoint != nil {
		var err error

		seq, err = strconv.ParseInt(input.Checkpoint.Sequence, 10, 64)
		if err != nil || seq < 0 {
			return nil, huma.Error400BadRequest("invalid checkpoint")
		}
	}

	tx, err := h.Database.BeginTx(ctx, nil)
	if err != nil {
		logSyncFailure(ctx, userID, kind, "begin pull transaction", err)
		return nil, huma.Error500InternalServerError("sync unavailable")
	}

	defer tx.Rollback()
	epoch, err := models.LockSyncState(tx, userID)
	if err != nil {
		return nil, syncStorageError(ctx, userID, kind, "lock sync state", err)
	}

	if input.Checkpoint != nil && input.Checkpoint.Epoch != epoch {
		return nil, huma.Error409Conflict("Sync history changed. Export pending edits before resetting this device.")
	}

	published, err := models.GetSyncSequence(tx, userID)
	if err != nil {
		return nil, syncStorageError(ctx, userID, kind, "read published sequence", err)
	}

	if seq > published {
		return nil, huma.Error409Conflict("Checkpoint is ahead of the server. Preserve local edits before resetting.")
	}

	docs, seq, err := models.PullLibraryDocuments(tx, userID, kind, seq, input.Limit)
	if err != nil {
		return nil, syncStorageError(ctx, userID, kind, "pull library documents", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, syncStorageError(ctx, userID, kind, "commit pull transaction", err)
	}

	return &jsonOutput[PullLibraryResponse]{Body: PullLibraryResponse{Documents: docs, Checkpoint: LibraryCheckpoint{Epoch: epoch, Sequence: strconv.FormatInt(seq, 10)}}}, nil
}
