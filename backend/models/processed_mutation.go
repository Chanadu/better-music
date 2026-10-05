package models

import (
	"database/sql"
	"encoding/json"
)

type ProcessedMutation struct {
	RequestHash string
	Status      int
	Headers     map[string]string
	Body        []byte
}

func LockUserForMutation(tx *sql.Tx, userID int) error {
	var id int
	return tx.QueryRow(`SELECT id FROM users WHERE id=$1 FOR UPDATE`, userID).Scan(&id)
}

func CreateLibrarySavepoint(tx *sql.Tx) error {
	_, err := tx.Exec("SAVEPOINT library_change")
	return err
}

func RollbackToLibrarySavepoint(tx *sql.Tx) error {
	_, err := tx.Exec("ROLLBACK TO SAVEPOINT library_change")
	return err
}

func GetProcessedMutation(tx *sql.Tx, userID int, key string) (ProcessedMutation, error) {
	var saved ProcessedMutation
	var headers []byte

	err := tx.QueryRow(`SELECT request_hash,response_status,response_headers,response_body
  FROM processed_mutations WHERE user_id=$1 AND mutation_id=$2`, userID, key).
		Scan(&saved.RequestHash, &saved.Status, &headers, &saved.Body)

	if err == nil {
		err = json.Unmarshal(headers, &saved.Headers)
	}

	return saved, err
}

func SaveProcessedMutation(tx *sql.Tx, userID int, key string, saved ProcessedMutation) error {
	headers, err := json.Marshal(saved.Headers)

	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO processed_mutations
  (user_id,mutation_id,request_hash,response_status,response_headers,response_body)
  VALUES($1,$2,$3,$4,$5,$6)`, userID, key, saved.RequestHash, saved.Status, string(headers), saved.Body)

	return err
}
