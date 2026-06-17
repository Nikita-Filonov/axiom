package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSuiteConfig_UsesDefaultRunner(t *testing.T) {
	cfg := axiom.NewSuiteConfig()

	require.NotNil(t, cfg.Runner)
}

func TestNewSuiteConfig_UsesConfiguredRunner(t *testing.T) {
	runner := axiom.NewRunner()

	cfg := axiom.NewSuiteConfig(
		axiom.WithSuiteConfigRunner(runner),
	)

	assert.Same(t, runner, cfg.Runner)
}

func TestNewSuiteConfig_UsesSequentialModeByDefault(t *testing.T) {
	cfg := axiom.NewSuiteConfig()

	assert.False(t, cfg.Parallel)
}

func TestNewSuiteConfig_UsesConfiguredParallelMode(t *testing.T) {
	cfg := axiom.NewSuiteConfig(
		axiom.WithSuiteConfigParallel(),
	)

	assert.True(t, cfg.Parallel)
}

func TestNewSuiteConfig_UsesDefaultRunnerWhenOptionSetsNilRunner(t *testing.T) {
	cfg := axiom.NewSuiteConfig(
		axiom.WithSuiteConfigRunner(nil),
	)

	require.NotNil(t, cfg.Runner)
}

func TestNewSuiteConfig_AppliesOptionsInOrder(t *testing.T) {
	firstRunner := axiom.NewRunner()
	secondRunner := axiom.NewRunner()

	cfg := axiom.NewSuiteConfig(
		axiom.WithSuiteConfigRunner(firstRunner),
		axiom.WithSuiteConfigRunner(secondRunner),
	)

	assert.Same(t, secondRunner, cfg.Runner)
}

func TestNewSuiteConfig_AllowsCustomOptions(t *testing.T) {
	runner := axiom.NewRunner()
	called := false

	cfg := axiom.NewSuiteConfig(func(cfg *axiom.SuiteConfig) {
		called = true
		cfg.Runner = runner
	})

	assert.True(t, called)
	assert.Same(t, runner, cfg.Runner)
}
