package models

import (
	"github.com/Chanadu/better-music/db"
)

type Artist struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	SpotifyID string `json:"spotify_id"`
	CreatedAt string `json:"created_at"`
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

func ArtistExistsBySpotifyID(userID int, spotifyID string) (bool, error) {
	var exists bool
	err := db.DB.QueryRow(
		`SELECT EXISTS (
			SELECT 1
			FROM artists
			WHERE user_id = $1 AND spotify_id = $2
		)
		`,
		userID, spotifyID,
	).Scan(&exists)

	return exists, err
}

func CreateArtist(userID int, name string, spotifyID string) (*Artist, error) {
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
