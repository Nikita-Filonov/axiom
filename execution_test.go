package axiom_test

import (
	"os"
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

func TestExecution_RetryKeepsInvocationIDAcrossAttempts(t *testing.T) {
	for _, tc := range []struct {
		name     string
		parallel string
	}{
		{name: "sequential", parallel: "0"},
		{name: "parallel", parallel: "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := runCaseExecutionHelper(
				t,
				"TestExecution_RetryIdentityHelperProcess",
				"AXIOM_EXECUTION_PARALLEL="+tc.parallel,
			)

			require.Error(t, err, "the first failed attempt remains visible to Go's testing package")
			assert.Contains(t, output, "execution retry identity verified")
		})
	}
}

func TestExecution_RetryIdentityHelperProcess(t *testing.T) {
	if os.Getenv(caseExecutionHelperEnv) != "1" {
		t.Skip("helper process")
	}

	var pluginExecutions []axiom.Execution
	var actionExecutions []axiom.Execution
	runnerOptions := []axiom.RunnerOption{
		axiom.WithRunnerRetry(axiom.WithRetryTimes(3)),
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			pluginExecutions = append(pluginExecutions, cfg.Execution)
		}),
	}
	if os.Getenv("AXIOM_EXECUTION_PARALLEL") == "1" {
		runnerOptions = append(runnerOptions, axiom.WithRunnerParallel(axiom.WithParallelEnabled()))
	}

	t.Cleanup(func() {
		require.Len(t, pluginExecutions, 3, "planning config and two attempt configs")
		invocationID := pluginExecutions[0].ID
		require.NotEmpty(t, invocationID)
		pluginsMatch := assert.Equal(t, []axiom.Execution{
			{ID: invocationID, Attempt: 0},
			{ID: invocationID, Attempt: 1},
			{ID: invocationID, Attempt: 2},
		}, pluginExecutions)
		actionsMatch := assert.Equal(t, pluginExecutions[1:], actionExecutions)
		if pluginsMatch && actionsMatch {
			t.Log("execution retry identity verified")
		}
	})

	axiom.NewRunner(runnerOptions...).RunCase(t, axiom.NewCase(), func(cfg *axiom.Config) {
		actionExecutions = append(actionExecutions, cfg.Execution)
		if cfg.Execution.Attempt == 1 {
			cfg.T().Error("fail the first attempt")
		}
	})
}

func TestExecution_PluginSkipOnFirstAttemptStopsRetries(t *testing.T) {
	for _, tc := range []struct {
		name     string
		parallel bool
	}{
		{name: "sequential"},
		{name: "parallel retry", parallel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			type observedEvent struct {
				execution axiom.Execution
				event     axiom.Event
			}
			var events []observedEvent
			var pluginInstalls, actions int
			runnerOptions := []axiom.RunnerOption{
				axiom.WithRunnerRetry(axiom.WithRetryTimes(3)),
				axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
					pluginInstalls++
					if cfg.Execution.Attempt == 1 {
						cfg.Skip = axiom.NewSkip(axiom.SkipBecause("plugin decision"))
					}
					cfg.Runtime.EmitEventSink(func(event axiom.Event) {
						events = append(events, observedEvent{execution: cfg.Execution, event: event})
					})
				}),
			}
			if tc.parallel {
				runnerOptions = append(runnerOptions, axiom.WithRunnerParallel(axiom.WithParallelEnabled()))
			}

			t.Run("scope", func(t *testing.T) {
				axiom.NewRunner(runnerOptions...).RunCase(t, axiom.NewCase(), func(*axiom.Config) {
					actions++
				})
			})

			require.Len(t, events, 1, "a skipped attempt has no start or finish event")
			assert.Equal(t, axiom.EventTypeCaseSkip, events[0].event.Type)
			assert.Equal(t, "plugin decision", events[0].event.Message)
			assert.Equal(t, 1, events[0].execution.Attempt)
			assert.Equal(t, 2, pluginInstalls, "the skipped attempt must stop retries")
			assert.Zero(t, actions)
		})
	}
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
