package config

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// AppConfig holds all application configuration
type AppConfig struct {
	RedisURL    string
	DatabaseURL string
	Port        string
	WorkerName  string
}

// Load reads configuration from environment variables and .env file
func Load() *AppConfig {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Try to load .env file from common locations
	// This handles both root directory and cmd/server execution
	envPaths := []string{
		".env",                                   // Current directory
		"../.env",                                // Parent directory
		"../../.env",                             // Two levels up
		filepath.Join(os.Getenv("HOME"), ".env"), // Home directory fallback
	}

	for _, envPath := range envPaths {
		if _, err := os.Stat(envPath); err == nil {
			if err := godotenv.Load(envPath); err == nil {
				logger.Debug("Loaded .env file", "path", envPath)
				break
			}
		}
	}

	// Get hostname for distributed worker identification
	workerName, _ := os.Hostname()
	if workerName == "" {
		workerName = "local-worker"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &AppConfig{
		RedisURL:    os.Getenv("REDIS_URL"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        port,
		WorkerName:  workerName,
	}
}
