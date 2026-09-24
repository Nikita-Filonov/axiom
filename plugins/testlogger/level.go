package testlogger

import (
	"log/slog"

	"github.com/Nikita-Filonov/axiom"
)

// MapLevel converts an Axiom log level to its slog equivalent.
func MapLevel(l axiom.LogLevel) slog.Level {
	switch l {
	case axiom.LogLevelDebug:
		return slog.LevelDebug
	case axiom.LogLevelWarning:
		return slog.LevelWarn
	case axiom.LogLevelError, axiom.LogLevelFatal:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
