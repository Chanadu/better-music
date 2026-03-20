package models

import (
	"database/sql"
)

type Artist struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	CoverURL  *string `json:"cover_url,omitempty"`
	SpotifyID *string `json:"spotify_id,omitempty"`
	CreatedAt string  `json:"created_at"`
}

func GetArtistsByUser(database *sql.DB, userID int) ([]Artist, error) {
	rows, err := database.Query(
		`SELECT id, name, cover_url, spotify_id, created_at 
		FROM artists 
		WHERE user_id = $1 
		ORDER BY created_at DESC
		`,
		userID,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	artists := []Artist{}

	for rows.Next() {
		var artist Artist
		err := rows.Scan(&artist.ID, &artist.Name, &artist.CoverURL, &artist.SpotifyID, &artist.CreatedAt)
		if err != nil {
			return nil, err
		}
		artists = append(artists, artist)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return artists, nil
}

func GetArtistByID(database *sql.DB, userID int, artistID int) (*Artist, error) {
	var artist Artist

	err := database.QueryRow(
		`SELECT id, name, cover_url, spotify_id, created_at
		FROM artists
		WHERE user_id = $1 AND id = $2`,
		userID, artistID,
	).Scan(&artist.ID, &artist.Name, &artist.CoverURL, &artist.SpotifyID, &artist.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &artist, nil
}

func ArtistExistsByName(database *sql.DB, userID int, name string) (bool, error) {
	var exists bool
	err := database.QueryRow(
		`SELECT EXISTS (
			SELECT 1
			FROM artists
			WHERE user_id = $1 AND name = $2
		)
		`,
		userID, name,
	).Scan(&exists)

	return exists, err
}

func ArtistExistsByID(database *sql.DB, userID int, id int) (bool, error) {
	var exists bool
	err := database.QueryRow(
		`SELECT EXISTS (
			SELECT 1
			FROM artists
			WHERE user_id = $1 AND id = $2
		)
		`,
		userID, id,
	).Scan(&exists)

	return exists, err
}

func CreateArtist(database *sql.DB, userID int, name string, coverURL *string, spotifyID *string) (*Artist, error) {
	var artist Artist

	err := database.QueryRow(
		`INSERT INTO artists (user_id, name, cover_url, spotify_id)
	         VALUES ($1, $2, $3, $4)
		 RETURNING id, name, cover_url, spotify_id, created_at`,
		userID, name, coverURL, spotifyID,
	).Scan(&artist.ID, &artist.Name, &artist.CoverURL, &artist.SpotifyID, &artist.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &artist, nil
}

func DeleteArtist(database *sql.DB, userID int, artistID int) error {
	result, err := database.Exec(
		`DELETE FROM artists
		WHERE user_id = $1 AND id = $2
		`,
		userID, artistID,
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

func UpdateArtist(database *sql.DB, userID int, artistID int, name *string, coverURL *string, spotifyID *string) error {
	result, err := database.Exec(
		`UPDATE artists
		SET name = COALESCE($3, name), cover_url = COALESCE($4, cover_url), spotify_id = COALESCE($5, spotify_id)
		WHERE user_id = $1 AND id = $2
		`,
		userID, artistID, name, coverURL, spotifyID,
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

func GetArtistAlbums(database *sql.DB, userID int, artistID int) ([]Album, error) {
	rows, err := database.Query(
		`SELECT id, artist_id, title, cover_url, year, spotify_id, listened, rating, comment, listened_at, created_at
		FROM albums 
		WHERE user_id = $1 AND artist_id = $2
		ORDER BY created_at DESC
		`,
		userID, artistID,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	albums := []Album{}

	for rows.Next() {
		var album Album
		err := rows.Scan(&album.ID, &album.ArtistID, &album.Title, &album.CoverUrl, &album.Year, &album.SpotifyID, &album.Listened, &album.Rating, &album.Comment, &album.ListenedAt, &album.CreatedAt)
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
