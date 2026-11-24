package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttemptFinalizeIgnoresActiveCallback(t *testing.T) {
	completed := false
	a := &attempt{
		t: &testing.T{},
		spec: Spec{Complete: func(Result) {
			completed = true
		}},
	}

	a.finalize()

	_, ready := a.result()
	assert.False(t, ready)
	assert.False(t, completed)
}

func TestAttemptFinalizeWaitsForCleanup(t *testing.T) {
	var completed []Result
	t.Run("parent", func(parent *testing.T) {
		var native *testing.T
		a := &attempt{spec: Spec{
			Name:     "attempt",
			Decision: Decision{Kind: DecisionRun},
			Bind:     func(t *testing.T) { native = t },
			Complete: func(result Result) { completed = append(completed, result) },
		}}
		a.spec.Body = func() {
			native.Cleanup(func() {
				a.finalize()
				_, ready := a.result()
				assert.False(native, ready, "cleanup is still running")
				assert.Empty(native, completed)
			})
		}
		a.run(parent)
	})

	require.Len(t, completed, 1, "deferred completion must still be delivered once")
	require.NotNil(t, completed[0].Outcome)
	assert.Equal(t, AttemptPassed, completed[0].Outcome.Status)
	assert.False(t, completed[0].Outcome.CleanupAt.IsZero())
}

func TestAttemptShouldRetry(t *testing.T) {
	for _, tc := range []struct {
		name     string
		want     bool
		result   *Result
		decision DecisionKind
	}{
		{
			name:     "pending",
			decision: DecisionRun,
		},
		{
			name: "passed",
			result: &Result{
				Kind:    ResultCompleted,
				Outcome: &Outcome{Status: AttemptPassed},
			},
			decision: DecisionRun,
		},
		{
			name: "failed",
			want: true,
			result: &Result{
				Kind:    ResultCompleted,
				Outcome: &Outcome{Status: AttemptFailed},
			},
			decision: DecisionRun,
		},
		{
			name: "runtime skip",
			result: &Result{
				Kind:    ResultCompleted,
				Outcome: &Outcome{Status: AttemptSkipped},
			},
			decision: DecisionRun,
		},
		{
			name: "policy skip with cleanup failure",
			result: &Result{
				Kind:    ResultCompleted,
				Outcome: &Outcome{Status: AttemptFailed},
			},
			decision: DecisionSkip,
		},
		{
			name:     "excluded",
			result:   &Result{Kind: ResultExcluded},
			decision: DecisionExclude,
		},
		{
			name:     "native deselection",
			result:   &Result{Kind: ResultNotSelected},
			decision: DecisionRun,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &attempt{
				spec:        Spec{Decision: Decision{Kind: tc.decision}},
				finalResult: tc.result,
			}

			assert.Equal(t, tc.want, a.shouldRetry())
		})
	}
}
