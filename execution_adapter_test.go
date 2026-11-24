package axiom

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom/internal/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionAdapterReportsOnlyFinalAttempts(t *testing.T) {
	submitted := Attempt{
		RunID:  "original",
		Number: 2,
		Decision: Decision{
			Kind:   DecisionSkip,
			Reason: "disabled",
		},
	}
	var received []Attempt
	cfg := &Config{
		Attempt: submitted,
		Runtime: NewRuntime(WithRuntimeAttemptSink(func(attempt Attempt) {
			received = append(received, attempt)
		})),
	}
	spec := cfg.spec(func(*Config) {})
	cfg.Attempt = Attempt{RunID: "changed", Number: 9}

	for _, kind := range []execution.EventType{
		execution.AttemptStarted, execution.AttemptExecutionFinished,
		execution.AttemptPanicRecovered,
	} {
		spec.Observe(execution.Event{Type: kind, Message: "recovered panic"})
	}
	assert.Empty(t, received, "lifecycle facts are not terminal attempts")

	started := time.Now()
	ended := started.Add(time.Second)
	input := execution.Event{
		Type: execution.AttemptFinished,
		Decision: execution.Decision{
			Kind:   execution.DecisionSkip,
			Reason: "disabled",
		},
		Result: &execution.Result{
			Kind: execution.ResultCompleted,
			Outcome: &execution.Outcome{
				Status:    execution.AttemptFailed,
				TestName:  "Test/attempt",
				StartedAt: started,
				CleanupAt: ended,
			},
		},
	}
	spec.Observe(input)
	require.Len(t, received, 1)
	got := received[0]
	assert.Equal(t, submitted.RunID, got.RunID)
	assert.Equal(t, submitted.Number, got.Number)
	assert.Equal(t, submitted.Decision, got.Decision)
	assert.Equal(t, &AttemptOutcome{
		Status:    AttemptFailed,
		TestName:  "Test/attempt",
		StartedAt: started,
		CleanupAt: ended,
	}, got.Outcome)
	assert.Equal(t, got, cfg.Attempt, "Config must expose the terminal attempt before sinks run")

	input.Result.Outcome.Status = execution.AttemptPassed
	assert.Equal(t, AttemptFailed, cfg.Attempt.Outcome.Status)
}

func TestExecutionAdapterPreservesSelectionDecision(t *testing.T) {
	for _, tc := range []struct {
		name     string
		result   execution.ResultKind
		decision DecisionKind
	}{
		{name: "excluded", result: execution.ResultExcluded, decision: DecisionExclude},
		{name: "not selected", result: execution.ResultNotSelected, decision: DecisionRun},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var received []Attempt
			cfg := &Config{
				Attempt: Attempt{Decision: Decision{Kind: tc.decision}},
				Runtime: NewRuntime(WithRuntimeAttemptSink(func(attempt Attempt) {
					received = append(received, attempt)
				})),
			}
			spec := cfg.spec(func(*Config) {})
			spec.Observe(execution.Event{
				Decision: execution.Decision{Kind: execution.DecisionKind(tc.decision)},
				Result:   &execution.Result{Kind: tc.result},
			})
			require.Len(t, received, 1)
			assert.Equal(t, tc.decision, received[0].Decision.Kind)
			assert.Nil(t, received[0].Outcome)
		})
	}
}

func TestAttemptSinksWaitForNativeCompletion(t *testing.T) {
	var attempts []Attempt
	var cleaned bool
	runner := NewRunner(WithRunnerParallel(WithParallelEnabled()), WithRunnerRuntime(
		WithRuntimeAttemptSink(func(attempt Attempt) {
			assert.True(t, cleaned)
			attempts = append(attempts, attempt)
		}),
	))
	t.Run("parent", func(parent *testing.T) {
		runner.RunCase(parent, NewCase(WithCaseName("parallel")), func(cfg *Config) {
			cfg.T().Run("child", func(child *testing.T) {
				child.Parallel()
				child.Cleanup(func() { cleaned = true })
			})
		})
		assert.Empty(parent, attempts)
	})
	require.Len(t, attempts, 1)
	assert.Equal(t, AttemptPassed, attempts[0].Outcome.Status)
}
