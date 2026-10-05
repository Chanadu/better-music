package models

type Artist struct {
	Version   int     `json:"version" validate:"required"`
	ID        int     `json:"id" validate:"required"`
	Name      string  `json:"name" validate:"required"`
	CoverURL  *string `json:"cover_url" extensions:"x-nullable"`
	SpotifyID *string `json:"spotify_id" extensions:"x-nullable"`
	CreatedAt string  `json:"created_at" validate:"required"`
}

func GetArtistsByUser(database DB, userID int) ([]Artist, error) {
	rows, err := database.Query(
		`SELECT id, name, cover_url, spotify_id, created_at, version
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
		err := rows.Scan(&artist.ID, &artist.Name, &artist.CoverURL, &artist.SpotifyID, &artist.CreatedAt, &artist.Version)
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

func GetArtistByID(database DB, userID int, artistID int) (*Artist, error) {
	var artist Artist

	err := database.QueryRow(
		`SELECT id, name, cover_url, spotify_id, created_at, version
		FROM artists
		WHERE user_id = $1 AND id = $2`,
		userID, artistID,
	).Scan(&artist.ID, &artist.Name, &artist.CoverURL, &artist.SpotifyID, &artist.CreatedAt, &artist.Version)
	if err != nil {
		return nil, err
	}

	return &artist, nil
}

func ArtistExistsByID(database DB, userID int, id int) (bool, error) {
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

func GetArtistAlbums(database DB, userID int, artistID int) ([]Album, error) {
	rows, err := database.Query(
		`SELECT id, artist_id, title, cover_url, year, spotify_id, listened, rating, comment, listened_at, created_at, version
		FROM albums 
		WHERE user_id = $1 AND artist_id = $2
		ORDER BY created_at DESC
		`,
		userID, artistID,
	)
	if err != nil {
		return nil, err
	}

	return scanAlbums(rows)
}
