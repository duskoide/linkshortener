package handlers

import (
	"database/sql"
	"net/http"

	"linkshortener/internal/db"
)

type DashboardViewModel struct {
	Links          []db.Link
	BaseURL        string
	TotalLinks     int
	LifetimeClicks int64
	TopLink        *db.Link
	RecentLinks    []db.Link
}

func Dashboard(database *sql.DB, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		links, err := db.GetAllLinks(database)
		if err != nil {
			http.Error(w, "Failed to fetch links", http.StatusInternalServerError)
			return
		}

		view := DashboardViewModel{
			Links:      links,
			BaseURL:    baseURL,
			TotalLinks: len(links),
		}
		for i := range links {
			view.LifetimeClicks += links[i].ClickCount
			if view.TopLink == nil || links[i].ClickCount > view.TopLink.ClickCount {
				view.TopLink = &links[i]
			}
		}

		const recentLinkLimit = 3
		if len(links) > recentLinkLimit {
			view.RecentLinks = links[:recentLinkLimit]
		} else {
			view.RecentLinks = links
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderTemplate(w, "dashboard.html", view)
	}
}
