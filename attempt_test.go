package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttemptExcludeKeepsFirstReason(t *testing.T) {
	attempt := axiom.Attempt{Decision: axiom.Decision{Kind: axiom.DecisionRun}}
	attempt.Exclude("")
	attempt.Exclude("later exclusion")

	assert.Equal(t, axiom.Decision{Kind: axiom.DecisionExclude}, attempt.Decision)
}

func TestAttempt_IdentifiesRunCaseInvocationAndNumber(t *testing.T) {
	var configs []*axiom.Config
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(func(cfg *axiom.Config) { configs = append(configs, cfg) }))
	c := axiom.NewCase(axiom.WithCaseName("same"))

	runner.RunCase(t, c, func(*axiom.Config) {})
	runner.RunCase(t, c, func(*axiom.Config) {})

	require.Len(t, configs, 4)
	planning, attempt, next := configs[0].Attempt, configs[1].Attempt, configs[2].Attempt
	assert.NotEmpty(t, planning.RunID)
	assert.Equal(t, planning.RunID, attempt.RunID)
	assert.Zero(t, planning.Number)
	assert.Equal(t, 1, attempt.Number)
	assert.NotEqual(t, planning.RunID, next.RunID)
	assert.Equal(t, axiom.DecisionRun, planning.Decision.Kind)
	assert.Equal(t, axiom.DecisionRun, attempt.Decision.Kind)
	assert.Nil(t, planning.Outcome)
	require.NotNil(t, attempt.Outcome)
	assert.Equal(t, axiom.AttemptPassed, attempt.Outcome.Status)
}

func TestAttempt_DoesNotInheritPlanningDecision(t *testing.T) {
	var configs []*axiom.Config
	ran := false
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
		configs = append(configs, cfg)
		if cfg.Attempt.Number == 0 {
			cfg.Attempt.Exclude("planning only")
		}
	}))

	runner.RunCase(t, axiom.NewCase(), func(*axiom.Config) { ran = true })

	assert.True(t, ran)
	require.Len(t, configs, 2)
	assert.Equal(t, []axiom.Decision{
		{Kind: axiom.DecisionExclude, Reason: "planning only"},
		{Kind: axiom.DecisionRun},
	}, []axiom.Decision{configs[0].Attempt.Decision, configs[1].Attempt.Decision})
}

func TestAttemptSinksWaitForNativeCompletion(t *testing.T) {
	var attempts []axiom.Attempt
	var cleaned bool
	runner := axiom.NewRunner(axiom.WithRunnerParallel(axiom.WithParallelEnabled()), axiom.WithRunnerRuntime(
		axiom.WithRuntimeAttemptSink(func(attempt axiom.Attempt) {
			assert.True(t, cleaned)
			attempts = append(attempts, attempt)
		}),
	))
	t.Run("parent", func(parent *testing.T) {
		runner.RunCase(parent, axiom.NewCase(axiom.WithCaseName("parallel")), func(cfg *axiom.Config) {
			cfg.T().Run("child", func(child *testing.T) {
				child.Parallel()
				child.Cleanup(func() { cleaned = true })
			})
		})
		assert.Empty(parent, attempts)
	})
	require.Len(t, attempts, 1)
	assert.Equal(t, axiom.AttemptPassed, attempts[0].Outcome.Status)
}
