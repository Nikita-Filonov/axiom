package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewParallel_Defaults(t *testing.T) {
	p := axiom.NewParallel()

	assert.False(t, p.Enabled)
	assert.False(t, p.EnabledSet)
}

func TestNewParallel_WithEnabled(t *testing.T) {
	p := axiom.NewParallel(
		axiom.WithParallelEnabled(),
	)

	assert.True(t, p.Enabled)
	assert.True(t, p.EnabledSet)
}

func TestNewParallel_WithDisabled(t *testing.T) {
	p := axiom.NewParallel(
		axiom.WithParallelEnabled(),
		axiom.WithParallelDisabled(),
	)

	assert.False(t, p.Enabled)
	assert.True(t, p.EnabledSet)
}

func TestWithParallelEnabled(t *testing.T) {
	p := axiom.Parallel{}
	axiom.WithParallelEnabled()(&p)

	assert.True(t, p.Enabled)
	assert.True(t, p.EnabledSet)
}

func TestWithParallelDisabled(t *testing.T) {
	p := axiom.Parallel{Enabled: true}
	axiom.WithParallelDisabled()(&p)

	assert.False(t, p.Enabled)
	assert.True(t, p.EnabledSet)
}

func TestParallelJoin_NeitherSet(t *testing.T) {
	base := axiom.Parallel{}
	other := axiom.Parallel{}

	result := base.Join(other)

	assert.False(t, result.Enabled)
	assert.False(t, result.EnabledSet)
}

func TestParallelJoin_OtherEnablesToTrue(t *testing.T) {
	base := axiom.Parallel{}
	other := axiom.NewParallel(axiom.WithParallelEnabled())

	result := base.Join(other)

	assert.True(t, result.Enabled)
	assert.True(t, result.EnabledSet)
}

func TestParallelJoin_OtherDisablesToFalse(t *testing.T) {
	base := axiom.NewParallel(axiom.WithParallelEnabled())
	other := axiom.NewParallel(axiom.WithParallelDisabled())

	result := base.Join(other)

	assert.False(t, result.Enabled)
	assert.True(t, result.EnabledSet)
}

func TestParallelJoin_OtherNotSet_KeepsBase(t *testing.T) {
	base := axiom.NewParallel(axiom.WithParallelEnabled())
	other := axiom.Parallel{}

	result := base.Join(other)

	assert.True(t, result.Enabled)
	assert.True(t, result.EnabledSet)
}

func TestParallelJoin_BothEnabled(t *testing.T) {
	base := axiom.NewParallel(axiom.WithParallelEnabled())
	other := axiom.NewParallel(axiom.WithParallelEnabled())

	result := base.Join(other)

	assert.True(t, result.Enabled)
	assert.True(t, result.EnabledSet)
}

func TestParallelJoin_BaseNotSet_OtherDisabled(t *testing.T) {
	base := axiom.Parallel{}
	other := axiom.NewParallel(axiom.WithParallelDisabled())

	result := base.Join(other)

	assert.False(t, result.Enabled)
	assert.True(t, result.EnabledSet)
}

func TestParallelCopy(t *testing.T) {
	p := axiom.NewParallel(axiom.WithParallelEnabled())
	cp := p.Copy()

	assert.Equal(t, p, cp)
}

func TestParallelCopy_PreservesEnabledSet(t *testing.T) {
	p := axiom.NewParallel(axiom.WithParallelDisabled())
	cp := p.Copy()

	assert.False(t, cp.Enabled)
	assert.True(t, cp.EnabledSet)
}
