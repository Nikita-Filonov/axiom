package testjunit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecorderConfirmsPublishedAttempt(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status axiom.AttemptStatus
		failed bool
		want   status
	}{
		{
			name:   "passed",
			status: axiom.AttemptPassed,
			want:   statusPassed,
		},
		{
			name:   "skipped",
			status: axiom.AttemptSkipped,
			want:   statusSkipped,
		},
		{
			name:   "late failure",
			status: axiom.AttemptFailed,
			want:   statusFailed,
		},
		{
			name:   "lifecycle error retained",
			status: axiom.AttemptPassed,
			failed: true,
			want:   statusFailed,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "report.xml")
			reporter := NewReporter()
			Plugin(WithReporter(reporter), WithOutputFile(path))(&axiom.Config{})
			start := time.Now().Add(-time.Second)
			r := &recorder{
				reporter: reporter,
				attempt: &attempt{
					TestName:   "child",
					Start:      start,
					SkipReason: "disabled",
				},
			}
			if tc.failed {
				r.fail("original error")
			}
			r.finish(&fixedOutcome{})
			assert.Equal(t, 1, readOutput(t, path).Tests, "cleanup must publish before the final sink")

			result := axiom.Attempt{
				Outcome: &axiom.AttemptOutcome{
					Status:    tc.status,
					StartedAt: start,
					CleanupAt: start.Add(100 * time.Millisecond),
				},
			}
			r.reconcile(result)
			got := reporter.snapshot()
			require.Len(t, got, 1, "confirmation must update the existing record")
			assert.Equal(t, tc.want, got[0].Status)
			assert.Equal(t, 100*time.Millisecond, got[0].Duration, "parent delivery delay must not inflate duration")
			assert.Equal(t, "disabled", got[0].SkipReason)
			if tc.failed {
				assert.Equal(t, "original error", got[0].Error)
			}
			if tc.want == statusFailed {
				assert.Equal(t, 1, readOutput(t, path).Failures, "late failure must rewrite the file")
			}

			require.NoError(t, os.Remove(path))
			r.reconcile(result)
			r.finish(&fixedOutcome{})
			_, err := os.Stat(path)
			assert.ErrorIs(t, err, os.ErrNotExist, "unchanged confirmation must not publish again")
			assert.Len(t, reporter.snapshot(), 1)
		})
	}
}

func TestRecorderIgnoresResultsWithoutARecord(t *testing.T) {
	r := &recorder{reporter: NewReporter()}
	for _, attempt := range []axiom.Attempt{
		{Decision: axiom.Decision{Kind: axiom.DecisionExclude}},
		{Decision: axiom.Decision{Kind: axiom.DecisionRun}},
		{Outcome: &axiom.AttemptOutcome{Status: axiom.AttemptPassed}},
	} {
		r.reconcile(attempt)
	}
	assert.Empty(t, r.reporter.snapshot())
}
