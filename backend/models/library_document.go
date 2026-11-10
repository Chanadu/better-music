package models

import (
	"encoding/json"
	"time"
)

type LibraryDocument interface {
	Fields() *DocumentFields
	ParentArtistID() *string
}

type NullableField[T any] struct {
	Present bool
	Value   *T
}

func (field *NullableField[T]) UnmarshalJSON(raw []byte) error {
	field.Present = true
	return json.Unmarshal(raw, &field.Value)
}

func (field NullableField[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(field.Value)
}

type DocumentFields struct {
	ID        string                `json:"id"`
	CoverURL  NullableField[string] `json:"cover_url"`
	SpotifyID NullableField[string] `json:"spotify_id"`
	CreatedAt string                `json:"created_at"`
	EditedAt  string                `json:"edited_at"`
	Deleted   *bool                 `json:"_deleted"`
	Raw       []byte                `json:"-"`
	Edited    time.Time             `json:"-"`
}

func (fields *DocumentFields) Fields() *DocumentFields {
	return fields
}

func (fields *DocumentFields) ParentArtistID() *string {
	return nil
}
