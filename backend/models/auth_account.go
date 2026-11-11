package models

import (
	"context"
	"database/sql"
)

func CheckActiveAuthAccount(ctx context.Context, db *sql.DB, authID string) error {
	var exists int
	return db.QueryRowContext(ctx, `
		SELECT 1 FROM users
		WHERE id = $1 AND NOT auth_disabled AND id NOT LIKE 'pending-%'`, authID).Scan(&exists)
}

func LockActiveAuthAccount(ctx context.Context, tx *sql.Tx, userID string) error {
	var exists int

	return tx.QueryRowContext(ctx, `
		SELECT 1
		FROM users
		WHERE id = $1 AND NOT auth_disabled
		FOR UPDATE`, userID).Scan(&exists)
}

func AuthEmailInUse(ctx context.Context, tx *sql.Tx, email string, userID string) (bool, error) {
	var exists bool

	err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users WHERE lower(email) = $1 AND id <> $2
		) OR EXISTS (
			SELECT 1 FROM auth_account_operations
			WHERE lower(target_email) = $1 AND user_id <> $2
		)`, email, userID).Scan(&exists)

	return exists, err
}

func DisableAuthAccount(ctx context.Context, tx *sql.Tx, userID string) error {
	_, err := tx.ExecContext(ctx, `UPDATE users SET auth_disabled=true WHERE id=$1`, userID)

	return err
}

func RecordAuthAccountOperation(ctx context.Context, tx *sql.Tx, userID string, kind string, targetEmail *string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO auth_account_operations(user_id,kind,target_email) VALUES($1,$2,$3)`, userID, kind, targetEmail)

	return err
}

func LockAuthAccount(ctx context.Context, tx *sql.Tx, userID string) (bool, bool, error) {
	var disabled, revocationPending bool

	err := tx.QueryRowContext(ctx, `
		SELECT auth_disabled, session_revocation_pending FROM users WHERE id = $1 FOR UPDATE`, userID).
		Scan(&disabled, &revocationPending)

	return disabled, revocationPending, err
}

func PendingAuthAccountOperation(ctx context.Context, tx *sql.Tx, userID string) (string, sql.NullString, error) {
	var kind string
	var target sql.NullString

	err := tx.QueryRowContext(ctx, `SELECT kind,target_email FROM auth_account_operations WHERE user_id=$1`, userID).Scan(&kind, &target)

	return kind, target, err
}

func DeleteAuthAccount(ctx context.Context, tx *sql.Tx, userID string) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)

	return err
}

func UpdateAuthAccountEmail(ctx context.Context, tx *sql.Tx, userID string, email string) error {
	_, err := tx.ExecContext(ctx, `UPDATE users SET email=$2 WHERE id=$1`, userID, email)

	return err
}

func ClearAuthAccountOperation(ctx context.Context, tx *sql.Tx, userID string) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM auth_account_operations WHERE user_id=$1`, userID)

	return err
}

func PendingAuthAccountIDs(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT user_id FROM auth_account_operations UNION SELECT id FROM users WHERE session_revocation_pending ORDER BY 1`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}

		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func LockPasswordRecoveryAccount(ctx context.Context, tx *sql.Tx, userID string) (sql.NullString, bool, bool, error) {
	var identity sql.NullString
	var disabled, pending bool

	err := tx.QueryRowContext(ctx, `
		SELECT CASE WHEN id LIKE 'pending-%' THEN NULL ELSE id END,
		       auth_disabled,
		       EXISTS (SELECT 1 FROM auth_account_operations WHERE user_id = users.id)
		FROM users
		WHERE id = $1
		FOR UPDATE`, userID).Scan(&identity, &disabled, &pending)

	return identity, disabled, pending, err
}

func ActiveAccountEmail(ctx context.Context, db *sql.DB, userID string) (string, error) {
	var email string
	err := db.QueryRowContext(ctx, `SELECT email FROM users WHERE id=$1 AND NOT auth_disabled`, userID).Scan(&email)

	return email, err
}

func ActiveAccountSession(ctx context.Context, db *sql.DB, userID string) (string, string, error) {
	var email, epoch string
	err := db.QueryRowContext(ctx, `SELECT email, library_sync_epoch FROM users WHERE id = $1 AND NOT auth_disabled`, userID).
		Scan(&email, &epoch)

	return email, epoch, err
}

func MarkSessionRevocationPending(ctx context.Context, db *sql.DB, userID string) error {
	var id string

	return db.QueryRowContext(ctx, `
		UPDATE users SET session_revocation_pending = true WHERE id = $1 RETURNING id`, userID).Scan(&id)
}

func ClearSessionRevocation(ctx context.Context, tx *sql.Tx, userID string) error {
	_, err := tx.ExecContext(ctx, `UPDATE users SET session_revocation_pending = false WHERE id = $1`, userID)

	return err
}
