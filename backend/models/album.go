package models

import (
	"database/sql"

	"github.com/Chanadu/better-music/db"
)

type Album struct {
	ID         int     `json:"id"`
	ArtistID   int     `json:"artist_id"`
	Title      string  `json:"title"`
	CoverUrl   *string `json:"cover_url,omitempty"`
	Year       *int    `json:"year,omitempty"`
	SpotifyID  *string `json:"spotify_id,omitempty"`
	Listened   bool    `json:"listened"`
	Rating     *int    `json:"rating,omitempty"`
	Comment    *string `json:"comment,omitempty"`
	ListenedAt *string `json:"listened_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

func GetAlbumsByUser(userID int) ([]Album, error) {
	rows, err := db.DB.Query(
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

func AlbumExistsByName(userID int, artistID int, title string) (bool, error) {
	var exists bool
	err := db.DB.QueryRow(
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

func AlbumExistsByID(userID int, artistID int, albumID int) (bool, error) {
	var exists bool
	err := db.DB.QueryRow(
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

func CreateAlbum(userID int, artistID int, title string) (*Album, error) {
	var album Album

	err := db.DB.QueryRow(
		`INSERT INTO albums (user_id, artist_id, title)
         VALUES ($1, $2, $3)
		 RETURNING id, artist_id, title, listened, created_at`,
		userID, artistID, title,
	).Scan(&album.ID, &album.ArtistID, &album.Title, &album.Listened, &album.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &album, nil
}

func GetAlbumByID(userID int, artistID int, albumID int) (*Album, error) {
	var album Album

	err := db.DB.QueryRow(
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

func UpdateAlbum(userID int, artistID int, albumID int, title *string, coverURL *string, year *int, spotifyID *string, listened *bool, rating *int, comment *string, listenedAt *string) error {
	result, err := db.DB.Exec(
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

func DeleteAlbum(userID int, artistID int, albumID int) error {
	result, err := db.DB.Exec(
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
