package teststats

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	stats := NewStats()
	c := newConfig(WithStats(stats), WithOutputDir("results"))
	assert.Same(t, stats, c.Stats)
	assert.True(t, filepath.IsAbs(c.OutputDir))
	assert.NotNil(t, c.ErrorHandler)
	defaults := newConfig()
	assert.NotNil(t, defaults.Stats)
	assert.Empty(t, defaults.OutputDir)
	assert.Panics(t, func() { Plugin(nil) })
	assert.Panics(t, func() { WithStats(nil) })
	assert.Panics(t, func() { Plugin(WithErrorHandler(nil)) })
	for _, path := range []string{"", " \t\n"} {
		assert.Panics(t, func() { WithOutputDir(path) })
	}
}

func TestConfig_RelativeOutputReportsWorkingDirectoryError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not use Unix directory permissions")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can access directories regardless of their permission bits")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.Chmod(dir, 0))
	t.Cleanup(func() { require.NoError(t, os.Chmod(dir, 0700)) })
	_, err := os.Getwd()
	require.Error(t, err)
	assert.PanicsWithValue(t, "teststats: resolve output directory: "+err.Error(), func() {
		Plugin(WithOutputDir("results"))
	})
}

func TestPlugin_DefaultErrorHandlerLogsExportFailures(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocked, []byte("keep"), 0600))
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })
	stats := NewStats()
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(Plugin(
		WithStats(stats),
		WithOutputDir(blocked),
	)))
	t.Run("parent", func(t *testing.T) {
		runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("case")), func(*axiom.Config) {})
	})
	assert.Contains(t, output.String(), "teststats: export run")
	assert.Contains(t, output.String(), blocked)
	assert.Equal(t, 1, stats.Summary().Runs.Passed, "export errors do not fail the case")
	data, err := os.ReadFile(blocked)
	require.NoError(t, err)
	assert.Equal(t, "keep", string(data))
}

func TestPlugin_DefaultCollectorBelongsToPlugin(t *testing.T) {
	plugin := Plugin()
	one := &axiom.Config{}
	two := &axiom.Config{}
	other := &axiom.Config{}
	plugin(one)
	plugin(one)
	plugin(two)
	Plugin()(other)
	r1, ok := axiom.GetLocal(one, installedKey)
	require.True(t, ok)
	r2, ok := axiom.GetLocal(two, installedKey)
	require.True(t, ok)
	r3, ok := axiom.GetLocal(other, installedKey)
	require.True(t, ok)
	require.Len(t, r1.collectors, 1)
	for stats := range r1.collectors {
		_, shared := r2.collectors[stats]
		assert.True(t, shared, "one plugin shares its collector across configs")
		_, sharedWithOtherPlugin := r3.collectors[stats]
		assert.False(t, sharedWithOtherPlugin, "different plugins own different collectors")
	}
	assert.Len(t, one.Runtime.EventSinks, 1)
}
