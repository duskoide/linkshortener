package db

import "database/sql"

func GetPasswordHash(db *sql.DB) (string, error) {
	var hash string
	err := db.QueryRow("SELECT value FROM settings WHERE key = 'password_hash'").Scan(&hash)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return hash, err
}

func SetPasswordHash(db *sql.DB, hash string) error {
	_, err := db.Exec(
		"INSERT OR REPLACE INTO settings (key, value) VALUES ('password_hash', ?)",
		hash,
	)
	return err
}