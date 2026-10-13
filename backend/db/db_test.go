package db

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Chanadu/better-music/internal/testdb"
)

func TestConnectAndMigrations(test *testing.T) {
	database, dsn := testdb.OpenUnmigrated(test)
	_, file, _, _ := runtime.Caller(0)
	test.Chdir(filepath.Join(filepath.Dir(file), ".."))
	RunMigrations(dsn)
	RunMigrations(dsn)
	connected := Connect(dsn)
	defer connected.Close()
	if err := connected.Ping(); err != nil {
		test.Fatal(err)
	}

	var version int
	var dirty bool
	if err := database.QueryRow("SELECT version,dirty FROM schema_migrations").Scan(&version, &dirty); err != nil || version != 9 || dirty {
		test.Fatalf("version=%d dirty=%v err=%v", version, dirty, err)
	}

	for _, table := range []string{
		"users",
		"artists",
		"albums",
		"refresh_tokens",
		"processed_mutations",
	} {
		var count int
		if err := database.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
			test.Fatalf("table %s: %v", table, err)
		}
	}
}

func TestFatalStartupErrors(test *testing.T) {
	if mode := os.Getenv("BETTER_MUSIC_TEST_FATAL"); mode != "" {
		if mode == "connect" {
			Connect("invalid://database")
		} else {
			RunMigrations("invalid://database")
		}

		return
	}

	for _, mode := range []string{"connect", "migrate"} {
		test.Run(mode, func(test *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestFatalStartupErrors$")
			cmd.Env = append(os.Environ(), "BETTER_MUSIC_TEST_FATAL="+mode)
			out, err := cmd.CombinedOutput()
			if err == nil || len(out) == 0 {
				test.Fatalf("expected fatal startup error: %v %s", err, out)
			}
		})
	}
}
