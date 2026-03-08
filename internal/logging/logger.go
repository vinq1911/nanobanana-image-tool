package logging

import (
	"log/slog"
	"os"
)

// New creates a structured JSON logger.
func New() *slog.Logger {
	level := slog.LevelInfo
	if os.Getenv("NANOBANANA_LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
}
