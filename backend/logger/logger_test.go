package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupLogger(test *testing.T) {
	original := slog.Default()
	test.Cleanup(func() {
		slog.SetDefault(original)
	})
	if err := SetupLogger(false, ""); err != nil {
		test.Fatal(err)
	}

	path := filepath.Join(test.TempDir(), "test.log")

	for range 2 {
		if err := SetupLogger(true, path); err != nil {
			test.Fatal(err)
		}

		slog.Debug("test entry")
	}

	body, err := os.ReadFile(path)
	if err != nil {
		test.Fatal(err)
	}

	if strings.Count(string(body), "test entry") != 2 || !strings.Contains(string(body), "source=") {
		test.Fatal(string(body))
	}

	if err := SetupLogger(true, filepath.Join(test.TempDir(), "missing", "test.log")); err == nil {
		test.Error("expected invalid path error")
	}
}
