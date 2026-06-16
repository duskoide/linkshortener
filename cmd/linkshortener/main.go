package main

import (
	"log/slog"
	"net/http"

	"linkshortener/internal/auth"
	"linkshortener/internal/config"
	"linkshortener/internal/db"
	"linkshortener/internal/handlers"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.Load()

	database, err := db.Init(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		panic(err)
	}
	defer database.Close()

	if err := handlers.InitTemplates("templates"); err != nil {
		slog.Error("failed to load templates", "error", err)
		panic(err)
	}

	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Public routes
	r.Get("/login", handlers.LoginPage)
	r.Post("/login", handlers.LoginPost(database))
	r.Post("/logout", handlers.Logout(database))
	r.Get("/{shortCode}", handlers.Redirect(database))

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(database))
		r.Get("/", handlers.Dashboard(database))
		r.Get("/dashboard", handlers.Dashboard(database))
		r.Get("/create", handlers.CreatePage)
		r.Post("/create", handlers.CreatePost(database))
		r.Get("/api/links", handlers.APILinks(database))
		r.Delete("/api/links/{id}", handlers.APIDeleteLink(database))
	})

	slog.Info("link shortener listening", "addr", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, r); err != nil {
		slog.Error("server failed", "error", err)
	}
}
