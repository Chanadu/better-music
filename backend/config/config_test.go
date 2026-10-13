package config

import (
	"strings"
	"testing"
	"time"
)

func environment(test *testing.T) {
	test.Helper()

	for key, value := range map[string]string{
		"LOG_ENABLE":               "false",
		"LOG_DEBUG":                "false",
		"LOG_DIR":                  "/tmp/logs",
		"JWT_SECRET":               "test-secret",
		"JWT_ACCESS_TOKEN_MINUTES": "",
		"JWT_REFRESH_TOKEN_HOURS":  "",
		"POSTGRES_URL":             "postgres://localhost/music",
		"SERVER_HOST":              "127.0.0.1",
		"SERVER_PORT":              "8080",
		"SPOTIFY_CLIENT_ID":        " id ",
		"SPOTIFY_CLIENT_SECRET":    " secret ",
	} {
		test.Setenv(key, value)
	}
}

func TestLoadConfig(test *testing.T) {
	environment(test)
	loadedConfig, err := LoadConfig()
	if err != nil {
		test.Fatal(err)
	}

	if loadedConfig.AccessTTL != 15*time.Minute || loadedConfig.RefreshTTL != 720*time.Hour || loadedConfig.Spotify.ClientID != "id" ||
		loadedConfig.Spotify.ClientSecret != "secret" ||
		loadedConfig.Server.Port != "8080" ||
		loadedConfig.JWTSecret != "test-secret" {
		test.Fatalf("unexpected config: %+v", loadedConfig)
	}

	test.Setenv("LOG_DEBUG", "true")
	test.Setenv("LOG_ENABLE", "true")
	test.Setenv("JWT_ACCESS_TOKEN_MINUTES", " 5 ")
	test.Setenv("JWT_REFRESH_TOKEN_HOURS", "48")
	loadedConfig, err = LoadConfig()
	if err != nil {
		test.Fatal(err)
	}

	if !loadedConfig.Logs.Enabled || loadedConfig.Logs.File != "/tmp/logs/debug.log" ||
		!strings.HasSuffix(loadedConfig.DB.Url, "?sslmode=disable") ||
		loadedConfig.AccessTTL != 5*time.Minute ||
		loadedConfig.RefreshTTL != 48*time.Hour {
		test.Fatalf("unexpected config: %+v", loadedConfig)
	}
}

func TestInvalidConfig(test *testing.T) {
	for _, testCase := range []struct {
		key, value string
	}{
		{"LOG_ENABLE", "bad"},
		{"LOG_DEBUG", "bad"},
		{"JWT_SECRET", " "},
		{"JWT_ACCESS_TOKEN_MINUTES", "bad"},
		{"JWT_ACCESS_TOKEN_MINUTES", "0"},
		{"JWT_ACCESS_TOKEN_MINUTES", "-1"},
		{"JWT_REFRESH_TOKEN_HOURS", "bad"},
		{"JWT_REFRESH_TOKEN_HOURS", "0"},
		{"JWT_REFRESH_TOKEN_HOURS", "-1"},
	} {
		test.Run(testCase.key+testCase.value, func(test *testing.T) {
			environment(test)
			test.Setenv(testCase.key, testCase.value)
			if _, err := LoadConfig(); err == nil {
				test.Fatal("expected error")
			}
		})
	}
}
