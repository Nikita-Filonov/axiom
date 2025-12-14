package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewLog_Empty(t *testing.T) {
	l := axiom.NewLog()

	assert.Equal(t, "", l.Text)
	assert.Equal(t, axiom.LogLevel(""), l.Level)
}

func TestNewLog_WithOptions(t *testing.T) {
	l := axiom.NewLog(
		axiom.WithLogText("hello"),
		axiom.WithLogLevel(axiom.LogLevelInfo),
	)

	assert.Equal(t, "hello", l.Text)
	assert.Equal(t, axiom.LogLevelInfo, l.Level)
}

func TestWithLogText(t *testing.T) {
	l := axiom.NewLog(
		axiom.WithLogText("text"),
	)

	assert.Equal(t, "text", l.Text)
}

func TestWithLogLevel(t *testing.T) {
	l := axiom.NewLog(
		axiom.WithLogLevel(axiom.LogLevelWarning),
	)

	assert.Equal(t, axiom.LogLevelWarning, l.Level)
}

func TestNewDebugLog(t *testing.T) {
	l := axiom.NewDebugLog("debug msg")

	assert.Equal(t, axiom.LogLevelDebug, l.Level)
	assert.Equal(t, "debug msg", l.Text)
}

func TestNewInfoLog(t *testing.T) {
	l := axiom.NewInfoLog("info msg")

	assert.Equal(t, axiom.LogLevelInfo, l.Level)
	assert.Equal(t, "info msg", l.Text)
}

func TestNewWarningLog(t *testing.T) {
	l := axiom.NewWarningLog("warn msg")

	assert.Equal(t, axiom.LogLevelWarning, l.Level)
	assert.Equal(t, "warn msg", l.Text)
}

func TestNewErrorLog(t *testing.T) {
	l := axiom.NewErrorLog("error msg")

	assert.Equal(t, axiom.LogLevelError, l.Level)
	assert.Equal(t, "error msg", l.Text)
}

func TestNewFatalLog(t *testing.T) {
	l := axiom.NewFatalLog("fatal msg")

	assert.Equal(t, axiom.LogLevelFatal, l.Level)
	assert.Equal(t, "fatal msg", l.Text)
}
