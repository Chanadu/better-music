package models

import "database/sql"

func LockSyncState(tx *sql.Tx, userID int) (string, error) {
	var epoch string
	err := tx.QueryRow(`SELECT library_sync_epoch FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&epoch)
	return epoch, err
}

func GetSyncSequence(tx *sql.Tx, userID int) (int64, error) {
	var sequence int64
	err := tx.QueryRow(`SELECT library_sync_sequence FROM users WHERE id=$1`, userID).Scan(&sequence)
	return sequence, err
}

func CreateSyncSavepoint(tx *sql.Tx) error {
	_, err := tx.Exec(`SAVEPOINT sync_row`)

	return err
}

func RollbackSyncSavepoint(tx *sql.Tx) error {
	_, err := tx.Exec(`ROLLBACK TO SAVEPOINT sync_row`)

	return err
}

func ReleaseSyncSavepoint(tx *sql.Tx) error {
	_, err := tx.Exec(`RELEASE SAVEPOINT sync_row`)

	return err
}
