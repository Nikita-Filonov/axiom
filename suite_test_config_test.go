package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewSuiteTestConfig_UsesEmptyRunnerByDefault(t *testing.T) {
	cfg := axiom.NewSuiteTestConfig()

	assert.Nil(t, cfg.Runner)
}

func TestNewSuiteTestConfig_UsesConfiguredRunner(t *testing.T) {
	runner := axiom.NewRunner()

	cfg := axiom.NewSuiteTestConfig(
		axiom.WithSuiteTestRunner(runner),
	)

	assert.Same(t, runner, cfg.Runner)
}

func TestNewSuiteTestConfig_UsesSequentialModeByDefault(t *testing.T) {
	cfg := axiom.NewSuiteTestConfig()

	assert.False(t, cfg.Parallel)
}

func TestNewSuiteTestConfig_UsesConfiguredParallelMode(t *testing.T) {
	cfg := axiom.NewSuiteTestConfig(
		axiom.WithSuiteTestParallel(),
	)

	assert.True(t, cfg.Parallel)
}

func TestNewSuiteTestConfig_AllowsNilRunner(t *testing.T) {
	cfg := axiom.NewSuiteTestConfig(
		axiom.WithSuiteTestRunner(nil),
	)

	assert.Nil(t, cfg.Runner)
}

func TestNewSuiteTestConfig_AppliesOptionsInOrder(t *testing.T) {
	firstRunner := axiom.NewRunner()
	secondRunner := axiom.NewRunner()

	cfg := axiom.NewSuiteTestConfig(
		axiom.WithSuiteTestRunner(firstRunner),
		axiom.WithSuiteTestRunner(secondRunner),
	)

	assert.Same(t, secondRunner, cfg.Runner)
}

func TestNewSuiteTestConfig_AllowsCustomOptions(t *testing.T) {
	runner := axiom.NewRunner()
	called := false

	cfg := axiom.NewSuiteTestConfig(func(cfg *axiom.SuiteTestConfig) {
		called = true
		cfg.Runner = runner
	})

	assert.True(t, called)
	assert.Same(t, runner, cfg.Runner)
}
