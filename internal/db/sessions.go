package db

import "database/sql"

func CreateSession(db *sql.DB, token string, expiresInHours int) error {
	_, err := db.Exec(
		"INSERT INTO sessions (token, expires_at) VALUES (?, datetime('now', '+' || ? || ' hours'))",
		token, expiresInHours,
	)
	return err
}

func ValidateSession(db *sql.DB, token string) (bool, error) {
	var count int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM sessions WHERE token = ? AND expires_at > datetime('now')",
		token,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func DeleteSession(db *sql.DB, token string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func CleanupExpiredSessions(db *sql.DB) error {
	_, err := db.Exec("DELETE FROM sessions WHERE expires_at <= datetime('now')")
	return err
}
