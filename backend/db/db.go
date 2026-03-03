package db

import (
	"database/sql"
	"log"
	"log/slog"

	_ "github.com/lib/pq"

	"github.com/Chanadu/better-music/config"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/golang-migrate/migrate/v4"
)

var DB *sql.DB

func Connect() {
	var err error
	DB, err = sql.Open("postgres", config.Conf.DB.Url)

	if err != nil {
		log.Fatal("failed to open database: ", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("failed to connect to database: ", err)
	}

	slog.Info("connected to database successfully")
}

func RunMigrations() {
	migrationInstance, err := migrate.New("file://migrations", config.Conf.DB.Url)

	if err != nil {
		log.Fatal("migration init failed:", err)
	}

	if err := migrationInstance.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("migration failed:", err)
	}

	slog.Info("migrations up to date")
}
