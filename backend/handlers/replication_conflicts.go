package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/Chanadu/better-music/models"
)

func equivalentJSON(a, b []byte) bool {
	var av, bv any
	if json.Unmarshal(a, &av) != nil || json.Unmarshal(b, &bv) != nil {
		return false
	}

	return reflect.DeepEqual(av, bv)
}

func applySyncRow(tx *sql.Tx, userID int, kind models.LibraryKind, doc models.LibraryDocument, assumed []byte) (json.RawMessage, *LibraryChangeFailure, error) {
	fields := doc.Fields()
	current, currentKind, e := models.GetLibraryDocument(tx, userID, fields.ID)
	exists := e == nil
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return nil, nil, e
	}

	if exists {
		if currentKind != kind {
			return nil, &LibraryChangeFailure{ID: fields.ID, Error: "Record kind cannot change"}, nil
		}

		if equivalentJSON(current, fields.Raw) {
			return nil, nil, nil
		}

		old, e := validateDocument(current, kind)
		if e != nil {
			return nil, nil, e
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
		live, e := models.LiveLibraryArtistExists(tx, userID, doc.ParentArtistID())
		if e != nil {
			return nil, nil, e
		}

		if !live {
			return nil, &LibraryChangeFailure{ID: fields.ID, Retryable: true, Error: "The album's artist must sync or be restored first"}, nil
		}
	}

	if kind == models.ArtistKind && *fields.Deleted {
		children, e := models.LibraryArtistHasAlbums(tx, userID, fields.ID)
		if e != nil {
			return nil, nil, e
		}

		if children {
			return nil, &LibraryChangeFailure{ID: fields.ID, Retryable: true, Error: "Remove this artist's albums first"}, nil
		}
	}

	e = models.SaveLibraryDocument(tx, userID, doc)
	return nil, nil, e
}
