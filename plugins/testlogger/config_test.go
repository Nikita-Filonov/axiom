package testlogger_test

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testlogger"
	"github.com/stretchr/testify/assert"
)

func TestPlugin_InvalidOptions(t *testing.T) {
	assert.PanicsWithValue(t, "testlogger: nil option", func() { testlogger.Plugin(nil) })
	assert.PanicsWithValue(t, "testlogger: nil logger", func() {
		testlogger.Plugin(testlogger.WithLogger(nil))
	})
	assert.PanicsWithValue(t, "testlogger: nil logger", func() {
		testlogger.Plugin(func(c *testlogger.Config) { c.Logger = nil })
	})
}

func TestPlugin_LastLoggerWinsAndOptionsRunOnce(t *testing.T) {
	var first, second bytes.Buffer
	firstLogger := slog.New(slog.NewTextHandler(&first, nil))
	secondLogger := slog.New(slog.NewTextHandler(&second, nil))
	var calls int
	plugin := testlogger.Plugin(
		testlogger.WithLogger(firstLogger),
		testlogger.WithLogger(secondLogger),
		func(*testlogger.Config) { calls++ },
	)
	assert.Equal(t, 1, calls)
	for range 2 {
		cfg := &axiom.Config{}
		plugin(cfg)
		assert.Len(t, cfg.Runtime.LogSinks, 1)
		cfg.Log(axiom.NewInfoLog("hello"))
	}
	assert.Equal(t, 1, calls)
	assert.Empty(t, first.String())
	assert.Equal(t, 2, bytes.Count(second.Bytes(), []byte("msg=hello")))
}
