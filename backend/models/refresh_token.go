package models

import (
	"database/sql"
	"time"
)

type RefreshToken struct {
	ID        int
	UserID    int
	TokenHash string
	ExpiresAt time.Time
	RevokedAt sql.NullTime
	CreatedAt time.Time
}

func CreateRefreshToken(database *sql.DB, userID int, tokenHash string, expiresAt time.Time) error {
	_, err := database.Exec(
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)

	return err
}

func GetRefreshTokenByHash(database *sql.DB, tokenHash string) (*RefreshToken, error) {
	var token RefreshToken
	err := database.QueryRow(
		`SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1`,
		tokenHash,
	).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt, &token.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func RevokeRefreshToken(database *sql.DB, id int) error {
	_, err := database.Exec(
		`UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL`,
		id,
	)

	return err
}

func CleanupRefreshTokens(database *sql.DB) (int64, error) {
	result, err := database.Exec(
		`DELETE FROM refresh_tokens
		WHERE expires_at < NOW()`,
	)
	if err != nil {
		return 0, err
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return deleted, nil
}
