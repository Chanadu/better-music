package models

import (
	"database/sql"
	"time"
)

type RefreshToken struct {
	ID        int
	UserID    int
	ExpiresAt time.Time
	RevokedAt sql.NullTime
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
		`SELECT id, user_id, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1`,
		tokenHash,
	).Scan(&token.ID, &token.UserID, &token.ExpiresAt, &token.RevokedAt)
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

// RotateRefreshToken atomically consumes an active refresh token and creates
// its replacement. If the token is missing, expired, or already revoked, it
// returns sql.ErrNoRows.
func RotateRefreshToken(database *sql.DB, tokenHash, replacementHash string, replacementExpiresAt time.Time) (int, error) {
	tx, err := database.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var userID int
	err = tx.QueryRow(
		`UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1
			AND revoked_at IS NULL
			AND expires_at > NOW()
		RETURNING user_id`,
		tokenHash,
	).Scan(&userID)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		userID, replacementHash, replacementExpiresAt,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return userID, nil
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
