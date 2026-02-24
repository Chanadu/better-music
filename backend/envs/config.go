package envs

import (
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

type Config struct {
	Logs LogConfig
	DB   PostgresConfig
}

type LogConfig struct {
	Enabled bool
	File    string
}

type PostgresConfig struct {
	Username string
	Password string
	Host     string
	Port     string
}

func LoadConfig() (*Config, error) {
	logEnabled, err := strconv.ParseBool(os.Getenv("LOG_ENABLE"))
	if err != nil {
		return nil, err
	}

	logDebug, err := strconv.ParseBool(os.Getenv("LOG_DEBUG"))
	if err != nil {
		return nil, err
	}

	logFilePath := ""

	if logDebug {
		logFilePath = os.Getenv("LOG_DIR") + "/" + "debug.log"
	} else {
		logFilePath = os.Getenv("LOG_DIR") + "/" + time.Now().Format("2006-01-02_15:04:05")
	}

	config := &Config{
		Logs: LogConfig{
			Enabled: logEnabled,
			File:    logFilePath,
		},
		DB: PostgresConfig{
			Username: os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
		},
	}

	return config, nil
}
