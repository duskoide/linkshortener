package db

import (
	"database/sql"
	"time"
)

type Link struct {
	ID          int64     `json:"id"`
	ShortCode   string    `json:"short_code"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	ClickCount  int64     `json:"click_count"`
}

func GetLinkByShortCode(db *sql.DB, shortCode string) (*Link, error) {
	link := &Link{}
	err := db.QueryRow(
		"SELECT id, short_code, original_url, created_at, click_count FROM links WHERE short_code = ?",
		shortCode,
	).Scan(&link.ID, &link.ShortCode, &link.OriginalURL, &link.CreatedAt, &link.ClickCount)
	if err != nil {
		return nil, err
	}
	return link, nil
}

func GetAllLinks(db *sql.DB) ([]Link, error) {
	rows, err := db.Query(
		"SELECT id, short_code, original_url, created_at, click_count FROM links ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.ID, &l.ShortCode, &l.OriginalURL, &l.CreatedAt, &l.ClickCount); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func CreateLink(db *sql.DB, shortCode, originalURL string) error {
	_, err := db.Exec(
		"INSERT INTO links (short_code, original_url) VALUES (?, ?)",
		shortCode, originalURL,
	)
	return err
}

func IncrementClickCount(db *sql.DB, linkID int64) error {
	_, err := db.Exec("UPDATE links SET click_count = click_count + 1 WHERE id = ?", linkID)
	return err
}

func DeleteLink(db *sql.DB, linkID int64) error {
	_, err := db.Exec("DELETE FROM links WHERE id = ?", linkID)
	return err
}

func GetLinkByID(db *sql.DB, id int64) (*Link, error) {
	link := &Link{}
	err := db.QueryRow(
		"SELECT id, short_code, original_url, created_at, click_count FROM links WHERE id = ?",
		id,
	).Scan(&link.ID, &link.ShortCode, &link.OriginalURL, &link.CreatedAt, &link.ClickCount)
	if err != nil {
		return nil, err
	}
	return link, nil
}

func UpdateLink(db *sql.DB, linkID int64, shortCode, originalURL string) error {
	_, err := db.Exec(
		"UPDATE links SET short_code = ?, original_url = ? WHERE id = ?",
		shortCode, originalURL, linkID,
	)
	return err
}

func DeleteLinks(db *sql.DB, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := ""
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, id)
	}

	query := "DELETE FROM links WHERE id IN (" + placeholders + ")"
	_, err := db.Exec(query, args...)
	return err
}
