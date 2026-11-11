package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/Chanadu/better-music/models"
)

func equivalentJSON(left, right []byte) bool {
	var av, bv any
	if json.Unmarshal(left, &av) != nil || json.Unmarshal(right, &bv) != nil {
		return false
	}

	return reflect.DeepEqual(av, bv)
}

func applySyncRow(tx *sql.Tx, userID string, kind models.LibraryKind, doc models.LibraryDocument, assumed []byte) (json.RawMessage, *LibraryChangeFailure, error) {
	fields := doc.Fields()
	current, currentKind, err := models.GetLibraryDocument(tx, userID, fields.ID)
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}

	if exists {
		if currentKind != kind {
			return nil, &LibraryChangeFailure{ID: fields.ID, Error: "Record kind cannot change"}, nil
		}

		if equivalentJSON(current, fields.Raw) {
			return nil, nil, nil
		}

		old, err := validateDocument(current, kind)
		if err != nil {
			return nil, nil, err
		}

		if old.Fields().CreatedAt != fields.CreatedAt || !reflect.DeepEqual(old.ParentArtistID(), doc.ParentArtistID()) {
			return nil, &LibraryChangeFailure{ID: fields.ID, Error: "Creation time and album artist cannot change"}, nil
		}

		if !equivalentJSON(current, assumed) || !fields.Edited.After(old.Fields().Edited) {
			return json.RawMessage(current), nil, nil
		}
	} else if len(assumed) > 0 && !bytes.Equal(assumed, []byte("null")) {
		return nil, &LibraryChangeFailure{ID: fields.ID, Error: "The previously synchronized record is missing; preserve edits before resetting"}, nil
	}

	if kind == models.AlbumKind && !*fields.Deleted {
		live, err := models.LiveLibraryArtistExists(tx, userID, doc.ParentArtistID())
		if err != nil {
			return nil, nil, err
		}

		if !live {
			return nil, &LibraryChangeFailure{ID: fields.ID, Retryable: true, Error: "The album's artist must sync or be restored first"}, nil
		}
	}

	if kind == models.ArtistKind && *fields.Deleted {
		children, err := models.LibraryArtistHasAlbums(tx, userID, fields.ID)
		if err != nil {
			return nil, nil, err
		}

		if children {
			return nil, &LibraryChangeFailure{ID: fields.ID, Retryable: true, Error: "Remove this artist's albums first"}, nil
		}
	}

	err = models.SaveLibraryDocument(tx, userID, doc)
	return nil, nil, err
}
