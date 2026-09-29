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

func CreateUserWithRefreshToken(database *sql.DB, email, passwordHash, tokenHash string, expiresAt time.Time) (*User, error) {
	tx, err := database.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var user User
	err = tx.QueryRow(
		"INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, created_at",
		email, passwordHash,
	).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	if _, err = tx.Exec(
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		user.ID, tokenHash, expiresAt,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	user.Email = email
	user.PasswordHash = passwordHash
	return &user, nil
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

func UpdateUserEmail(database *sql.DB, id int, email, expectedPasswordHash string) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		"UPDATE users SET email = $1 WHERE id = $2 AND password_hash = $3",
		email, id, expectedPasswordHash,
	)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return sql.ErrNoRows
	}
	if _, err = tx.Exec("UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL", id); err != nil {
		return err
	}

	return tx.Commit()
}

func UpdateUserPassword(database *sql.DB, id int, passwordHash, expectedPasswordHash string) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.Exec(
		"UPDATE users SET password_hash = $1 WHERE id = $2 AND password_hash = $3",
		passwordHash, id, expectedPasswordHash,
	)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return sql.ErrNoRows
	}
	if _, err = tx.Exec("UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL", id); err != nil {
		return err
	}

	return tx.Commit()
}

func DeleteUser(database *sql.DB, id int, expectedPasswordHash string) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var lockedUserID int
	if err = tx.QueryRow(
		"SELECT id FROM users WHERE id = $1 AND password_hash = $2 FOR UPDATE",
		id, expectedPasswordHash,
	).Scan(&lockedUserID); err != nil {
		return err
	}

	if _, err = tx.Exec("DELETE FROM albums WHERE user_id = $1", id); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM artists WHERE user_id = $1", id); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM users WHERE id = $1", lockedUserID); err != nil {
		return err
	}

	return tx.Commit()
}
