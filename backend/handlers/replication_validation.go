package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Chanadu/better-music/models"
)

func validUUID(value string) bool {
	if len(value) != 36 {
		return false
	}

	for i := 0; i < len(value); i++ {
		switch i {
		case 8, 13, 18, 23:
			if value[i] != '-' {
				return false
			}
		default:
			if !strings.ContainsRune("0123456789abcdef", rune(value[i])) {
				return false
			}
		}
	}

	return true
}

func validateDocument(raw []byte, kind models.LibraryKind) (models.LibraryDocument, error) {
	switch kind {
	case models.ArtistKind:
		return validateArtistDocument(raw)
	case models.AlbumKind:
		return validateAlbumDocument(raw)
	default:
		return nil, errors.New("unknown collection")
	}
}

func decodeDocument(raw []byte, out any) error {
	if raw = bytes.TrimSpace(raw); len(raw) == 0 || raw[0] != '{' {
		return errors.New("document must be an object")
	}

	return decodeJSON(bytes.NewReader(raw), out)
}

func decodeJSON(reader io.Reader, out any) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("expected one JSON value")
	}

	return nil
}

func validatePushRows(rows []LibraryChange, kind models.LibraryKind) ([]models.LibraryDocument, error) {
	docs := make([]models.LibraryDocument, len(rows))
	seen := map[string]bool{}
	for i, row := range rows {
		doc, err := validateDocument(row.New, kind)
		if err != nil {
			return nil, err
		}

		if seen[doc.Fields().ID] {
			return nil, errors.New("duplicate document in batch")
		}

		if len(row.Assumed) > 0 && !bytes.Equal(row.Assumed, []byte("null")) {
			old, err := validateDocument(row.Assumed, kind)
			if err != nil || old.Fields().ID != doc.Fields().ID {
				return nil, errors.New("invalid assumed state")
			}
		}

		seen[doc.Fields().ID] = true
		docs[i] = doc
	}

	return docs, nil
}

func validateArtistDocument(raw []byte) (*models.Artist, error) {
	var artist models.Artist
	if err := decodeDocument(raw, &artist); err != nil {
		return nil, err
	}

	if err := validateDocumentFields(raw, &artist.DocumentFields, "name", artist.Name); err != nil {
		return nil, err
	}

	return &artist, nil
}

func validateAlbumDocument(raw []byte) (*models.Album, error) {
	var album models.Album
	if err := decodeDocument(raw, &album); err != nil {
		return nil, err
	}

	if err := validateDocumentFields(raw, &album.DocumentFields, "title", album.Title); err != nil {
		return nil, err
	}

	if !validUUID(album.ArtistID) {
		return nil, errors.New("invalid artist UUID")
	}

	if album.Listened == nil {
		return nil, errors.New("listened must be boolean")
	}

	if !album.Year.Present || !album.Rating.Present || !album.Comment.Present || !album.ListenedAt.Present {
		return nil, errors.New("year, rating, comment, and listened_at must be supplied, using null for empty values")
	}

	if rating := album.Rating.Value; rating != nil && (*rating < 1 || *rating > 10) {
		return nil, errors.New("rating must be between 1 and 10")
	}

	if date := album.ListenedAt.Value; date != nil && *date != "" && !validSyncDate(*date) {
		return nil, errors.New("invalid listened_at")
	}

	if !*album.Listened && (album.Rating.Value != nil || album.ListenedAt.Value != nil) {
		return nil, errors.New("unlistened albums cannot have a rating or listening date")
	}

	return &album, nil
}

func validateDocumentFields(raw []byte, fields *models.DocumentFields, name, value string) error {
	if !validUUID(fields.ID) {
		return errors.New("invalid UUID")
	}

	if strings.TrimSpace(value) == "" || len(value) > 4096 {
		return fmt.Errorf("%s is required and must be at most 4096 bytes", name)
	}

	if !validSyncDate(fields.CreatedAt) {
		return errors.New("invalid created_at")
	}

	stamp, err := time.Parse("2006-01-02T15:04:05.000Z", fields.EditedAt)
	if err != nil {
		return errors.New("edited_at must be a UTC timestamp with millisecond precision")
	}

	if fields.Deleted == nil {
		return errors.New("invalid deletion flag")
	}

	if !fields.CoverURL.Present || !fields.SpotifyID.Present {
		return errors.New("cover_url and spotify_id must be supplied, using null for empty values")
	}

	fields.Edited = stamp
	fields.Raw = append([]byte(nil), raw...)
	return nil
}

func validSyncDate(value string) bool {
	for _, format := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999", "2006-01-02"} {
		if _, err := time.Parse(format, value); err == nil {
			return true
		}
	}

	return false
}
