package models

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/Chanadu/better-music/internal/testdb"
)

func must(test *testing.T, err error) {
	test.Helper()
	if err != nil {
		test.Fatal(err)
	}
}

func TestUsersAndRefreshTokens(test *testing.T) {
	database := testdb.Open(test)
	expiry := time.Now().Add(time.Hour)
	user, err := CreateUserWithRefreshToken(database, "user@example.com", "hash", "original", expiry)
	must(test, err)

	for _, lookup := range []func() (*User, error){
		func() (*User, error) {
			return GetUserByID(database, user.ID)
		},
		func() (*User, error) {
			return GetUserByEmail(database, user.Email)
		},
	} {
		got, err := lookup()
		must(test, err)
		if got.ID != user.ID || got.PasswordHash != "hash" || got.Email != user.Email || got.CreatedAt.IsZero() {
			test.Fatalf("%+v", got)
		}
	}

	if _, err := GetUserByID(database, -1); !errors.Is(err, sql.ErrNoRows) {
		test.Fatal(err)
	}

	if _, err := CreateUserWithRefreshToken(database, "rollback@example.com", "hash", "original", expiry); err == nil {
		test.Fatal("duplicate token accepted")
	}

	if _, err := GetUserByEmail(database, "rollback@example.com"); !errors.Is(err, sql.ErrNoRows) {
		test.Fatalf("user leaked after rollback: %v", err)
	}

	must(test, CreateRefreshToken(database, user.ID, "active", expiry))

	token, err := GetRefreshTokenByHash(database, "active")
	must(test, err)
	if token.UserID != user.ID || token.RevokedAt.Valid {
		test.Fatal(token)
	}

	if _, err := RotateRefreshToken(database, "active", "original", expiry); err == nil {
		test.Fatal("duplicate replacement accepted")
	}

	token, err = GetRefreshTokenByHash(database, "active")
	must(test, err)
	if token.RevokedAt.Valid {
		test.Fatal("failed rotation consumed original")
	}

	rotatedUserID, err := RotateRefreshToken(database, "active", "replacement", expiry)
	must(test, err)
	if rotatedUserID != user.ID {
		test.Fatal(rotatedUserID)
	}

	if _, err := RotateRefreshToken(database, "active", "another", expiry); !errors.Is(err, sql.ErrNoRows) {
		test.Fatal(err)
	}

	replacement, err := GetRefreshTokenByHash(database, "replacement")
	must(test, err)
	must(test, RevokeRefreshToken(database, replacement.ID))
	must(test, RevokeRefreshToken(database, replacement.ID))
	if _, err := RotateRefreshToken(database, "replacement", "another", expiry); !errors.Is(err, sql.ErrNoRows) {
		test.Fatal(err)
	}

	must(test, CreateRefreshToken(database, user.ID, "expired", time.Now().Add(-time.Hour)))
	if _, err := RotateRefreshToken(database, "expired", "another", expiry); !errors.Is(err, sql.ErrNoRows) {
		test.Fatal(err)
	}

	count, err := CleanupRefreshTokens(database)
	must(test, err)
	if count != 1 {
		test.Fatalf("cleanup deleted %d", count)
	}

	if err := UpdateUserEmail(database, user.ID, "new@example.com", "stale"); !errors.Is(err, sql.ErrNoRows) {
		test.Fatal(err)
	}

	must(test, UpdateUserEmail(database, user.ID, "new@example.com", "hash"))

	token, err = GetRefreshTokenByHash(database, "original")
	must(test, err)
	if !token.RevokedAt.Valid {
		test.Error("email update did not revoke sessions")
	}

	must(test, CreateRefreshToken(database, user.ID, "password-session", expiry))
	if err := UpdateUserPassword(database, user.ID, "new-hash", "stale"); !errors.Is(err, sql.ErrNoRows) {
		test.Fatal(err)
	}

	must(test, UpdateUserPassword(database, user.ID, "new-hash", "hash"))

	token, err = GetRefreshTokenByHash(database, "password-session")
	must(test, err)
	if !token.RevokedAt.Valid {
		test.Error("password update did not revoke sessions")
	}

	if err := DeleteUser(database, user.ID, "stale"); !errors.Is(err, sql.ErrNoRows) {
		test.Fatal(err)
	}

	must(test, DeleteUser(database, user.ID, "new-hash"))
	if _, err := GetUserByID(database, user.ID); !errors.Is(err, sql.ErrNoRows) {
		test.Fatal(err)
	}

	if _, err := GetRefreshTokenByHash(database, "original"); !errors.Is(err, sql.ErrNoRows) {
		test.Fatal("token did not cascade", err)
	}
}

func TestLibraryStorageAndSavepoints(test *testing.T) {
	database := testdb.Open(test)
	user, err := CreateUserWithRefreshToken(database, "owner@example.com", "hash", "token", time.Now().Add(time.Hour))
	must(test, err)

	transaction, err := database.Begin()
	must(test, err)
	defer transaction.Rollback()
	must(test, LockUserForMutation(transaction, user.ID))
	must(test, CreateLibrarySavepoint(transaction))

	artistID, err := SaveLibraryRecord(transaction, user.ID, ArtistEntity, 0, map[string]any{"name": "Artist"}, true)
	must(test, err)

	version, artist, err := LockLibraryRecord(transaction, user.ID, ArtistEntity, artistID)
	must(test, err)
	if version != 1 || artist != 0 {
		test.Fatal(version, artist)
	}

	albumID, err := SaveLibraryRecord(transaction, user.ID, AlbumEntity, 0, map[string]any{
		"artist_id":   artistID,
		"title":       "Album",
		"rating":      7,
		"year":        2020,
		"listened":    true,
		"listened_at": "2025-01-01",
	}, true)
	must(test, err)

	version, artist, err = LockLibraryRecord(transaction, user.ID, AlbumEntity, albumID)
	must(test, err)
	if version != 1 || artist != artistID {
		test.Fatal(version, artist)
	}

	if _, err := SaveLibraryRecord(transaction, user.ID, ArtistEntity, artistID, map[string]any{}, false); !errors.Is(err, ErrEmptyUpdate) {
		test.Fatal(err)
	}

	_, err = SaveLibraryRecord(transaction, user.ID, ArtistEntity, artistID, map[string]any{"name": "Updated", "cover_url": "image"}, false)
	must(test, err)

	saved := ProcessedMutation{
		RequestHash: "hash",
		Status:      201,
		Headers:     map[string]string{"ETag": `"2"`},
		Body:        []byte("response\n"),
	}
	must(test, SaveProcessedMutation(transaction, user.ID, "key", saved))

	got, err := GetProcessedMutation(transaction, user.ID, "key")
	must(test, err)
	if got.RequestHash != saved.RequestHash || got.Status != saved.Status || got.Headers["ETag"] != saved.Headers["ETag"] ||
		string(got.Body) != string(saved.Body) {
		test.Fatal(got)
	}

	if _, err := GetProcessedMutation(transaction, user.ID+1, "key"); !errors.Is(err, sql.ErrNoRows) {
		test.Fatal(err)
	}

	must(test, transaction.Commit())

	artists, err := GetArtistsByUser(database, user.ID)
	must(test, err)
	if len(artists) != 1 || artists[0].Version != 2 || artists[0].Name != "Updated" {
		test.Fatal(artists)
	}

	albums, err := GetAlbumsByUser(database, user.ID)
	must(test, err)
	if len(albums) != 1 || albums[0].Rating == nil || *albums[0].Rating != 7 {
		test.Fatal(albums)
	}

	for _, lookup := range []func() (bool, error){
		func() (bool, error) {
			return ArtistExistsByID(database, user.ID, artistID)
		},
		func() (bool, error) {
			return AlbumExistsByID(database, user.ID, artistID, albumID)
		},
	} {
		exists, err := lookup()
		must(test, err)
		if !exists {
			test.Error("record missing")
		}
	}

	exists, err := AlbumExistsByID(database, user.ID+1, artistID, albumID)
	must(test, err)
	if exists {
		test.Error("cross-account album visible")
	}

	if _, err := GetAlbumByID(database, user.ID+1, artistID, albumID); !errors.Is(err, sql.ErrNoRows) {
		test.Fatal(err)
	}

	empty, err := GetAlbumsByUser(database, user.ID+1)
	must(test, err)
	if empty == nil || len(empty) != 0 {
		test.Fatal(empty)
	}

	transaction, err = database.Begin()
	must(test, err)
	defer transaction.Rollback()
	must(test, CreateLibrarySavepoint(transaction))
	must(test, DeleteLibraryRecord(transaction, user.ID, AlbumEntity, albumID))
	must(test, RollbackToLibrarySavepoint(transaction))
	if _, err := GetAlbumByID(transaction, user.ID, artistID, albumID); err != nil {
		test.Fatal("savepoint did not restore deleted album", err)
	}

	for _, entity := range []LibraryEntity{"users", "artists; DROP TABLE users"} {
		if _, err := libraryTable(entity); err == nil {
			test.Fatal("unsafe table accepted")
		}

		if _, _, err := LockLibraryRecord(transaction, user.ID, entity, 1); err == nil {
			test.Fatal("unsafe lock accepted")
		}

		if err := DeleteLibraryRecord(transaction, user.ID, entity, 1); err == nil {
			test.Fatal("unsafe delete accepted")
		}

		if _, err := SaveLibraryRecord(transaction, user.ID, entity, 1, nil, true); err == nil {
			test.Fatal("unsafe insert accepted")
		}
	}

	must(test, transaction.Rollback())
}

func TestClosedDatabase(test *testing.T) {
	database := testdb.Open(test)
	must(test, database.Close())

	for _, lookup := range []func() error{
		func() error {
			_, err := GetArtistsByUser(database, 1)

			return err
		},
		func() error {
			_, err := GetArtistByID(database, 1, 1)

			return err
		},
		func() error {
			_, err := GetAlbumsByUser(database, 1)

			return err
		},
		func() error {
			_, err := GetArtistAlbums(database, 1, 1)

			return err
		},
		func() error {
			_, err := GetAlbumByID(database, 1, 1, 1)

			return err
		},
		func() error {
			_, err := CreateUserWithRefreshToken(database, "x", "x", "x", time.Now())

			return err
		},
		func() error {
			return UpdateUserEmail(database, 1, "x", "x")
		},
		func() error {
			return UpdateUserPassword(database, 1, "x", "x")
		},
		func() error {
			return DeleteUser(database, 1, "x")
		},
		func() error {
			_, err := RotateRefreshToken(database, "x", "y", time.Now())

			return err
		},
		func() error {
			_, err := CleanupRefreshTokens(database)

			return err
		},
		func() error {
			return CreateRefreshToken(database, 1, "x", time.Now())
		},
		func() error {
			return RevokeRefreshToken(database, 1)
		},
		func() error {
			_, err := GetRefreshTokenByHash(database, "x")

			return err
		},
	} {
		if err := lookup(); err == nil {
			test.Error("closed database returned success")
		}
	}
}
