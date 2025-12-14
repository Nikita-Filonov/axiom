package testlogger_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testlogger"
	"github.com/stretchr/testify/assert"
)

type record struct {
	level slog.Level
	msg   string
}

type testHandler struct {
	records []record
}

func (h *testHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *testHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, record{
		level: r.Level,
		msg:   r.Message,
	})
	return nil
}

func (h *testHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *testHandler) WithGroup(_ string) slog.Handler {
	return h
}

func withStdout(buf *bytes.Buffer, fn func()) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old
	buf.ReadFrom(r)
}

func TestPlugin_EmitsLog(t *testing.T) {
	var output bytes.Buffer

	withStdout(&output, func() {
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

	text := output.String()

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
		var out bytes.Buffer

		withStdout(&out, func() {
			cfg := &axiom.Config{
				Context: axiom.Context{Raw: context.Background()},
				Runtime: axiom.NewRuntime(),
			}

			plugin := testlogger.Plugin()
			plugin(cfg)

			cfg.Log(tt.log)
		})

		assert.Contains(t, out.String(), tt.expect)
		assert.Contains(t, out.String(), tt.log.Text)
	}
}
