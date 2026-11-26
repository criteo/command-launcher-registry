package server

import (
	"io"
	"log/slog"
)

// NewLogger creates the server's structured logger, writing to w at the
// configured level ("debug", "info", "warn" or "error"; anything else means
// "info") in the configured format ("json", anything else means text).
func NewLogger(w io.Writer, level, format string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	return newLogger(w, logLevel, format)
}

// NewAuditLogger creates the logger for write audit lines: the same output
// and format as NewLogger, but always at Info, whatever the configured
// level, so audit lines are never filtered out.
func NewAuditLogger(w io.Writer, format string) *slog.Logger {
	return newLogger(w, slog.LevelInfo, format)
}

func newLogger(w io.Writer, level slog.Level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	if format == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}
