package models

import "github.com/Chanadu/better-music/db"

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

func CreateAlbum(userID int, artistID int, title string) (*Album, error) {
	var album Album

	err := db.DB.QueryRow(
		`INSERT INTO albums (user_id, artist_id, title)
         VALUES ($1, $2, $3)
         RETURNING id, listened, created_at`,
		userID, artistID, title,
	).Scan(&album.ID, &album.Listened, &album.CreatedAt)

	if err != nil {
		return nil, err
	}

	album.Title = title

	return &album, nil
}
