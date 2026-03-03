package models

import "github.com/Chanadu/better-music/db"

type Artist struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	SpotifyID string `json:"spotify_id"`
	CreatedAt string `json:"created_at"`
}

func GetArtistsByUser(userID int) ([]Artist, error) {
	rows, err := db.DB.Query(
		`SELECT id, user_id, name, spotify_id, created_at 
		FROM artists 
		WHERE user_id = $1 
		ORDER BY created_at DESC
		`,
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
