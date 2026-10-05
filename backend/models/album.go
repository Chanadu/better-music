package models

import "database/sql"

type Album struct {
	Version    int     `json:"version" validate:"required"`
	ID         int     `json:"id" validate:"required"`
	ArtistID   int     `json:"artist_id" validate:"required"`
	Title      string  `json:"title" validate:"required"`
	CoverUrl   *string `json:"cover_url" extensions:"x-nullable"`
	Year       *int    `json:"year" extensions:"x-nullable"`
	SpotifyID  *string `json:"spotify_id" extensions:"x-nullable"`
	Listened   bool    `json:"listened" validate:"required"`
	Rating     *int    `json:"rating" extensions:"x-nullable"`
	Comment    *string `json:"comment" extensions:"x-nullable"`
	ListenedAt *string `json:"listened_at" extensions:"x-nullable"`
	CreatedAt  string  `json:"created_at" validate:"required"`
}

func GetAlbumsByUser(database DB, userID int) ([]Album, error) {
	rows, err := database.Query(
		`SELECT id, artist_id, title, cover_url, year, spotify_id, listened, rating, comment, listened_at, created_at, version
		FROM albums
		WHERE user_id = $1
		ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}

	return scanAlbums(rows)
}

func AlbumExistsByID(database DB, userID int, artistID int, albumID int) (bool, error) {
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

func GetAlbumByID(database DB, userID int, artistID int, albumID int) (*Album, error) {
	return scanAlbum(database.QueryRow(
		`SELECT id, artist_id, title, cover_url, year, spotify_id, listened, rating, comment, listened_at, created_at, version
  FROM albums WHERE user_id=$1 AND artist_id=$2 AND id=$3`,
		userID, artistID, albumID,
	))
}

func scanAlbum(row interface{ Scan(...any) error }) (*Album, error) {
	var album Album
	err := row.Scan(&album.ID, &album.ArtistID, &album.Title, &album.CoverUrl,
		&album.Year, &album.SpotifyID, &album.Listened, &album.Rating,
		&album.Comment, &album.ListenedAt, &album.CreatedAt, &album.Version)
	if err != nil {
		return nil, err
	}
	return &album, nil
}

func scanAlbums(rows *sql.Rows) ([]Album, error) {
	defer rows.Close()
	albums := []Album{}
	for rows.Next() {
		album, err := scanAlbum(rows)
		if err != nil {
			return nil, err
		}
		albums = append(albums, *album)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return albums, nil
}
