package testlogger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testlogger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_CustomLoggerFormattingAttributesAndFiltering(t *testing.T) {
	var output bytes.Buffer
	var level slog.LevelVar
	level.Set(slog.LevelWarn)
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: &level})).
		With("service", "payments").WithGroup("test").With("id", "case-1")
	cfg := &axiom.Config{}
	testlogger.Plugin(testlogger.WithLogger(logger))(cfg)
	assert.Empty(t, output.String(), "installation must not emit logs")
	cfg.Log(axiom.NewInfoLog("filtered"))
	assert.Empty(t, output.String())
	cfg.Log(axiom.NewWarningLog("quoted \"text\"\nnext line"))
	var record map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &record))
	assert.Equal(t, "WARN", record["level"])
	assert.Equal(t, "quoted \"text\"\nnext line", record["msg"])
	assert.Equal(t, "payments", record["service"])
	assert.Equal(t, map[string]any{"id": "case-1"}, record["test"])
	output.Reset()
	level.Set(slog.LevelDebug)
	cfg.Log(axiom.NewDebugLog("enabled dynamically"))
	assert.Contains(t, output.String(), "enabled dynamically")
}

type logEntry struct {
	ctx    context.Context
	record slog.Record
}

type recordingHandler struct {
	slog.Handler
	entries chan logEntry
}

func (h *recordingHandler) Handle(ctx context.Context, record slog.Record) error {
	h.entries <- logEntry{ctx: ctx, record: record.Clone()}
	return nil
}

func newRecordingHandler(size int) *recordingHandler {
	return &recordingHandler{
		Handler: slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}),
		entries: make(chan logEntry, size),
	}
}

func TestPlugin_UsesContextAtLogTime(t *testing.T) {
	handler := newRecordingHandler(3)
	cfg := &axiom.Config{}
	testlogger.Plugin(testlogger.WithLogger(slog.New(handler)))(cfg)
	cfg.Log(axiom.NewInfoLog("nil context"))
	assert.NotNil(t, (<-handler.entries).ctx, "slog supplies a background context")
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "attempt")
	cfg.Context.Raw = ctx
	cfg.Log(axiom.NewInfoLog("current context"))
	assert.Same(t, ctx, (<-handler.entries).ctx)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	cfg.Context.Raw = canceled
	cfg.Log(axiom.NewInfoLog("cleanup"))
	entry := <-handler.entries
	assert.Same(t, canceled, entry.ctx)
	assert.ErrorIs(t, entry.ctx.Err(), context.Canceled)
	assert.Equal(t, "cleanup", entry.record.Message)
}

func TestPlugin_CustomLoggerReceivesMappedLevels(t *testing.T) {
	handler := newRecordingHandler(8)
	cfg := &axiom.Config{}
	testlogger.Plugin(testlogger.WithLogger(slog.New(handler)))(cfg)
	for _, tc := range []struct {
		input axiom.LogLevel
		want  slog.Level
	}{
		{axiom.LogLevelDebug, slog.LevelDebug},
		{axiom.LogLevelInfo, slog.LevelInfo},
		{axiom.LogLevelWarning, slog.LevelWarn},
		{axiom.LogLevelError, slog.LevelError},
		{axiom.LogLevelFatal, slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	} {
		cfg.Log(axiom.Log{Level: tc.input})
		entry := <-handler.entries
		assert.Equal(t, tc.want, entry.record.Level)
		assert.Empty(t, entry.record.Message)
	}
}

func TestPlugin_ConcurrentInstallationsAndLogging(t *testing.T) {
	const count = 32
	var output bytes.Buffer
	plugin := testlogger.Plugin(testlogger.WithLogger(slog.New(slog.NewJSONHandler(&output, nil))))
	shared := &axiom.Config{}
	plugin(shared)
	var wg sync.WaitGroup
	for i := range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg := &axiom.Config{}
			plugin(cfg)
			cfg.Log(axiom.NewInfoLog(fmt.Sprintf("own-%d", i)))
			shared.Log(axiom.NewInfoLog(fmt.Sprintf("shared-%d", i)))
		}()
	}
	wg.Wait()
	decoder := json.NewDecoder(&output)
	seen := make(map[string]bool)
	for range count * 2 {
		var record struct {
			Message string `json:"msg"`
		}
		require.NoError(t, decoder.Decode(&record))
		assert.False(t, seen[record.Message], "duplicate record %q", record.Message)
		seen[record.Message] = true
	}
	for i := range count {
		assert.True(t, seen[fmt.Sprintf("own-%d", i)])
		assert.True(t, seen[fmt.Sprintf("shared-%d", i)])
	}
	var extra any
	assert.ErrorIs(t, decoder.Decode(&extra), io.EOF)
}
