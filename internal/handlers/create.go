package handlers

import (
	"database/sql"
	"net/http"
	"strings"

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
		customCode := r.FormValue("custom_code")

		shortCode := customCode
		if shortCode == "" {
			shortCode = generateShortCode()
		}

		if len(shortCode) < 3 || len(shortCode) > 20 {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			renderTemplate(w, "create.html", map[string]any{
				"Error": "Short code must be 3-20 characters",
			})
			return
		}

		for _, c := range shortCode {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				renderTemplate(w, "create.html", map[string]any{
					"Error": "Short code must be alphanumeric",
				})
				return
			}
		}

		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			renderTemplate(w, "create.html", map[string]any{
				"Error": "URL must start with http:// or https://",
			})
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
