package models

type LibraryKind uint8

const (
	UnknownLibraryKind LibraryKind = iota
	ArtistKind
	AlbumKind
)
