package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Setup initializes structured JSON logging with the given level.
// Valid levels: "debug", "info", "warn", "error". Defaults to "info".
func Setup(level string) {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))
}

// WithComponent returns a logger with a "component" attribute.
func WithComponent(name string) *slog.Logger {
	return slog.Default().With("component", name)
}
