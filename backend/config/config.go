package config

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	Auth    AuthConfig
	Logs    LogConfig
	DB      PostgresConfig
	Server  ServerConfig
	Spotify SpotifyConfig
}

type AuthConfig struct {
	CoreURI       string
	APIKey        string
	APIDomain     string
	WebsiteDomain string
}

func loadAuthConfig() (AuthConfig, error) {
	authConfig := AuthConfig{CoreURI: getEnvWithFallback("SUPERTOKENS_CONNECTION_URI"), APIKey: getEnvWithFallback("SUPERTOKENS_API_KEY"), APIDomain: getEnvWithFallback("AUTH_API_DOMAIN"), WebsiteDomain: getEnvWithFallback("AUTH_WEBSITE_DOMAIN")}
	for _, value := range []string{authConfig.CoreURI, authConfig.APIDomain, authConfig.WebsiteDomain} {
		originURL, err := url.Parse(value)
		if err != nil || (originURL.Scheme != "http" && originURL.Scheme != "https") || originURL.Host == "" || originURL.User != nil || originURL.RawQuery != "" || originURL.Fragment != "" || (originURL.Path != "" && originURL.Path != "/") {
			return authConfig, errors.New("SuperTokens Core/API/website settings must be HTTP(S) origins")
		}
	}
	if authConfig.APIKey == "" {
		return authConfig, errors.New("SUPERTOKENS_API_KEY is required")
	}

	return authConfig, nil
}

type LogConfig struct {
	Enabled bool
	File    string
	Debug   bool
}

type PostgresConfig struct {
	Url string
}

type ServerConfig struct {
	Host string
	Port string
}

type SpotifyConfig struct {
	ClientID     string
	ClientSecret string
}

func getEnvWithFallback(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}

	return ""
}

func LoadConfig() (Config, error) {
	auth, err := loadAuthConfig()
	if err != nil {
		return Config{}, err
	}
	logEnabled, err := strconv.ParseBool(os.Getenv("LOG_ENABLE"))
	if err != nil {
		return Config{}, err
	}

	logDebug, err := strconv.ParseBool(os.Getenv("LOG_DEBUG"))
	if err != nil {
		return Config{}, err
	}

	logFilePath := ""

	if logDebug {
		logFilePath = os.Getenv("LOG_DIR") + "/" + "debug.log"
	} else {
		logFilePath = os.Getenv("LOG_DIR") + "/" + time.Now().Format("2006-01-02_15:04:05")
	}

	spotifyClientID := getEnvWithFallback("SPOTIFY_CLIENT_ID")
	spotifyClientSecret := getEnvWithFallback("SPOTIFY_CLIENT_SECRET")

	config := Config{
		Auth: auth,
		Logs: LogConfig{
			Enabled: logEnabled,
			File:    logFilePath,
			Debug:   logDebug,
		},
		DB: PostgresConfig{
			Url: os.Getenv("POSTGRES_URL"),
		},
		Server: ServerConfig{
			Host: os.Getenv("SERVER_HOST"),
			Port: os.Getenv("SERVER_PORT"),
		},
		Spotify: SpotifyConfig{
			ClientID:     spotifyClientID,
			ClientSecret: spotifyClientSecret,
		},
	}

	if logDebug {
		config.DB.Url += "?sslmode=disable"
	}

	return config, nil
}
