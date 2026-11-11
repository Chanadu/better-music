package db

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Chanadu/better-music/internal/testdb"
	"github.com/Chanadu/better-music/models"
)

func TestRxDBMigrationPreservesLibrary(test *testing.T) {
	db, _ := testdb.OpenUnmigrated(test)
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(file), "../migrations")
	paths, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		test.Fatal(err)
	}

	for _, path := range paths {
		if filepath.Base(path) >= "010" {
			continue
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			test.Fatal(err)
		}

		if _, err = db.Exec(string(raw)); err != nil {
			test.Fatal(err)
		}
	}

	_, err = db.Exec(`INSERT INTO users(id,email,password_hash) VALUES(41,'owner@test','hash'),(42,'other@test','hash'); INSERT INTO artists(id,user_id,name,spotify_id,cover_url,created_at) VALUES(71,41,'Artist','spotify-artist','artist-image','2020-02-03 04:05:06.123456'); INSERT INTO albums(id,user_id,artist_id,title,year,listened,rating,comment,listened_at,created_at) VALUES(81,41,71,'Album',1999,true,9,'Original notes','2021-02-03 04:05:06.123456','2020-02-04 04:05:06.123456')`)
	if err != nil {
		test.Fatal(err)
	}

	if _, err := db.Exec(`INSERT INTO artists(id,user_id,name) VALUES(72,42,'Other artist'); INSERT INTO albums(id,user_id,artist_id,title) VALUES(82,42,72,'Other album')`); err != nil {
		test.Fatal(err)
	}

	var beforeArtist, beforeAlbum []byte
	if err := db.QueryRow(`SELECT to_jsonb(a) FROM artists a WHERE id=71`).Scan(&beforeArtist); err != nil {
		test.Fatal(err)
	}
	if err := db.QueryRow(`SELECT to_jsonb(a) FROM albums a WHERE id=81`).Scan(&beforeAlbum); err != nil {
		test.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "010_add_rxdb_library.up.sql"))
	if err != nil {
		test.Fatal(err)
	}
	if _, err := db.Exec(string(raw)); err != nil {
		test.Fatal(err)
	}

	var afterArtist, afterAlbum []byte
	if err := db.QueryRow(`SELECT to_jsonb(a) FROM artists a WHERE id=71`).Scan(&afterArtist); err != nil {
		test.Fatal(err)
	}
	if err := db.QueryRow(`SELECT to_jsonb(a) FROM albums a WHERE id=81`).Scan(&afterAlbum); err != nil {
		test.Fatal(err)
	}
	if !bytes.Equal(beforeArtist, afterArtist) || !bytes.Equal(beforeAlbum, afterAlbum) {
		test.Fatal("original library data changed")
	}

	for _, kind := range []models.LibraryKind{models.ArtistKind, models.AlbumKind} {
		tx, err := db.Begin()
		if err != nil {
			test.Fatal(err)
		}
		docs, sequence, err := models.PullLibraryDocuments(tx, "41", kind, 0, 10)
		tx.Rollback()
		if err != nil || len(docs) != 0 || sequence != 0 {
			test.Fatalf("new library was not empty: %v %d %v", docs, sequence, err)
		}
	}

	var columns int
	if err := db.QueryRow(`SELECT count(*) FROM information_schema.columns WHERE
	    ((table_name IN ('library_artists','library_albums') AND column_name='id') OR
	     (table_name='library_albums' AND column_name='artist_id')) AND data_type='uuid'`).Scan(&columns); err != nil || columns != 3 {
		test.Fatalf("new library keys are not UUIDs: %d %v", columns, err)
	}
}
