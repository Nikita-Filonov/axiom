package testlogger_test

import (
	"log/slog"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testlogger"
	"github.com/stretchr/testify/assert"
)

func TestMapLevel(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, testlogger.MapLevel(axiom.LogLevelDebug))
	assert.Equal(t, slog.LevelInfo, testlogger.MapLevel(axiom.LogLevelInfo))
	assert.Equal(t, slog.LevelWarn, testlogger.MapLevel(axiom.LogLevelWarning))
	assert.Equal(t, slog.LevelError, testlogger.MapLevel(axiom.LogLevelError))
	assert.Equal(t, slog.LevelError, testlogger.MapLevel(axiom.LogLevelFatal))
}
