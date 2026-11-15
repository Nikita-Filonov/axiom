package testjunit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigAndNilReporter(t *testing.T) {
	assert.Equal(t, "axiom", newConfig().SuiteName)
	assert.Equal(t, "package", newConfig(WithSuiteName("package")).SuiteName)
	assert.Equal(t, "last", newConfig(WithSuiteName("first"), WithSuiteName("last")).SuiteName)
	for _, test := range []struct {
		name string
		fn   func()
	}{
		{"nil option", func() { Plugin(nil) }},
		{"empty suite name", func() { WithSuiteName("") }},
		{"empty path", func() { WithOutputFile("") }},
		{"blank path", func() { WithOutputFile(" \t") }},
		{"nil handler", func() { Plugin(WithErrorHandler(nil)) }},
		{"nil reporter option", func() { WithReporter(nil) }},
		{"nil config", func() { Plugin()(nil) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			assert.Panics(t, test.fn)
		})
	}
}

func TestOutputPathResolutionFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can access directories regardless of permissions")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.Chmod(dir, 0))
	t.Cleanup(func() { require.NoError(t, os.Chmod(dir, 0700)) })
	_, err := os.Getwd()
	require.Error(t, err)
	assert.PanicsWithValue(t, "testjunit: resolve output file: "+err.Error(), func() {
		Plugin(WithOutputFile("junit.xml"))
	})
}

func TestOutputConfig(t *testing.T) {
	first, second := newConfig(), newConfig()
	assert.NotNil(t, first.Reporter)
	assert.NotSame(t, first.Reporter, second.Reporter)
	assert.NotNil(t, first.ErrorHandler)
	reporter := NewReporter()
	settings := newConfig(WithReporter(reporter), WithOutputFile("reports/junit.xml"))
	path, err := filepath.Abs("reports/junit.xml")
	assert.NoError(t, err)
	assert.Equal(t, path, settings.OutputFile)
	assert.Same(t, reporter, settings.Reporter)
}
