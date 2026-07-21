package handlers

import (
	"database/sql"
	"net/http"

	"linkshortener/internal/auth"
	"linkshortener/internal/db"
)

func CreatePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTemplate(w, "create.html", nil)
}

func CreatePost(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		url := r.FormValue("url")
		shortCode := r.FormValue("custom_code")
		if shortCode == "" {
			shortCode = generateShortCode()
		}

		shortCode, url, err := validateLinkInput(shortCode, url)
		if err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			renderTemplate(w, "create.html", map[string]any{"Error": err.Error()})
			return
		}

		if err := db.CreateLink(database, shortCode, url); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			renderTemplate(w, "create.html", map[string]any{
				"Error": "Short code already exists",
			})
			return
		}

		http.Redirect(w, r, "/dashboard", http.StatusFound)
	}
}

func generateShortCode() string {
	token := auth.GenerateToken()
	return token[:6]
}
