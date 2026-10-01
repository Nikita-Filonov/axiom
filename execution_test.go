package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecution_IdentifiesRunCaseInvocationAndAttempt(t *testing.T) {
	var configs []*axiom.Config
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(func(cfg *axiom.Config) { configs = append(configs, cfg) }))
	c := axiom.NewCase(axiom.WithCaseName("same"))

	runner.RunCase(t, c, func(*axiom.Config) {})
	runner.RunCase(t, c, func(*axiom.Config) {})

	require.Len(t, configs, 4)
	planning, attempt, next := configs[0].Execution, configs[1].Execution, configs[2].Execution
	assert.NotEmpty(t, planning.ID)
	assert.Equal(t, axiom.Execution{ID: planning.ID, Attempt: 0}, planning)
	assert.Equal(t, axiom.Execution{ID: planning.ID, Attempt: 1}, attempt)
	assert.NotEqual(t, planning.ID, next.ID)
}

func TestExecution_NextAttemptKeepsIDAndLeavesReceiverUnchanged(t *testing.T) {
	planning := axiom.NewExecution()
	first := planning.NextAttempt()
	second := first.NextAttempt()

	assert.NotEmpty(t, planning.ID)
	assert.NotEqual(t, planning.ID, axiom.NewExecution().ID)
	assert.Equal(t, axiom.Execution{ID: planning.ID, Attempt: 0}, planning)
	assert.Equal(t, axiom.Execution{ID: planning.ID, Attempt: 1}, first)
	assert.Equal(t, axiom.Execution{ID: planning.ID, Attempt: 2}, second)
}

func TestExecution_PolicySkipEmitsCaseSkip(t *testing.T) {
	type observed struct {
		attempt int
		event   axiom.Event
	}

	for _, tc := range []struct {
		name     string
		parallel bool
		attempt  int
	}{
		{name: "sequential", attempt: 1},
		{name: "parallel retry", parallel: true, attempt: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var events []observed
			runner := axiom.NewRunner(
				axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
				axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
					cfg.Runtime.EmitEventSink(func(e axiom.Event) {
						events = append(events, observed{attempt: cfg.Execution.Attempt, event: e})
					})
				}),
			)
			if tc.parallel {
				runner.Parallel = axiom.NewParallel(axiom.WithParallelEnabled())
			}

			runner.RunCase(t, axiom.NewCase(axiom.WithCaseSkip(axiom.SkipBecause("reason"))), func(*axiom.Config) {
				t.Error("skipped action ran")
			})

			require.Len(t, events, 1)
			assert.Equal(t, axiom.EventTypeCaseSkip, events[0].event.Type)
			assert.Equal(t, "reason", events[0].event.Message)
			assert.Equal(t, tc.attempt, events[0].attempt)
		})
	}
}

func TestExecution_CleanupRegisteredOnCaseStartOrSkipRunsLast(t *testing.T) {
	var order []string
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			cfg.Runtime.EmitEventSink(func(e axiom.Event) {
				if e.Type == axiom.EventTypeCaseStart || e.Type == axiom.EventTypeCaseSkip {
					cfg.T().Cleanup(func() { order = append(order, "observer") })
				}
			})
		}),
		axiom.WithRunnerRuntime(axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(cfg *axiom.Config) {
				cfg.T().Cleanup(func() { order = append(order, "wrap") })
				next(cfg)
			}
		})),
		axiom.WithRunnerFixture("fixture", func(*axiom.Config) (any, func(), error) {
			return 1, func() { order = append(order, "fixture") }, nil
		}),
	)

	runner.RunCase(t, axiom.NewCase(), func(cfg *axiom.Config) {
		axiom.GetFixture[int](cfg, "fixture")
		cfg.T().Cleanup(func() { order = append(order, "test") })
	})
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseSkip(axiom.SkipBecause("reason"))), func(*axiom.Config) {})

	assert.Equal(t, []string{"fixture", "test", "wrap", "observer", "observer"}, order)
}
