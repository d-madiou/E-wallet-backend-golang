package config

import (
	"fmt"
	"os"
)

// Config holds all dynamic values for the app.
// We read these from the OS Environment (e.g., Docker, Kubernetes secrets).
type Config struct {
	AppEnv  string // "production" or "development"
	AppPort string // ":8080"
	DBUrl   string // "postgres://user:pass@localhost:5432/ledger"
}

// Load fetches variables or sets defaults.
func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),
		DBUrl:   getEnv("DATABASE_URL", ""), // Essential!
	}

	if cfg.DBUrl == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

// Helper to read env or return default
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
