package handlers

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

func StaticHandler() http.Handler {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		http.StripPrefix("/static/", http.FileServer(http.FS(sub))).ServeHTTP(w, r)
	})
}