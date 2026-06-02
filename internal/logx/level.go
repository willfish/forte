package logx

import (
	"log/slog"
	"os"
	"strings"
)

// Level names persisted in settings and FORTE_LOG_LEVEL.
const (
	LevelOff   = "off"
	LevelError = "error"
	LevelWarn  = "warn"
	LevelInfo  = "info"
	LevelDebug = "debug"
)

// ParseLevel maps a user-facing level string to slog.Level.
// Unknown values default to warn.
func ParseLevel(name string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case LevelOff:
		return slog.LevelError + 4
	case LevelError:
		return slog.LevelError
	case LevelInfo:
		return slog.LevelInfo
	case LevelDebug:
		return slog.LevelDebug
	default:
		return slog.LevelWarn
	}
}

// LevelName returns the canonical name for a slog level.
func LevelName(level slog.Level) string {
	switch {
	case level >= slog.LevelError+4:
		return LevelOff
	case level >= slog.LevelError:
		return LevelError
	case level >= slog.LevelWarn:
		return LevelWarn
	case level >= slog.LevelInfo:
		return LevelInfo
	default:
		return LevelDebug
	}
}

// StartupLevel returns FORTE_LOG_LEVEL when set, otherwise warn.
func StartupLevel() string {
	if v := strings.TrimSpace(os.Getenv("FORTE_LOG_LEVEL")); v != "" {
		return v
	}
	return LevelWarn
}