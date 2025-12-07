package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewParallel_Defaults(t *testing.T) {
	p := axiom.NewParallel()

	assert.False(t, p.Enabled)
}

func TestNewParallel_WithEnabled(t *testing.T) {
	p := axiom.NewParallel(
		axiom.WithParallelEnabled(),
	)

	assert.True(t, p.Enabled)
}

func TestNewParallel_WithDisabled(t *testing.T) {
	p := axiom.NewParallel(
		axiom.WithParallelEnabled(),
		axiom.WithParallelDisabled(), // overrides enabled
	)

	assert.False(t, p.Enabled)
}

func TestWithParallelEnabled(t *testing.T) {
	p := axiom.Parallel{}
	axiom.WithParallelEnabled()(&p)

	assert.True(t, p.Enabled)
}

func TestWithParallelDisabled(t *testing.T) {
	p := axiom.Parallel{Enabled: true}
	axiom.WithParallelDisabled()(&p)

	assert.False(t, p.Enabled)
}

func TestParallelJoin_NoOverride(t *testing.T) {
	base := axiom.Parallel{Enabled: false}
	other := axiom.Parallel{Enabled: false}

	result := base.Join(other)

	assert.False(t, result.Enabled)
}

func TestParallelJoin_OverrideToTrue(t *testing.T) {
	base := axiom.Parallel{Enabled: false}
	other := axiom.Parallel{Enabled: true}

	result := base.Join(other)

	assert.True(t, result.Enabled)
}

func TestParallelJoin_KeepTrue(t *testing.T) {
	base := axiom.Parallel{Enabled: true}
	other := axiom.Parallel{Enabled: false}

	result := base.Join(other)

	assert.True(t, result.Enabled)
}

func TestParallelJoin_TrueWithTrue(t *testing.T) {
	base := axiom.Parallel{Enabled: true}
	other := axiom.Parallel{Enabled: true}

	result := base.Join(other)

	assert.True(t, result.Enabled)
}
