package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewSkip_Default(t *testing.T) {
	s := axiom.NewSkip()

	assert.False(t, s.Enabled)
	assert.False(t, s.EnabledSet)
	assert.Equal(t, "", s.Reason)
}

func TestNewSkip_WithOptions(t *testing.T) {
	s := axiom.NewSkip(
		axiom.WithSkipEnabled(true),
		axiom.WithSkipReason("maintenance"),
	)

	assert.True(t, s.Enabled)
	assert.True(t, s.EnabledSet)
	assert.Equal(t, "maintenance", s.Reason)
}

func TestSkipBecause(t *testing.T) {
	s := axiom.NewSkip(
		axiom.SkipBecause("not supported"),
	)

	assert.True(t, s.Enabled)
	assert.True(t, s.EnabledSet)
	assert.Equal(t, "not supported", s.Reason)
}

func TestWithSkipEnabled_Only(t *testing.T) {
	s := axiom.NewSkip(
		axiom.WithSkipEnabled(true),
	)

	assert.True(t, s.Enabled)
	assert.True(t, s.EnabledSet)
	assert.Equal(t, "", s.Reason)
}

func TestWithSkipDisabled(t *testing.T) {
	s := axiom.NewSkip(
		axiom.WithSkipDisabled(),
	)

	assert.False(t, s.Enabled)
	assert.True(t, s.EnabledSet)
}

func TestWithSkipReason_Only(t *testing.T) {
	s := axiom.NewSkip(
		axiom.WithSkipReason("custom"),
	)

	assert.False(t, s.Enabled)
	assert.False(t, s.EnabledSet, "reason alone must not mark Enabled as set")
	assert.Equal(t, "custom", s.Reason)
}

func TestSkipJoin_OverrideEnabledTrue(t *testing.T) {
	base := axiom.NewSkip(
		axiom.WithSkipDisabled(),
		axiom.WithSkipReason("base"),
	)
	other := axiom.NewSkip(axiom.WithSkipEnabled(true))

	result := base.Join(other)

	assert.True(t, result.Enabled)
	assert.True(t, result.EnabledSet)
	assert.Equal(t, "base", result.Reason)
}

func TestSkipJoin_CaseLevelDisabledOverridesRunnerLevelEnabled(t *testing.T) {
	runner := axiom.NewSkip(axiom.SkipBecause("under maintenance"))
	caseLevel := axiom.NewSkip(axiom.WithSkipDisabled())

	result := runner.Join(caseLevel)

	assert.False(t, result.Enabled, "explicit case-level disable must override runner-level enable")
	assert.True(t, result.EnabledSet)
	assert.Equal(t, "under maintenance", result.Reason)
}

func TestSkipJoin_OverrideReason(t *testing.T) {
	base := axiom.Skip{Reason: "old"}
	other := axiom.Skip{Reason: "new"}

	result := base.Join(other)

	assert.Equal(t, "new", result.Reason)
}

func TestSkipJoin_NoOverride(t *testing.T) {
	base := axiom.NewSkip(
		axiom.WithSkipEnabled(true),
		axiom.WithSkipReason("keep"),
	)
	other := axiom.Skip{}

	result := base.Join(other)

	assert.True(t, result.Enabled)
	assert.True(t, result.EnabledSet)
	assert.Equal(t, "keep", result.Reason)
}

func TestSkipCopy(t *testing.T) {
	s := axiom.NewSkip(axiom.WithSkipEnabled(true), axiom.WithSkipReason("because"))
	cp := s.Copy()

	assert.Equal(t, s, cp)
}
