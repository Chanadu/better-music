package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Chanadu/better-music/models"
)

func parseIfMatch(r *http.Request) (int, error) {
	value := r.Header.Get("If-Match")
	if value == "" {
		return 0, nil
	}

	if len(r.Header.Values("If-Match")) != 1 {
		return 0, errors.New("invalid If-Match")
	}

	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		value = value[1 : len(value)-1]
	}

	version, err := strconv.Atoi(value)
	if err != nil || version <= 0 || version > 2147483647 {
		return 0, errors.New("If-Match must contain a positive integer version")
	}

	return version, nil
}

func decodeMutationFields(raw []byte, entity models.LibraryEntity) (map[string]any, error) {
	var input map[string]json.RawMessage
	if err := json.Unmarshal(raw, &input); err != nil || input == nil {
		return nil, errors.New("body must be a JSON object")
	}

	allowed := map[string]string{"name": "string", "cover_url": "string", "spotify_id": "string"}
	if entity == models.AlbumEntity {
		allowed = map[string]string{"artist_id": "int", "title": "string", "cover_url": "string", "year": "int", "spotify_id": "string", "listened": "bool", "rating": "int", "comment": "string", "listened_at": "string"}
	}

	fields := map[string]any{}
	for field, rawValue := range input {
		kind, ok := allowed[field]
		if !ok {
			return nil, fmt.Errorf("unknown field: %s", field)
		}

		if string(rawValue) == "null" {
			if field == "name" || field == "title" || field == "artist_id" || field == "listened" {
				return nil, fmt.Errorf("%s cannot be null", field)
			}

			fields[field] = nil
			continue
		}

		switch kind {
		case "string":
			var value string
			if json.Unmarshal(rawValue, &value) != nil {
				return nil, fmt.Errorf("%s must be a string", field)
			}

			if field == "name" || field == "title" {
				value = strings.TrimSpace(value)
				if value == "" {
					return nil, fmt.Errorf("%s is required", field)
				}
			}

			fields[field] = value
		case "int":
			var value int
			if json.Unmarshal(rawValue, &value) != nil || value < -2147483648 || value > 2147483647 {
				return nil, fmt.Errorf("%s must be an integer", field)
			}

			if field == "artist_id" && value <= 0 {
				return nil, errors.New("invalid artist ID")
			}

			if field == "rating" && (value < 1 || value > 10) {
				return nil, errors.New("rating must be between 1 and 10")
			}

			fields[field] = value
		case "bool":
			var value bool
			if json.Unmarshal(rawValue, &value) != nil {
				return nil, fmt.Errorf("%s must be a boolean", field)
			}

			fields[field] = value
		}
	}
	return fields, nil
}
