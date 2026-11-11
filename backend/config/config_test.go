package config

import (
	"strings"
	"testing"
)

func environment(test *testing.T) {
	test.Helper()
	for key, value := range map[string]string{
		"SUPERTOKENS_CONNECTION_URI": "http://127.0.0.1:3567",
		"SUPERTOKENS_API_KEY":        "test-core-key",
		"AUTH_API_DOMAIN":            "http://localhost:5173",
		"AUTH_WEBSITE_DOMAIN":        "http://localhost:5173",
		"LOG_ENABLE":                 "false", "LOG_DEBUG": "false", "LOG_DIR": "/tmp/logs",
		"POSTGRES_URL": "postgres://localhost/music", "SERVER_HOST": "127.0.0.1", "SERVER_PORT": "8080",
		"SPOTIFY_CLIENT_ID": " id ", "SPOTIFY_CLIENT_SECRET": " secret ",
	} {
		test.Setenv(key, value)
	}
}
func TestLoadConfig(test *testing.T) {
	environment(test)
	authConfig, err := LoadConfig()
	if err != nil {
		test.Fatal(err)
	}
	if authConfig.Spotify.ClientID != "id" || authConfig.Spotify.ClientSecret != "secret" || authConfig.Server.Port != "8080" || authConfig.Auth.CoreURI != "http://127.0.0.1:3567" {
		test.Fatal("unexpected config")
	}
	test.Setenv("LOG_DEBUG", "true")
	test.Setenv("LOG_ENABLE", "true")
	authConfig, err = LoadConfig()
	if err != nil {
		test.Fatal(err)
	}
	if !authConfig.Logs.Enabled || authConfig.Logs.File != "/tmp/logs/debug.log" || !strings.HasSuffix(authConfig.DB.Url, "?sslmode=disable") {
		test.Fatal("unexpected debug config")
	}
}
func TestInvalidConfig(test *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"LOG_ENABLE", "bad"}, {"LOG_DEBUG", "bad"},
		{"SUPERTOKENS_CONNECTION_URI", ""}, {"SUPERTOKENS_API_KEY", ""}, {"AUTH_API_DOMAIN", ""}, {"AUTH_WEBSITE_DOMAIN", ""},
		{"SUPERTOKENS_CONNECTION_URI", "ftp://localhost"}, {"AUTH_API_DOMAIN", "https://example.com/path"},
		{"AUTH_WEBSITE_DOMAIN", "https://user:password@example.com"}, {"AUTH_WEBSITE_DOMAIN", "https://example.com?query=1"},
	} {
		test.Run(tc.key+tc.value, func(test *testing.T) {
			environment(test)
			test.Setenv(tc.key, tc.value)
			if _, err := LoadConfig(); err == nil {
				test.Fatal("invalid setting accepted")
			}
		})
	}
}
