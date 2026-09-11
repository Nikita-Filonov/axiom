package testquarantine_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testquarantine"
	"github.com/stretchr/testify/assert"
)

func TestPlugin_TagMatch_Skips(t *testing.T) {
	p := testquarantine.Plugin()

	cfg := &axiom.Config{Meta: axiom.Meta{Tags: []string{"api", "quarantine"}}}
	p(cfg)

	assert.True(t, cfg.Skip.Enabled)
	assert.Equal(t, "quarantined: flaky", cfg.Skip.Reason)
}

func TestPlugin_TagMatch_IsCaseInsensitive(t *testing.T) {
	p := testquarantine.Plugin()

	cfg := &axiom.Config{Meta: axiom.Meta{Tags: []string{"Quarantine"}}}
	p(cfg)

	assert.True(t, cfg.Skip.Enabled)
}

func TestPlugin_NoTag_DoesNotSkip(t *testing.T) {
	p := testquarantine.Plugin()

	cfg := &axiom.Config{Meta: axiom.Meta{Tags: []string{"api"}}}
	p(cfg)

	assert.False(t, cfg.Skip.Enabled)
}

func TestPlugin_EmptyTag_DoesNotSkip(t *testing.T) {
	p := testquarantine.Plugin(testquarantine.WithTag(""))

	cfg := &axiom.Config{Meta: axiom.Meta{Tags: []string{"quarantine"}}}
	p(cfg)

	assert.False(t, cfg.Skip.Enabled, "an empty tag must never match")
}

func TestPlugin_CustomTagAndReason(t *testing.T) {
	p := testquarantine.Plugin(
		testquarantine.WithTag("unstable"),
		testquarantine.WithReason("JIRA-42"),
	)

	cfg := &axiom.Config{Meta: axiom.Meta{Tags: []string{"unstable"}}}
	p(cfg)

	assert.True(t, cfg.Skip.Enabled)
	assert.Equal(t, "quarantined: JIRA-42", cfg.Skip.Reason)
}

func TestPlugin_Predicate_Skips(t *testing.T) {
	p := testquarantine.Plugin(testquarantine.WithPredicate(
		func(cfg *axiom.Config) (string, bool) { return "owned by team-x", true },
	))

	cfg := &axiom.Config{}
	p(cfg)

	assert.True(t, cfg.Skip.Enabled)
	assert.Equal(t, "quarantined: owned by team-x", cfg.Skip.Reason)
}

func TestPlugin_Predicate_NoQuarantine(t *testing.T) {
	p := testquarantine.Plugin(testquarantine.WithPredicate(
		func(cfg *axiom.Config) (string, bool) { return "", false },
	))

	cfg := &axiom.Config{}
	p(cfg)

	assert.False(t, cfg.Skip.Enabled)
}

func TestPlugin_Predicate_EmptyReasonFallsBackToDefault(t *testing.T) {
	p := testquarantine.Plugin(testquarantine.WithPredicate(
		func(cfg *axiom.Config) (string, bool) { return "", true },
	))

	cfg := &axiom.Config{}
	p(cfg)

	assert.True(t, cfg.Skip.Enabled)
	assert.Equal(t, "quarantined: flaky", cfg.Skip.Reason)
}

func TestPlugin_Run_DoesNotSkip(t *testing.T) {
	p := testquarantine.Plugin(testquarantine.WithRun(true))

	cfg := &axiom.Config{Meta: axiom.Meta{Tags: []string{"quarantine"}}}
	p(cfg)

	assert.False(t, cfg.Skip.Enabled, "WithRun(true) must execute quarantined cases")
}

func TestPlugin_PreservesExistingSkipReason(t *testing.T) {
	p := testquarantine.Plugin()

	cfg := &axiom.Config{
		Meta: axiom.Meta{Tags: []string{"quarantine"}},
		Skip: axiom.NewSkip(axiom.SkipBecause("already skipped")),
	}
	p(cfg)

	assert.True(t, cfg.Skip.Enabled)
	assert.Equal(t, "quarantined: flaky", cfg.Skip.Reason)
}

func TestRunner_Quarantined_SkipsBody(t *testing.T) {
	r := axiom.NewRunner(axiom.WithRunnerPlugins(testquarantine.Plugin()))
	c := axiom.NewCase(axiom.WithCaseMeta(axiom.WithMetaTags("quarantine")))

	called := false
	t.Run("case", func(st *testing.T) {
		r.RunCase(st, c, func(cfg *axiom.Config) { called = true })
	})

	assert.False(t, called, "quarantined case body must not run")
}

func TestRunner_QuarantinedButRun_RunsBody(t *testing.T) {
	r := axiom.NewRunner(axiom.WithRunnerPlugins(testquarantine.Plugin(testquarantine.WithRun(true))))
	c := axiom.NewCase(axiom.WithCaseMeta(axiom.WithMetaTags("quarantine")))

	called := false
	t.Run("case", func(st *testing.T) {
		r.RunCase(st, c, func(cfg *axiom.Config) { called = true })
	})

	assert.True(t, called, "WithRun(true) must execute the quarantined case body")
}
