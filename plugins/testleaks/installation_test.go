package testleaks

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_ReinstallationPreservesTrackedResources(t *testing.T) {
	plugin := Plugin(WithoutGoroutines())
	cfg := &axiom.Config{}
	plugin(cfg)
	first := Track(cfg, "first")
	defer first.Release()

	plugin(cfg)
	Plugin()(cfg)
	second := Track(cfg, "second")
	defer second.Release()
	assert.Len(t, cfg.Runtime.TestWraps, 1)
	assert.Same(t, first.state, second.state)
	assert.Len(t, second.state.snapshot(), 2, "reinstallation lost an open resource")

	other := &axiom.Config{}
	plugin(other)
	third := Track(other, "other attempt")
	defer third.Release()
	assert.NotSame(t, first.state, third.state)
	assert.Len(t, other.Runtime.TestWraps, 1)
	assert.Len(t, third.state.snapshot(), 1)

	first.Release()
	remaining := second.state.snapshot()
	require.Len(t, remaining, 1)
	assert.Equal(t, "second", remaining[0].name)
	assert.Len(t, third.state.snapshot(), 1, "release affected another Config")
}

func TestPlugin_PlanningConfigDoesNotInstallState(t *testing.T) {
	plugin := Plugin(WithoutGoroutines())
	cfg := &axiom.Config{Execution: axiom.Execution{ID: "run"}}
	plugin(cfg)
	assert.Empty(t, cfg.Runtime.TestWraps)
	assert.PanicsWithValue(t, "testleaks: plugin not installed", func() { Track(cfg, "resource") })

	cfg.Execution.Attempt = 1
	plugin(cfg)
	plugin(cfg)
	assert.Len(t, cfg.Runtime.TestWraps, 1)
	handle := Track(cfg, "resource")
	handle.Release()
	assert.Empty(t, handle.state.snapshot())
}
