package models

import (
	"context"
	"database/sql"
	"errors"
)

func LockAuthEmail(ctx context.Context, conn *sql.Conn, email string) error {
	_, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtextextended($1,7112))`, email)

	return err
}

func UnlockAuthEmail(ctx context.Context, conn *sql.Conn, email string) error {
	_, err := conn.ExecContext(ctx, `SELECT pg_advisory_unlock(hashtextextended($1,7112))`, email)

	return err
}

func CompleteSignup(ctx context.Context, db *sql.DB, pendingID, authID string) error {
	result, err := db.ExecContext(ctx, `
		UPDATE users SET id = $2
		WHERE id = $1 AND NOT auth_disabled AND id LIKE 'pending-%'`, pendingID, authID)
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return errors.New("signup account changed")
	}
	return nil
}

func ReserveSignupAccount(ctx context.Context, db *sql.DB, email string) (string, error) {
	var id string

	err := db.QueryRowContext(ctx, `
		WITH available AS (
			SELECT $1::text AS email
			WHERE NOT EXISTS (
				SELECT 1 FROM auth_account_operations WHERE lower(target_email) = $1
			)
		), existing AS (
			SELECT u.id, u.auth_disabled
			FROM users u JOIN available a ON lower(u.email) = a.email
		), created AS (
			INSERT INTO users (id, email)
			SELECT 'pending-' || gen_random_uuid()::text, email FROM available
			WHERE NOT EXISTS (SELECT 1 FROM existing)
			RETURNING id
		)
		SELECT id FROM created
		UNION ALL
		SELECT id FROM existing WHERE id LIKE 'pending-%' AND NOT auth_disabled`, email).Scan(&id)

	return id, err
}

func PendingSignupAccount(ctx context.Context, db *sql.DB, email string) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, `
		SELECT id FROM users
		WHERE lower(email) = $1 AND id LIKE 'pending-%' AND NOT auth_disabled`, email).Scan(&id)

	return id, err
}
