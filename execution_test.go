package axiom_test

import (
	"os"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/internal/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionAdapterReportsOnlyFinalAttempts(t *testing.T) {
	submitted := axiom.Attempt{
		RunID:  "original",
		Number: 2,
		Decision: axiom.Decision{
			Kind:   axiom.DecisionSkip,
			Reason: "disabled",
		},
	}
	var received []axiom.Attempt
	cfg := &axiom.Config{
		Attempt: submitted,
		Runtime: axiom.NewRuntime(axiom.WithRuntimeAttemptSink(func(attempt axiom.Attempt) {
			received = append(received, attempt)
		})),
	}
	spec := cfg.Spec(func(*axiom.Config) {})
	cfg.Attempt = axiom.Attempt{RunID: "changed", Number: 9}

	for _, kind := range []execution.EventType{
		execution.AttemptStarted, execution.AttemptExecutionFinished,
		execution.AttemptPanicRecovered,
	} {
		spec.Observe(execution.Event{Type: kind, Message: "recovered panic"})
	}
	assert.Empty(t, received, "lifecycle facts are not terminal attempts")

	started := time.Now()
	ended := started.Add(time.Second)
	input := execution.Result{
		Kind: execution.ResultCompleted,
		Outcome: &execution.Outcome{
			Status:    execution.AttemptFailed,
			TestName:  "Test/attempt",
			StartedAt: started,
			CleanupAt: ended,
		},
	}
	spec.Complete(input)
	require.Len(t, received, 1)
	got := received[0]
	assert.Equal(t, submitted.RunID, got.RunID)
	assert.Equal(t, submitted.Number, got.Number)
	assert.Equal(t, submitted.Decision, got.Decision)
	assert.Equal(t, &axiom.AttemptOutcome{
		Status:    axiom.AttemptFailed,
		TestName:  "Test/attempt",
		StartedAt: started,
		CleanupAt: ended,
	}, got.Outcome)
	assert.Equal(t, got, cfg.Attempt, "Config must expose the terminal attempt before sinks run")

	input.Outcome.Status = execution.AttemptPassed
	assert.Equal(t, axiom.AttemptFailed, cfg.Attempt.Outcome.Status)
}

func TestExecutionAdapterPreservesSelectionDecision(t *testing.T) {
	for _, tc := range []struct {
		name     string
		result   execution.ResultKind
		decision axiom.DecisionKind
	}{
		{name: "excluded", result: execution.ResultExcluded, decision: axiom.DecisionExclude},
		{name: "not selected", result: execution.ResultNotSelected, decision: axiom.DecisionRun},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var received []axiom.Attempt
			cfg := &axiom.Config{
				Attempt: axiom.Attempt{Decision: axiom.Decision{Kind: tc.decision}},
				Runtime: axiom.NewRuntime(axiom.WithRuntimeAttemptSink(func(attempt axiom.Attempt) {
					received = append(received, attempt)
				})),
			}
			spec := cfg.Spec(func(*axiom.Config) {})
			spec.Complete(execution.Result{Kind: tc.result})
			require.Len(t, received, 1)
			assert.Equal(t, tc.decision, received[0].Decision.Kind)
			assert.Nil(t, received[0].Outcome)
		})
	}
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

	var pluginConfigs []*axiom.Config
	var actionAttempts []axiom.Attempt
	runnerOptions := []axiom.RunnerOption{
		axiom.WithRunnerRetry(axiom.WithRetryTimes(3)),
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			pluginConfigs = append(pluginConfigs, cfg)
		}),
	}
	if os.Getenv("AXIOM_EXECUTION_PARALLEL") == "1" {
		runnerOptions = append(runnerOptions, axiom.WithRunnerParallel(axiom.WithParallelEnabled()))
	}

	t.Cleanup(func() {
		require.Len(t, pluginConfigs, 3, "planning config and two attempt configs")
		require.Len(t, actionAttempts, 2)
		runID := pluginConfigs[0].Attempt.RunID
		require.NotEmpty(t, runID)
		matches := true
		for number, cfg := range pluginConfigs {
			matches = assert.Equal(t, runID, cfg.Attempt.RunID) && matches
			matches = assert.Equal(t, number, cfg.Attempt.Number) && matches
			matches = assert.Equal(t, axiom.DecisionRun, cfg.Attempt.Decision.Kind) && matches
		}
		for i, attempt := range actionAttempts {
			prepared := pluginConfigs[i+1].Attempt
			matches = assert.Equal(t, prepared.RunID, attempt.RunID) && matches
			matches = assert.Equal(t, prepared.Number, attempt.Number) && matches
			matches = assert.Equal(t, prepared.Decision, attempt.Decision) && matches
			matches = assert.Nil(t, attempt.Outcome) && matches
		}
		if matches {
			t.Log("execution retry identity verified")
		}
	})

	axiom.NewRunner(runnerOptions...).RunCase(t, axiom.NewCase(), func(cfg *axiom.Config) {
		actionAttempts = append(actionAttempts, cfg.Attempt)
		if cfg.Attempt.Number == 1 {
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
				attempt axiom.Attempt
				event   axiom.Event
			}
			var events []observedEvent
			var pluginInstalls, actions int
			runnerOptions := []axiom.RunnerOption{
				axiom.WithRunnerRetry(axiom.WithRetryTimes(3)),
				axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
					pluginInstalls++
					if cfg.Attempt.Number == 1 {
						cfg.Skip = axiom.NewSkip(axiom.SkipBecause("plugin decision"))
					}
					cfg.Runtime.EmitEventSink(func(event axiom.Event) {
						events = append(events, observedEvent{attempt: cfg.Attempt, event: event})
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
			assert.Equal(t, 1, events[0].attempt.Number)
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
		{name: "parallel retry", parallel: true, attempt: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var events []observed
			runner := axiom.NewRunner(
				axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
				axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
					cfg.Runtime.EmitEventSink(func(e axiom.Event) {
						events = append(events, observed{attempt: cfg.Attempt.Number, event: e})
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

	assert.Equal(t, []string{"fixture", "test", "wrap", "observer", "wrap", "observer"}, order)
}
