package models

import "database/sql"

type Album struct {
	DocumentFields
	ArtistID   string                `json:"artist_id"`
	Title      string                `json:"title"`
	Year       NullableField[int32]  `json:"year" nullable:"true"`
	Listened   *bool                 `json:"listened" nullable:"false"`
	Rating     NullableField[int32]  `json:"rating" nullable:"true"`
	Comment    NullableField[string] `json:"comment" nullable:"true"`
	ListenedAt NullableField[string] `json:"listened_at" nullable:"true"`
}

func (album *Album) ParentArtistID() *string {
	return &album.ArtistID
}

func SaveAlbum(tx *sql.Tx, userID string, album Album, sequence int64) (sql.Result, error) {
	return tx.Exec(
		`INSERT INTO library_albums (
			id, user_id, artist_id, title, cover_url, spotify_id, year, listened,
			rating, comment, listened_at, created_at, edited_at, _deleted, change_seq
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			cover_url = EXCLUDED.cover_url,
			spotify_id = EXCLUDED.spotify_id,
			year = EXCLUDED.year,
			listened = EXCLUDED.listened,
			rating = EXCLUDED.rating,
			comment = EXCLUDED.comment,
			listened_at = EXCLUDED.listened_at,
			edited_at = EXCLUDED.edited_at,
			_deleted = EXCLUDED._deleted,
			change_seq = EXCLUDED.change_seq
		WHERE library_albums.user_id = EXCLUDED.user_id`,
		album.ID, userID, album.ArtistID, album.Title, album.CoverURL.Value, album.SpotifyID.Value,
		album.Year.Value, album.Listened, album.Rating.Value, album.Comment.Value, album.ListenedAt.Value,
		album.CreatedAt, album.EditedAt, album.Deleted, sequence,
	)
}

func LibraryArtistHasAlbums(tx *sql.Tx, userID string, id string) (bool, error) {
	var exists bool

	err := tx.QueryRow(
		`SELECT EXISTS (
			SELECT 1
			FROM library_albums
			WHERE user_id = $1 AND artist_id = $2 AND NOT _deleted
		)`,
		userID, id,
	).Scan(&exists)

	return exists, err
}
