package testtags_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testtags"
	"github.com/stretchr/testify/assert"
)

func TestPlugin_ExcludeMatch(t *testing.T) {
	p := testtags.Plugin(
		testtags.WithConfigExclude("slow"),
	)

	cfg := &axiom.Config{
		Meta: axiom.Meta{Tags: []string{"fast", "slow"}},
	}

	p(cfg)

	assert.True(t, cfg.Skip.Enabled)
	assert.Equal(t, "excluded by tag filter", cfg.Skip.Reason)
}

func TestPlugin_ExcludeNoMatch(t *testing.T) {
	p := testtags.Plugin(
		testtags.WithConfigExclude("db"),
	)

	cfg := &axiom.Config{
		Meta: axiom.Meta{Tags: []string{"fast", "api"}},
	}

	p(cfg)

	assert.False(t, cfg.Skip.Enabled)
}

func TestPlugin_IncludeMatch(t *testing.T) {
	p := testtags.Plugin(
		testtags.WithConfigInclude("api"),
	)

	cfg := &axiom.Config{
		Meta: axiom.Meta{Tags: []string{"fast", "api"}},
	}

	p(cfg)

	assert.False(t, cfg.Skip.Enabled)
}

func TestPlugin_IncludeNoMatch(t *testing.T) {
	p := testtags.Plugin(
		testtags.WithConfigInclude("db"),
	)

	cfg := &axiom.Config{
		Meta: axiom.Meta{Tags: []string{"fast", "api"}},
	}

	p(cfg)

	assert.True(t, cfg.Skip.Enabled)
	assert.Equal(t, "not included by tag filter", cfg.Skip.Reason)
}

func TestPlugin_IncludeAndExclude_PriorityExclude(t *testing.T) {
	p := testtags.Plugin(
		testtags.WithConfigInclude("fast"),
		testtags.WithConfigExclude("slow"),
	)

	cfg := &axiom.Config{
		Meta: axiom.Meta{Tags: []string{"fast", "slow"}},
	}

	p(cfg)

	// Exclude wins
	assert.True(t, cfg.Skip.Enabled)
	assert.Equal(t, "excluded by tag filter", cfg.Skip.Reason)
}

func TestPlugin_NoFilters_NoSkip(t *testing.T) {
	p := testtags.Plugin()

	cfg := &axiom.Config{
		Meta: axiom.Meta{Tags: []string{"anything"}},
	}

	p(cfg)

	assert.False(t, cfg.Skip.Enabled)
}

func TestRunner_WithTagsPlugin_Excluded(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testtags.Plugin(testtags.WithConfigExclude("slow")),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseMeta(axiom.WithMetaTags("slow")),
	)

	called := false

	t.Run("case", func(st *testing.T) {
		r.RunCase(st, c, func(cfg *axiom.Config) {
			called = true
		})
	})

	assert.False(t, called, "test body should NOT run because test was skipped")
}

func TestRunner_WithTagsPlugin_Included(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testtags.Plugin(testtags.WithConfigInclude("api")),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseMeta(axiom.WithMetaTags("fast", "api")),
	)

	called := false

	t.Run("case", func(st *testing.T) {
		r.RunCase(st, c, func(cfg *axiom.Config) {
			called = true
		})
	})

	assert.True(t, called, "test should run because tag matches include")
}
