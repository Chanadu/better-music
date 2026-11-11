package db

import (
	"database/sql"
	"fmt"
	"log"
	"log/slog"

	_ "github.com/lib/pq"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/golang-migrate/migrate/v4"
)

func Connect(databaseURL string) *sql.DB {
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal("failed to open database connection: ", err)
	}

	if err = database.Ping(); err != nil {
		_ = database.Close()
		log.Fatal("failed to ping database: ", err)
	}

	slog.Info("connected to database successfully")
	return database
}

func RunMigrations(databaseURL string) {
	migrationInstance, err := migrate.New("file://migrations", databaseURL)

	if err != nil {
		log.Fatal("failed to create migration instance: ", err)
	}

	if err := migrationInstance.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("migration failed: ", err)
	}

	slog.Info("migrations up to date")
}

func RequireAuthSchema(database *sql.DB) error {
	const requiredVersion = 16
	var version int
	var dirty bool

	if err := database.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil {
		return fmt.Errorf("check auth schema: %w", err)
	}

	if version < requiredVersion || dirty {
		return fmt.Errorf("auth-admin requires a clean database at schema version %d or later (found version %d, dirty=%t)", requiredVersion, version, dirty)
	}

	return nil
}
