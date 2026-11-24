package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTerminalEventType(t *testing.T) {
	for kind, eventType := range map[ResultKind]EventType{
		ResultCompleted:   AttemptFinished,
		ResultExcluded:    AttemptExcluded,
		ResultNotSelected: AttemptNotSelected,
	} {
		assert.Equal(t, eventType, terminalEventType(kind))
	}
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
