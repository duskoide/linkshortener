package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"linkshortener/internal/db"

	"github.com/go-chi/chi/v5"
)

type editRequest struct {
	ShortCode   string `json:"short_code"`
	OriginalURL string `json:"original_url"`
}

func APIEditLink(database *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"Invalid ID"}`, http.StatusBadRequest)
			return
		}

		var req editRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid JSON"}`, http.StatusBadRequest)
			return
		}

		req.ShortCode, req.OriginalURL, err = validateLinkInput(req.ShortCode, req.OriginalURL)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		err = db.UpdateLink(database, id, req.ShortCode, req.OriginalURL)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"error":"Short code already taken"}`))
				return
			}
			http.Error(w, `{"error":"Failed to update"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}
}
