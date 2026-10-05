package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Chanadu/better-music/models"
)

type ArtistConflictResponse struct {
	Error   string         `json:"error" validate:"required"`
	Current *models.Artist `json:"current" validate:"required"`
}
type AlbumConflictResponse struct {
	Error   string        `json:"error" validate:"required"`
	Current *models.Album `json:"current" validate:"required"`
}

type mutationResponse struct {
	status  int
	body    any
	version int
}

func mutationErrorResponse(status int, message string) mutationResponse {
	return mutationResponse{status: status, body: apiError(message)}
}

func (h *Handler) handleLibraryMutation(w http.ResponseWriter, r *http.Request, entity models.LibraryEntity) {
	userID, ok := getUserID(w, r)
	if !ok {
		return
	}

	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeJSON(w, 400, apiError("request body is too large or unreadable"))
		return
	}

	key := r.Header.Get("Idempotency-Key")
	if len(r.Header.Values("Idempotency-Key")) > 1 || (key != "" && (strings.TrimSpace(key) == "" || len(key) > 255)) {
		writeJSON(w, 400, apiError("invalid Idempotency-Key"))
		return
	}

	fingerprint, _ := json.Marshal([]string{r.Method, r.URL.RequestURI(), r.Header.Get("If-Match"), string(raw)})
	digest := sha256.Sum256(fingerprint)
	hash := hex.EncodeToString(digest[:])

	tx, err := h.Database.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, 500, apiError("failed to start mutation"))
		return
	}

	defer tx.Rollback()

	if err = models.LockUserForMutation(tx, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, 401, apiError("account no longer exists"))
		} else {
			writeJSON(w, 500, apiError("failed to lock account"))
		}
		return
	}

	if key != "" {
		saved, err := models.GetProcessedMutation(tx, userID, key)

		if err == nil {
			if saved.RequestHash != hash {
				writeJSON(w, 409, apiError("Idempotency-Key was already used for a different request"))
			} else {
				writeEncodedJSON(w, saved.Status, saved.Headers, saved.Body)
			}

			return
		}

		if !errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, 500, apiError("failed to load mutation"))
			return
		}
	}

	if err = models.CreateLibrarySavepoint(tx); err != nil {
		writeJSON(w, 500, apiError("failed to prepare mutation"))
		return
	}

	result, err := applyLibraryMutation(tx, userID, entity, r, raw)

	if err != nil {
		if rollbackErr := models.RollbackToLibrarySavepoint(tx); rollbackErr != nil {
			writeJSON(w, 500, apiError("failed to roll back mutation"))
			return
		}

		switch {
		case isPostgresError(err, "23505"):
			result = mutationErrorResponse(409, "record with this name already exists")

		case isPostgresError(err, "23503"):
			result = mutationErrorResponse(400, "artist does not exist or still has albums")

		case isPostgresError(err, "22007"), isPostgresError(err, "22008"), isPostgresError(err, "22003"), isPostgresError(err, "23502"), isPostgresError(err, "23514"):
			result = mutationErrorResponse(400, "invalid field value")

		default:
			writeJSON(w, 500, apiError("failed to apply mutation"))
			return
		}
	}

	response, err := encodeMutationResponse(result)
	if err != nil {
		writeJSON(w, 500, apiError("failed to encode mutation response"))
		return
	}

	if key != "" {
		response.RequestHash = hash
		if err = models.SaveProcessedMutation(tx, userID, key, response); err != nil {
			writeJSON(w, 500, apiError("failed to save mutation response"))
			return
		}
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, 500, apiError("failed to commit mutation"))
		return
	}

	writeEncodedJSON(w, response.Status, response.Headers, response.Body)
}

func encodeMutationResponse(result mutationResponse) (models.ProcessedMutation, error) {
	body, err := encodeJSON(result.body)
	if err != nil {
		return models.ProcessedMutation{}, err
	}

	response := models.ProcessedMutation{
		Status:  result.status,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    body,
	}

	if result.version > 0 {
		response.Headers["ETag"] = fmt.Sprintf(`"%d"`, result.version)
	}

	return response, nil
}
