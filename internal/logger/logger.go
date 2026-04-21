package logger

import (
	"log/slog"
	"os"
)

// NewLogger creates and returns a configured logger instance
func NewLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

// Initialize sets up the default logger
func Initialize() {
	log := NewLogger()
	slog.SetDefault(log)
}
