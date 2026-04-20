package logging

import (
	"log/slog"
	"os"
	"strings"
)

// NewLogger initializes the logger with a specific level.
func NewLogger(levelStr string) *slog.Logger {
	var level slog.Level

	// Map strings to slog Levels.
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create the Handler Options with the chosen level
	opts := &slog.HandlerOptions{
		Level: level,
		// Optional: AddSource: true, // Uncomment if you want file:line number in logs
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)

	return slog.New(handler)
}
