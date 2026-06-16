package handlers

import (
	"database/sql"
	"net/http"

	"linkshortener/internal/db"
)

func Dashboard(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		links, err := db.GetAllLinks(database)
		if err != nil {
			http.Error(w, "Failed to fetch links", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderTemplate(w, "dashboard.html", map[string]any{
			"Links": links,
		})
	}
}
