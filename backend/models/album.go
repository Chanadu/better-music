package models

import (
	"database/sql"
)

type Album struct {
	ID         int      `json:"id"`
	ArtistID   int      `json:"artist_id"`
	Title      string   `json:"title"`
	CoverUrl   *string  `json:"cover_url,omitempty"`
	Year       *int     `json:"year,omitempty"`
	SpotifyID  *string  `json:"spotify_id,omitempty"`
	Listened   bool     `json:"listened"`
	Rating     *float64 `json:"rating,omitempty"`
	Comment    *string  `json:"comment,omitempty"`
	ListenedAt *string  `json:"listened_at,omitempty"`
	CreatedAt  string   `json:"created_at"`
}

func GetAlbumsByUser(database *sql.DB, userID int) ([]Album, error) {
	rows, err := database.Query(
		`SELECT id, artist_id, title, cover_url, year, spotify_id, listened, rating, comment, listened_at, created_at
		FROM albums
		WHERE user_id = $1
		ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	albums := []Album{}

	for rows.Next() {
		var album Album
		err := rows.Scan(
			&album.ID,
			&album.ArtistID,
			&album.Title,
			&album.CoverUrl,
			&album.Year,
			&album.SpotifyID,
			&album.Listened,
			&album.Rating,
			&album.Comment,
			&album.ListenedAt,
			&album.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		albums = append(albums, album)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return albums, nil
}

func AlbumExistsByName(database *sql.DB, userID int, artistID int, title string) (bool, error) {
	var exists bool
	err := database.QueryRow(
		`SELECT EXISTS (
			SELECT 1
			FROM albums
			WHERE user_id = $1 AND artist_id = $2 AND title = $3
		)
		`,
		userID, artistID, title,
	).Scan(&exists)

	return exists, err
}

func AlbumExistsByID(database *sql.DB, userID int, artistID int, albumID int) (bool, error) {
	var exists bool
	err := database.QueryRow(
		`SELECT EXISTS (
			SELECT 1
			FROM albums
			WHERE user_id = $1 AND artist_id = $2 AND id = $3
		)
		`,
		userID, artistID, albumID,
	).Scan(&exists)

	return exists, err
}

func CreateAlbum(database *sql.DB, userID int, artistID int, title string, spotifyID *string) (*Album, error) {
	var album Album

	err := database.QueryRow(
		`INSERT INTO albums (user_id, artist_id, title, spotify_id)
	         VALUES ($1, $2, $3, $4)
		 RETURNING id, artist_id, title, spotify_id, listened, created_at`,
		userID, artistID, title, spotifyID,
	).Scan(&album.ID, &album.ArtistID, &album.Title, &album.SpotifyID, &album.Listened, &album.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &album, nil
}

func GetAlbumByID(database *sql.DB, userID int, artistID int, albumID int) (*Album, error) {
	var album Album

	err := database.QueryRow(
		`SELECT id, artist_id, title, cover_url, year, spotify_id, listened, rating, comment, listened_at, created_at
		FROM albums
		WHERE user_id = $1 AND artist_id = $2 AND id = $3`,
		userID, artistID, albumID,
	).Scan(
		&album.ID,
		&album.ArtistID,
		&album.Title,
		&album.CoverUrl,
		&album.Year,
		&album.SpotifyID,
		&album.Listened,
		&album.Rating,
		&album.Comment,
		&album.ListenedAt,
		&album.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &album, nil
}

func UpdateAlbum(database *sql.DB, userID int, artistID int, albumID int, title *string, coverURL *string, year *int, spotifyID *string, listened *bool, rating *float64, comment *string, listenedAt *string) error {
	result, err := database.Exec(
		`UPDATE albums
		SET title = COALESCE($4, title),
			cover_url = COALESCE($5, cover_url),
			year = COALESCE($6, year),
			spotify_id = COALESCE($7, spotify_id),
			listened = COALESCE($8, listened),
			rating = COALESCE($9, rating),
			comment = COALESCE($10, comment),
			listened_at = COALESCE($11, listened_at)
		WHERE user_id = $1 AND artist_id = $2 AND id = $3`,
		userID, artistID, albumID, title, coverURL, year, spotifyID, listened, rating, comment, listenedAt,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func DeleteAlbum(database *sql.DB, userID int, artistID int, albumID int) error {
	result, err := database.Exec(
		`DELETE FROM albums
		WHERE user_id = $1 AND artist_id = $2 AND id = $3`,
		userID, artistID, albumID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}
