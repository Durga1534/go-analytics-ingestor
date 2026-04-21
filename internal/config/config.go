package config

import (
	"log/slog"
	"os"

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

	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		logger.Warn("No .env file found, using system environment variables")
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
