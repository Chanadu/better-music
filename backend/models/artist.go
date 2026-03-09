package models

import (
	"database/sql"

	"github.com/Chanadu/better-music/db"
)

type Artist struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	SpotifyID *string `json:"spotify_id,omitempty"`
	CreatedAt string  `json:"created_at"`
}

func GetArtistsByUser(userID int) ([]Artist, error) {
	rows, err := db.DB.Query(
		`SELECT id, name, spotify_id, created_at 
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
	var artists []Artist

	for rows.Next() {
		var artist Artist
		err := rows.Scan(&artist.ID, &artist.Name, &artist.SpotifyID, &artist.CreatedAt)
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

func ArtistExistsByName(userID int, name string) (bool, error) {
	var exists bool
	err := db.DB.QueryRow(
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

func ArtistExistsByID(userID int, id int) (bool, error) {
	var exists bool
	err := db.DB.QueryRow(
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

func CreateArtist(userID int, name string, spotifyID *string) (*Artist, error) {
	var artist Artist

	err := db.DB.QueryRow(
		`INSERT INTO artists (user_id, name, spotify_id)
         VALUES ($1, $2, $3)
         RETURNING id, created_at`,
		userID, name, spotifyID,
	).Scan(&artist.ID, &artist.CreatedAt)

	if err != nil {
		return nil, err
	}

	artist.Name = name
	artist.SpotifyID = spotifyID

	return &artist, nil
}

func DeleteArtist(userID int, artistID int) error {
	result, err := db.DB.Exec(
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

func UpdateArtist(userID int, artistID int, name *string, spotifyID *string) error {
	result, err := db.DB.Exec(
		`UPDATE artists
		SET name = COALESCE($3, name), spotify_id = COALESCE($4, spotify_id)
		WHERE user_id = $1 AND id = $2
		`,
		userID, artistID, name, spotifyID,
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
