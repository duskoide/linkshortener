package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"linkshortener/internal/db"

	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
)

func APIQRCode(database *sql.DB, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		link, err := db.GetLinkByID(database, id)
		if err != nil {
			http.Error(w, "Link not found", http.StatusNotFound)
			return
		}

		shortURL := baseURL + "/" + link.ShortCode
		if baseURL == "" {
			scheme := "https"
			if r.TLS == nil {
				scheme = "http"
			}
			shortURL = scheme + "://" + r.Host + "/" + link.ShortCode
		}

		png, err := qrcode.Encode(shortURL, qrcode.Medium, -5)
		if err != nil {
			http.Error(w, "Failed to generate QR", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		w.Write(png)
	}
}