package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

func libraryQuery(kind LibraryKind) (table, document string, err error) {
	common := `'id',id,'cover_url',cover_url,'spotify_id',spotify_id,
 'created_at',created_at,'_deleted',_deleted,'edited_at',
 to_char(edited_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')`

	switch kind {
	case ArtistKind:
		return "library_artists", "jsonb_build_object(" + common + ",'name',name)", nil

	case AlbumKind:
		return "library_albums", "jsonb_build_object(" + common + ",'artist_id',artist_id,'title',title,'year',year,'listened',listened,'rating',rating,'comment',comment,'listened_at',listened_at)", nil

	default:
		return "", "", errors.New("unknown collection")
	}
}

func PullLibraryDocuments(tx *sql.Tx, userID int, kind LibraryKind, sequence int64, limit int) ([]json.RawMessage, int64, error) {
	table, document, err := libraryQuery(kind)
	if err != nil {
		return nil, sequence, err
	}

	rows, err := tx.Query(
		fmt.Sprintf(`SELECT %s, change_seq FROM %s record
	 WHERE user_id=$1 AND change_seq>$2 ORDER BY change_seq LIMIT $3`, document, table),
		userID, sequence, limit,
	)
	if err != nil {
		return nil, sequence, err
	}

	defer rows.Close()

	documents := []json.RawMessage{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw, &sequence); err != nil {
			return nil, sequence, err
		}

		documents = append(documents, json.RawMessage(raw))
	}

	return documents, sequence, rows.Err()
}

func GetLibraryDocument(tx *sql.Tx, userID int, id string) ([]byte, LibraryKind, error) {
	for _, kind := range []LibraryKind{ArtistKind, AlbumKind} {
		table, document, _ := libraryQuery(kind)

		var data []byte

		err := tx.QueryRow(fmt.Sprintf(`SELECT %s FROM %s record WHERE user_id=$1 AND id=$2 FOR UPDATE`, document, table), userID, id).Scan(&data)

		if errors.Is(err, sql.ErrNoRows) {
			continue
		}

		return data, kind, err
	}
	return nil, UnknownLibraryKind, sql.ErrNoRows
}

func SaveLibraryDocument(tx *sql.Tx, userID int, document LibraryDocument) error {
	var sequence int64

	if err := tx.QueryRow(
		`UPDATE users SET library_sync_sequence=library_sync_sequence+1 WHERE id=$1 RETURNING library_sync_sequence`, userID,
	).Scan(&sequence); err != nil {
		return err
	}

	var result sql.Result
	var err error

	switch doc := document.(type) {
	case *Artist:
		result, err = SaveArtist(tx, userID, *doc, sequence)

	case *Album:
		result, err = SaveAlbum(tx, userID, *doc, sequence)

	default:
		return errors.New("unknown document type")
	}

	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return errors.New("record belongs to another account or has a different kind")
	}

	return nil
}
