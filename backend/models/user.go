package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID           int
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

func CreateUser(database *sql.DB, email, passwordHash string) (*User, error) {
	var user User

	err := database.QueryRow(
		"INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, created_at",
		email, passwordHash,
	).Scan(&user.ID, &user.CreatedAt)

	user.Email = email
	user.PasswordHash = passwordHash

	return &user, err
}

func GetUserByEmail(database *sql.DB, email string) (*User, error) {
	var user User
	err := database.QueryRow(
		"SELECT id, email, password_hash, created_at FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)

	return &user, err
}

func GetUserByID(database *sql.DB, id int) (*User, error) {
	var user User
	err := database.QueryRow(
		"SELECT id, email, password_hash, created_at FROM users WHERE id = $1",
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)

	return &user, err
}

func UpdateUserEmail(database *sql.DB, id int, email string) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec("UPDATE users SET email = $1 WHERE id = $2", email, id); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL", id); err != nil {
		return err
	}

	return tx.Commit()
}

func UpdateUserPassword(database *sql.DB, id int, passwordHash string) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", passwordHash, id); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL", id); err != nil {
		return err
	}

	return tx.Commit()
}

func DeleteUser(database *sql.DB, id int) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec("DELETE FROM albums WHERE user_id = $1", id); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM artists WHERE user_id = $1", id); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM users WHERE id = $1", id); err != nil {
		return err
	}

	return tx.Commit()
}
