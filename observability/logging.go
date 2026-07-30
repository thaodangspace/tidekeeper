// Package observability configures bounded, structured service telemetry.
package observability

import (
	"io"
	"log/slog"
)

// NewLogger creates a JSON logger at the validated level.
func NewLogger(output io.Writer, level string) *slog.Logger {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slogLevel}))
}
