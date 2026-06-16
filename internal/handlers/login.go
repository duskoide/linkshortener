package handlers

import (
	"database/sql"
	"net/http"

	"linkshortener/internal/auth"
	"linkshortener/internal/db"
)

func LoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderTemplate(w, "login.html", nil)
}

func LoginPost(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		password := r.FormValue("password")

		storedHash, err := db.GetPasswordHash(database)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		// First login: no password set yet, store this one
		if storedHash == "" {
			hash, err := auth.HashPassword(password)
			if err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			if err := db.SetPasswordHash(database, hash); err != nil {
				http.Error(w, "Internal error", http.StatusInternalServerError)
				return
			}
			storedHash = hash
		}

		if !auth.VerifyPassword(password, storedHash) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			renderTemplate(w, "login.html", map[string]any{
				"Error": "Invalid password",
			})
			return
		}

		// Generate session token
		token := auth.GenerateToken()
		if err := db.CreateSession(database, token, 24); err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(w, r, "/dashboard", http.StatusFound)
	}
}

func Logout(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err == nil {
			db.DeleteSession(database, cookie.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})

		http.Redirect(w, r, "/login", http.StatusFound)
	}
}
