package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrEmptyUpdate = errors.New("at least one field must be provided")

func libraryTable(entity LibraryEntity) (string, error) {
	switch entity {
	case ArtistEntity, AlbumEntity:
		return string(entity), nil
	default:
		return "", errors.New("unknown library entity")
	}
}

func LockLibraryRecord(tx *sql.Tx, userID int, entity LibraryEntity, id int) (int, int, error) {
	var version, artistID int

	_, err := libraryTable(entity)
	if err != nil {
		return 0, 0, err
	}

	if entity == AlbumEntity {
		err = tx.QueryRow(`SELECT version, artist_id FROM albums WHERE user_id=$1 AND id=$2 FOR UPDATE`, userID, id).Scan(&version, &artistID)
	} else {
		err = tx.QueryRow(`SELECT version FROM artists WHERE user_id=$1 AND id=$2 FOR UPDATE`, userID, id).Scan(&version)
	}

	return version, artistID, err
}

func DeleteLibraryRecord(tx *sql.Tx, userID int, entity LibraryEntity, id int) error {
	table, err := libraryTable(entity)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM `+table+` WHERE user_id=$1 AND id=$2`, userID, id)
	return err
}

func SaveLibraryRecord(tx *sql.Tx, userID int, entity LibraryEntity, id int, fields map[string]any, creating bool) (int, error) {
	table, err := libraryTable(entity)
	if err != nil {
		return 0, err
	}

	columns := []string{"name", "cover_url", "spotify_id"}
	if entity == AlbumEntity {
		columns = []string{"artist_id", "title", "cover_url", "year", "spotify_id", "listened", "rating", "comment", "listened_at"}
	}

	args := []any{userID}
	names, placeholders := []string{}, []string{}

	if creating {
		names, placeholders = append(names, "user_id"), append(placeholders, "$1")
	} else {
		args = append(args, id)
	}

	for _, column := range columns {
		value, present := fields[column]
		if !present || (!creating && column == "artist_id") {
			continue
		}

		args = append(args, value)
		names = append(names, column)
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}

	if creating {
		err = tx.QueryRow(`INSERT INTO `+table+` (`+strings.Join(names, ",")+`) VALUES (`+strings.Join(placeholders, ",")+`) RETURNING id`, args...).Scan(&id)
	} else {
		if len(names) == 0 {
			return 0, ErrEmptyUpdate
		}

		sets := []string{"version=version+1"}
		for index, name := range names {
			sets = append(sets, name+"="+placeholders[index])
		}

		_, err = tx.Exec(`UPDATE `+table+` SET `+strings.Join(sets, ",")+` WHERE user_id=$1 AND id=$2`, args...)
	}
	return id, err
}
