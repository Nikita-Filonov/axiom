package testquarantine_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testquarantine"
	"github.com/stretchr/testify/assert"
)

func TestNewConfig_Defaults(t *testing.T) {
	cfg := testquarantine.NewConfig()

	assert.Equal(t, testquarantine.DefaultTag, cfg.Tag)
	assert.Equal(t, testquarantine.DefaultReason, cfg.Reason)
	assert.False(t, cfg.Run)
	assert.Nil(t, cfg.Predicate)
}

func TestWithTag_Normalizes(t *testing.T) {
	cfg := testquarantine.NewConfig(testquarantine.WithTag("  Flaky "))

	assert.Equal(t, "flaky", cfg.Tag)
}

func TestWithReason(t *testing.T) {
	cfg := testquarantine.NewConfig(testquarantine.WithReason("JIRA-123"))

	assert.Equal(t, "JIRA-123", cfg.Reason)
}

func TestWithRun(t *testing.T) {
	cfg := testquarantine.NewConfig(testquarantine.WithRun(true))

	assert.True(t, cfg.Run)
}

func TestWithPredicate(t *testing.T) {
	cfg := testquarantine.NewConfig(testquarantine.WithPredicate(
		func(*axiom.Config) (string, bool) { return "", false },
	))

	assert.NotNil(t, cfg.Predicate)
}

func TestConfigFromEnv_Enables(t *testing.T) {
	t.Setenv(testquarantine.AxiomTestQuarantineRun, "1")

	cfg := testquarantine.NewConfig(testquarantine.ConfigFromEnv())

	assert.True(t, cfg.Run)
}

func TestConfigFromEnv_Disables(t *testing.T) {
	t.Setenv(testquarantine.AxiomTestQuarantineRun, "false")

	cfg := testquarantine.NewConfig(testquarantine.WithRun(true), testquarantine.ConfigFromEnv())

	assert.False(t, cfg.Run)
}

func TestConfigFromEnv_UnrecognizedKeepsCurrent(t *testing.T) {
	t.Setenv(testquarantine.AxiomTestQuarantineRun, "maybe")

	cfg := testquarantine.NewConfig(testquarantine.WithRun(true), testquarantine.ConfigFromEnv())

	assert.True(t, cfg.Run, "an unrecognized value must not override the current setting")
}

func TestConfigFromEnv_EmptyKeepsCurrent(t *testing.T) {
	t.Setenv(testquarantine.AxiomTestQuarantineRun, "")

	cfg := testquarantine.NewConfig(testquarantine.WithRun(true), testquarantine.ConfigFromEnv())

	assert.True(t, cfg.Run)
}
