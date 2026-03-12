package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	Logs       LogConfig
	DB         PostgresConfig
	Server     ServerConfig
	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type LogConfig struct {
	Enabled bool
	File    string
	Debug   bool
}

type PostgresConfig struct {
	Url      string
	Username string
	Password string
	Host     string
	Port     string
}

type ServerConfig struct {
	Host string
	Port string
}

func LoadConfig() (Config, error) {
	logEnabled, err := strconv.ParseBool(os.Getenv("LOG_ENABLE"))
	if err != nil {
		return Config{}, err
	}

	logDebug, err := strconv.ParseBool(os.Getenv("LOG_DEBUG"))
	if err != nil {
		return Config{}, err
	}

	accessTTL := 15 * time.Minute
	accessMinutes := strings.TrimSpace(os.Getenv("JWT_ACCESS_TOKEN_MINUTES"))
	if accessMinutes != "" {
		minutes, err := strconv.Atoi(accessMinutes)
		if err != nil {
			return Config{}, err
		}
		if minutes <= 0 {
			return Config{}, errors.New("JWT_ACCESS_TOKEN_MINUTES must be greater than 0")
		}
		accessTTL = time.Duration(minutes) * time.Minute
	}

	refreshTTL := 30 * 24 * time.Hour
	refreshHours := strings.TrimSpace(os.Getenv("JWT_REFRESH_TOKEN_HOURS"))
	if refreshHours != "" {
		hours, err := strconv.Atoi(refreshHours)
		if err != nil {
			return Config{}, err
		}
		if hours <= 0 {
			return Config{}, errors.New("JWT_REFRESH_TOKEN_HOURS must be greater than 0")
		}
		refreshTTL = time.Duration(hours) * time.Hour
	}

	logFilePath := ""

	if logDebug {
		logFilePath = os.Getenv("LOG_DIR") + "/" + "debug.log"
	} else {
		logFilePath = os.Getenv("LOG_DIR") + "/" + time.Now().Format("2006-01-02_15:04:05")
	}

	config := Config{
		Logs: LogConfig{
			Enabled: logEnabled,
			File:    logFilePath,
			Debug:   logDebug,
		},
		DB: PostgresConfig{
			Username: os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			Url:      os.Getenv("POSTGRES_URL"),
		},
		Server: ServerConfig{
			Host: os.Getenv("SERVER_HOST"),
			Port: os.Getenv("SERVER_PORT"),
		},
		JWTSecret:  os.Getenv("JWT_SECRET"),
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
	}

	if logDebug {
		config.DB.Url += "?sslmode=disable"
	}

	if strings.TrimSpace(config.JWTSecret) == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	return config, nil
}
