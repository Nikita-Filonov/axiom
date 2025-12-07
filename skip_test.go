package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewSkip_Default(t *testing.T) {
	s := axiom.NewSkip()

	assert.False(t, s.Enabled)
	assert.Equal(t, "", s.Reason)
}

func TestNewSkip_WithOptions(t *testing.T) {
	s := axiom.NewSkip(
		axiom.WithSkipEnabled(true),
		axiom.WithSkipReason("maintenance"),
	)

	assert.True(t, s.Enabled)
	assert.Equal(t, "maintenance", s.Reason)
}

func TestSkipBecause(t *testing.T) {
	s := axiom.NewSkip(
		axiom.SkipBecause("not supported"),
	)

	assert.True(t, s.Enabled)
	assert.Equal(t, "not supported", s.Reason)
}

func TestWithSkipEnabled_Only(t *testing.T) {
	s := axiom.NewSkip(
		axiom.WithSkipEnabled(true),
	)

	assert.True(t, s.Enabled)
	assert.Equal(t, "", s.Reason)
}

func TestWithSkipReason_Only(t *testing.T) {
	s := axiom.NewSkip(
		axiom.WithSkipReason("custom"),
	)

	assert.False(t, s.Enabled) // enabled wasn't set
	assert.Equal(t, "custom", s.Reason)
}

func TestSkipJoin_OverrideEnabled(t *testing.T) {
	base := axiom.Skip{
		Enabled: false,
		Reason:  "base",
	}
	other := axiom.Skip{
		Enabled: true,
	}

	result := base.Join(other)

	assert.True(t, result.Enabled)
	assert.Equal(t, "base", result.Reason) // reason unchanged
}

func TestSkipJoin_OverrideReason(t *testing.T) {
	base := axiom.Skip{Reason: "old"}
	other := axiom.Skip{Reason: "new"}

	result := base.Join(other)

	assert.Equal(t, "new", result.Reason)
}

func TestSkipJoin_NoOverride(t *testing.T) {
	base := axiom.Skip{
		Enabled: true,
		Reason:  "keep",
	}
	other := axiom.Skip{} // empty override

	result := base.Join(other)

	assert.True(t, result.Enabled)
	assert.Equal(t, "keep", result.Reason)
}
