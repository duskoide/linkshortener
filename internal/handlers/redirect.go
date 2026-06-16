package handlers

import (
	"database/sql"
	"net/http"

	"linkshortener/internal/db"

	"github.com/go-chi/chi/v5"
)

func Redirect(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortCode := chi.URLParam(r, "shortCode")

		link, err := db.GetLinkByShortCode(database, shortCode)
		if err == sql.ErrNoRows {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			renderTemplate(w, "404.html", nil)
			return
		}
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		go db.IncrementClickCount(database, link.ID)

		http.Redirect(w, r, link.OriginalURL, http.StatusFound)
	}
}
