package testlogger_test

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testlogger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)
	defer func() { assert.NoError(t, file.Close()) }()

	previous := os.Stdout
	os.Stdout = file
	defer func() { os.Stdout = previous }()

	fn()

	_, err = file.Seek(0, io.SeekStart)
	require.NoError(t, err)
	output, err := io.ReadAll(file)
	require.NoError(t, err)
	return string(output)
}

func TestPlugin_EmitsLog(t *testing.T) {
	text := captureStdout(t, func() {
		cfg := &axiom.Config{
			Context: axiom.Context{
				Raw: context.Background(),
			},
			Runtime: axiom.NewRuntime(),
		}

		plugin := testlogger.Plugin()
		plugin(cfg)

		cfg.Log(axiom.NewWarningLog("hello world"))
	})

	assert.Contains(t, text, "hello world")
	assert.Contains(t, text, "WARN")
}

func TestPlugin_LogLevels(t *testing.T) {
	tests := []struct {
		log    axiom.Log
		expect string
	}{
		{axiom.NewDebugLog("dbg"), "DEBUG"},
		{axiom.NewInfoLog("info"), "INFO"},
		{axiom.NewWarningLog("warn"), "WARN"},
		{axiom.NewErrorLog("err"), "ERROR"},
	}

	for _, tt := range tests {
		text := captureStdout(t, func() {
			cfg := &axiom.Config{
				Context: axiom.Context{Raw: context.Background()},
				Runtime: axiom.NewRuntime(),
			}

			plugin := testlogger.Plugin()
			plugin(cfg)

			cfg.Log(tt.log)
		})

		assert.Contains(t, text, tt.expect)
		assert.Contains(t, text, tt.log.Text)
	}
}
