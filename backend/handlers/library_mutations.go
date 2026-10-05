package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Chanadu/better-music/models"
)

func applyLibraryMutation(tx *sql.Tx, userID int, entity models.LibraryEntity, r *http.Request, raw []byte) (mutationResponse, error) {
	expected, err := parseIfMatch(r)

	if err != nil {
		return mutationErrorResponse(400, err.Error()), nil
	}

	fields := map[string]any{}
	if r.Method != http.MethodDelete || (entity == models.AlbumEntity && len(raw) > 0) {
		fields, err = decodeMutationFields(raw, entity)

		if err != nil {
			return mutationErrorResponse(400, err.Error()), nil
		}
	}

	id := 0
	if r.Method != http.MethodPost {
		id, err = strconv.Atoi(r.PathValue("id"))

		if err != nil || id <= 0 || id > 2147483647 {
			return mutationErrorResponse(400, "invalid record ID"), nil
		}
	}

	var version int
	var artistID int

	if r.Method != http.MethodPost {
		version, artistID, err = models.LockLibraryRecord(tx, userID, entity, id)

		if errors.Is(err, sql.ErrNoRows) {
			if r.Method == http.MethodDelete {
				return mutationResponse{status: 200, body: apiMessage(strings.TrimSuffix(string(entity), "s") + " deleted")}, nil
			}

			return mutationErrorResponse(404, "record not found"), nil
		}

		if err != nil {
			return mutationResponse{}, err
		}

		if expected != 0 && expected != version {
			current, err := recordResponse(tx, userID, entity, id, artistID, 412)
			if err != nil {
				return mutationResponse{}, err
			}

			if entity == models.ArtistEntity {
				current.body = ArtistConflictResponse{Error: "record version conflict", Current: current.body.(*models.Artist)}
			} else {
				current.body = AlbumConflictResponse{Error: "record version conflict", Current: current.body.(*models.Album)}
			}

			return current, nil
		}
	}
	if entity == models.AlbumEntity {
		if supplied, ok := fields["artist_id"]; ok {
			if r.Method != http.MethodPost && supplied.(int) != artistID {
				return mutationErrorResponse(404, "album not found for this artist"), nil
			}

			artistID = supplied.(int)
		}

		if r.Method == http.MethodPost {
			if artistID <= 0 {
				return mutationErrorResponse(400, "artist_id is required"), nil
			}

			exists, err := models.ArtistExistsByID(tx, userID, artistID)
			if err != nil {
				return mutationResponse{}, err
			}

			if !exists {
				return mutationErrorResponse(400, "artist does not exist"), nil
			}
		}
	}

	if r.Method == http.MethodDelete {
		if err = models.DeleteLibraryRecord(tx, userID, entity, id); err != nil {
			return mutationResponse{}, err
		}

		return mutationResponse{status: 200, body: apiMessage(strings.TrimSuffix(string(entity), "s") + " deleted")}, nil
	}

	if entity == models.AlbumEntity {
		if listened, ok := fields["listened"]; ok && !listened.(bool) {
			fields["rating"] = nil
			fields["listened_at"] = nil
		}
	}

	creating := r.Method == http.MethodPost

	if creating {
		required := "name"
		if entity == models.AlbumEntity {
			required = "title"
		}

		if _, ok := fields[required]; !ok {
			return mutationErrorResponse(400, required+" is required"), nil
		}
	}

	id, err = models.SaveLibraryRecord(tx, userID, entity, id, fields, creating)
	if errors.Is(err, models.ErrEmptyUpdate) {
		return mutationErrorResponse(400, err.Error()), nil
	}

	if err != nil {
		return mutationResponse{}, err
	}

	status := 200
	if r.Method == http.MethodPost {
		status = 201
	}

	return recordResponse(tx, userID, entity, id, artistID, status)
}

func recordResponse(tx *sql.Tx, userID int, entity models.LibraryEntity, id, artistID, status int) (mutationResponse, error) {
	if entity == models.ArtistEntity {
		record, err := models.GetArtistByID(tx, userID, id)
		if err != nil {
			return mutationResponse{}, err
		}

		return mutationResponse{status: status, body: record, version: record.Version}, nil
	}

	record, err := models.GetAlbumByID(tx, userID, artistID, id)
	if err != nil {
		return mutationResponse{}, err
	}

	return mutationResponse{status: status, body: record, version: record.Version}, nil
}
