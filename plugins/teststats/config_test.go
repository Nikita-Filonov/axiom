package teststats

import (
	"path/filepath"
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
