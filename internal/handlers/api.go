package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"linkshortener/internal/db"

	"github.com/go-chi/chi/v5"
)

func APILinks(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		links, err := db.GetAllLinks(database)
		if err != nil {
			http.Error(w, `{"error":"Failed to fetch links"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(links)
	}
}

func APIDeleteLink(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		if err := db.DeleteLink(database, id); err != nil {
			http.Error(w, "Failed to delete", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
