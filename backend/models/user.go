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
