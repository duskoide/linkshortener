package auth

import (
	"context"
	"database/sql"
	"net/http"

	"linkshortener/internal/db"
)

type contextKey string

const AuthKey contextKey = "auth"

type AuthSession struct {
	IsAuthenticated bool
}

func Middleware(database *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			valid, err := db.ValidateSession(database, cookie.Value)
			if err != nil || !valid {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			ctx := context.WithValue(r.Context(), AuthKey, &AuthSession{IsAuthenticated: true})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetAuth(r *http.Request) *AuthSession {
	session, _ := r.Context().Value(AuthKey).(*AuthSession)
	return session
}
