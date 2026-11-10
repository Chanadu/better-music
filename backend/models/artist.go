package models

import "database/sql"

type Artist struct {
	DocumentFields
	Name string `json:"name"`
}

func SaveArtist(tx *sql.Tx, userID int, artist Artist, sequence int64) (sql.Result, error) {
	return tx.Exec(
		`INSERT INTO library_artists(id,user_id,name,cover_url,spotify_id,created_at,edited_at,_deleted,change_seq)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT(id) DO UPDATE SET name=EXCLUDED.name,cover_url=EXCLUDED.cover_url,
		spotify_id=EXCLUDED.spotify_id,edited_at=EXCLUDED.edited_at,
		_deleted=EXCLUDED._deleted,change_seq=EXCLUDED.change_seq
		WHERE library_artists.user_id=EXCLUDED.user_id`,
		artist.ID, userID, artist.Name, artist.CoverURL.Value, artist.SpotifyID.Value, artist.CreatedAt,
		artist.EditedAt, artist.Deleted, sequence,
	)
}

func LiveLibraryArtistExists(tx *sql.Tx, userID int, id *string) (bool, error) {
	var exists bool
	err := tx.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM library_artists
		WHERE user_id=$1 AND id=$2 AND NOT _deleted)`,
		userID, id,
	).Scan(&exists)
	return exists, err
}
