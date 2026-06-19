package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"linkshortener/internal/db"

	"github.com/go-chi/chi/v5"
)

var shortCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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

		req.ShortCode = strings.TrimSpace(req.ShortCode)
		req.OriginalURL = strings.TrimSpace(req.OriginalURL)

		if req.ShortCode == "" || len(req.ShortCode) > 64 || !shortCodePattern.MatchString(req.ShortCode) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"Short code must be 1-64 chars, alphanumeric/underscore/hyphen only"}`))
			return
		}

		parsedURL, err := url.Parse(req.OriginalURL)
		if err != nil || parsedURL.Scheme == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"URL must be a valid http(s) URL"}`))
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