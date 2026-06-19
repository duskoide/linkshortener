package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	ListenAddr    string
	SessionSecret string
	TunnelToken   string
	BaseURL       string
}

func Load() *Config {
	godotenv.Load()

	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "sqlite:data/links.db"),
		ListenAddr:    getEnv("LISTEN_ADDR", "0.0.0.0:8081"),
		SessionSecret: getEnv("SESSION_SECRET", ""),
		TunnelToken:   getEnv("TUNNEL_TOKEN", ""),
		BaseURL:       getEnv("BASE_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
