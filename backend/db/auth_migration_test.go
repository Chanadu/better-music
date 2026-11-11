package db

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Chanadu/better-music/internal/testdb"
	"github.com/golang-migrate/migrate/v4"
)

func TestSuperTokensCutoverFromVersionTen(test *testing.T) {
	database, dsn := testdb.OpenUnmigrated(test)
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "../migrations")
	migrations, err := migrate.New("file://"+dir, dsn)
	if err != nil {
		test.Fatal(err)
	}
	defer migrations.Close()

	if err := migrations.Steps(10); err != nil {
		test.Fatal(err)
	}
	if err := RequireAuthSchema(database); err == nil {
		test.Fatal("admin command accepted a pre-cutover schema")
	}

	if _, err := database.Exec(`
		INSERT INTO users (id, email, password_hash) VALUES (41, 'legacy@test', 'hash');
		INSERT INTO artists (user_id, name) VALUES (41, 'Historical artist');
		INSERT INTO library_artists (id, user_id, name, created_at, edited_at, change_seq)
		VALUES ('00000000-0000-4000-8000-000000000041', 41, 'Legacy artist', '2020-01-01', NOW(), 1);
		INSERT INTO library_albums (user_id, artist_id, title, listened, created_at, edited_at, change_seq)
		VALUES (41, '00000000-0000-4000-8000-000000000041', 'Legacy album', false, '2020-01-01', NOW(), 2);
	`); err != nil {
		test.Fatal(err)
	}

	if err := migrations.Up(); err != nil {
		test.Fatal(err)
	}
	if version, dirty, err := migrations.Version(); err != nil || version != 16 || dirty {
		test.Fatalf("version=%d dirty=%v err=%v", version, dirty, err)
	}
	if err := RequireAuthSchema(database); err != nil {
		test.Fatal(err)
	}

	var remaining int
	if err := database.QueryRow(`
		SELECT (SELECT count(*) FROM users) +
		       (SELECT count(*) FROM library_artists) +
		       (SELECT count(*) FROM library_albums)
	`).Scan(&remaining); err != nil || remaining != 0 {
		test.Fatalf("legacy account or library still present: %d %v", remaining, err)
	}
	if _, err := database.Exec(`
		INSERT INTO users (id, email) VALUES ('core-owner', 'legacy@test');
		INSERT INTO library_artists (id, user_id, name, created_at, edited_at, change_seq)
		VALUES ('00000000-0000-4000-8000-000000000042', 'core-owner', 'Imported artist', '2020-01-01', NOW(), 1);
		INSERT INTO library_albums (user_id, artist_id, title, listened, created_at, edited_at, change_seq)
		VALUES ('core-owner', '00000000-0000-4000-8000-000000000042', 'Imported album', false, '2020-01-01', NOW(), 2);
	`); err != nil {
		test.Fatal(err)
	}
}
