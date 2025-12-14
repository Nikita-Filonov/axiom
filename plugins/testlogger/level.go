package testlogger

import (
	"log/slog"

	"github.com/Nikita-Filonov/axiom"
)

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
