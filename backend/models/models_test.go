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

func TestClosedDatabase(test *testing.T) {
	database := testdb.Open(test)
	must(test, database.Close())

	for _, lookup := range []func() error{
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
