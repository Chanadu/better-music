// Package testdb creates disposable PostgreSQL databases for integration tests.
package testdb

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"

	_ "github.com/lib/pq"
)

var sequence atomic.Uint64

// Open starts a local PostgreSQL cluster, or creates a separate database on
// TEST_POSTGRES_URL. It never runs migrations against the supplied database.
// Missing PostgreSQL binaries skip integration tests unless REQUIRE_TEST_DB=1.
func Open(test *testing.T) *sql.DB {
	test.Helper()
	database, _ := open(test, true)
	return database
}

// OpenUnmigrated returns an empty disposable database and its connection URL.
func OpenUnmigrated(test *testing.T) (*sql.DB, string) {
	test.Helper()
	return open(test, false)
}

func open(test *testing.T, migrate bool) (*sql.DB, string) {
	test.Helper()
	dsn := os.Getenv("TEST_POSTGRES_URL")
	if dsn == "" {
		for _, name := range []string{"initdb", "pg_ctl"} {
			if _, err := exec.LookPath(name); err != nil {
				if os.Getenv("REQUIRE_TEST_DB") == "1" {
					test.Fatal(err)
				}
				test.Skip("integration tests require PostgreSQL binaries or TEST_POSTGRES_URL")
			}
		}
		dir, err := os.MkdirTemp("", "bm-pg-")
		if err != nil {
			test.Fatal(err)
		}
		test.Cleanup(func() {
			if err := os.RemoveAll(dir); err != nil {
				test.Error(err)
			}
		})
		data := filepath.Join(dir, "data")
		run := func(name string, args ...string) {
			test.Helper()
			if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
				log, _ := os.ReadFile(filepath.Join(dir, "postgres.log"))
				test.Fatalf("%s: %v\n%s\n%s", name, err, out, log)
			}
		}
		run("initdb", "-D", data, "-A", "trust", "-U", "postgres", "--no-locale", "--encoding=UTF8")
		run("pg_ctl", "-D", data, "-l", filepath.Join(dir, "postgres.log"), "-o", fmt.Sprintf("-F -k %s -c listen_addresses=''", dir), "-w", "start")
		test.Cleanup(func() {
			if out, err := exec.Command("pg_ctl", "-D", data, "-m", "immediate", "-w", "stop").CombinedOutput(); err != nil {
				test.Errorf("stop postgres: %v %s", err, out)
			}
		})
		dsn = "postgres://postgres@/postgres?sslmode=disable&host=" + url.QueryEscape(dir)
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() { admin.Close() })
	if err = admin.Ping(); err != nil {
		test.Fatal(err)
	}
	databaseURL, err := url.Parse(dsn)
	if err != nil || (databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql") {
		test.Fatal("TEST_POSTGRES_URL must be a postgres:// URL")
	}
	name := fmt.Sprintf("better_music_test_%d_%d", os.Getpid(), sequence.Add(1))
	if _, err = admin.Exec(`CREATE DATABASE ` + name); err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() {
		if _, err := admin.Exec(`DROP DATABASE ` + name + ` WITH (FORCE)`); err != nil {
			test.Errorf("drop test database: %v", err)
		}
	})
	databaseURL.Path = "/" + name
	database, err := sql.Open("postgres", databaseURL.String())
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() { database.Close() })
	if !migrate {
		return database, databaseURL.String()
	}
	_, file, _, _ := runtime.Caller(0)
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(file), "../../migrations/*.up.sql"))
	if err != nil {
		test.Fatal(err)
	}
	if len(paths) == 0 {
		test.Fatal("no migrations found")
	}
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			test.Fatal(err)
		}
		if _, err = database.Exec(string(body)); err != nil {
			test.Fatalf("migration %s: %v", path, err)
		}
	}
	return database, databaseURL.String()
}
