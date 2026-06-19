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

func APIBulkDeleteLink(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			IDs []int64 `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
			return
		}

		if len(req.IDs) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"deleted":0}`))
			return
		}

		if len(req.IDs) > 1000 {
			http.Error(w, `{"error":"Too many IDs (max 1000)"}`, http.StatusBadRequest)
			return
		}

		if err := db.DeleteLinks(database, req.IDs); err != nil {
			http.Error(w, `{"error":"Failed to delete"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"deleted":` + strconv.FormatInt(int64(len(req.IDs)), 10) + `}`))
	}
}
