package teststats

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuns_ClassifyAttemptHistory(t *testing.T) {
	for _, tc := range []struct {
		statuses []Status
		want     Status
	}{
		{[]Status{"passed"}, "passed"},
		{[]Status{"skipped"}, "skipped"},
		{[]Status{"failed"}, "failed"},
		{[]Status{"failed", "passed"}, "flaky"},
		{[]Status{"failed", "failed"}, "failed"},
		{[]Status{"failed", "skipped"}, "failed"},
		{[]Status{"passed", "failed"}, "failed"},
	} {
		s := NewStats()
		for i, status := range tc.statuses {
			s.add(Attempt{RunID: "run", Number: i + 1, Status: status})
		}

		runs := s.Runs()
		require.Len(t, runs, 1)
		assert.Equal(t, tc.want, runs[0].Status, "%v", tc.statuses)
		assert.Len(t, runs[0].Attempts, len(tc.statuses))
	}
}

func TestRuns_OrderTimingAndIdentity(t *testing.T) {
	start := time.Now()
	s := NewStats()
	// Numbers and timestamps are deliberately out of order: status follows the
	// attempt number, while the bounds use the actual earliest and latest times.
	s.add(Attempt{
		RunID: "run", CaseID: "C-1", Name: "latest", Number: 2, Status: StatusPassed,
		Start: start, End: start.Add(time.Second), Duration: time.Second,
		Meta: axiom.Meta{Tags: []string{"latest"}},
	})
	s.add(Attempt{
		RunID: "run", CaseID: "C-1", Name: "first", Number: 1, Status: StatusFailed,
		Start: start.Add(2 * time.Second), End: start.Add(3 * time.Second), Duration: time.Second,
	})
	s.add(Attempt{RunID: "other", Number: 1, Status: StatusPassed, Start: start.Add(4 * time.Second)})

	runs := s.Runs()
	require.Len(t, runs, 2)
	run := runs[0]
	assert.Equal(t, "run", run.ID)
	assert.Equal(t, "C-1", run.CaseID)
	assert.Equal(t, "latest", run.Name)
	assert.Equal(t, []string{"latest"}, run.Meta.Tags)
	assert.Equal(t, StatusFlaky, run.Status)
	assert.Equal(t, start, run.Start)
	assert.Equal(t, start.Add(3*time.Second), run.End)
	assert.Equal(t, 2*time.Second, run.Duration)
	assert.Equal(t, 3*time.Second, run.Elapsed)
	assert.Equal(t, []int{1, 2}, []int{run.Attempts[0].Number, run.Attempts[1].Number})
	assert.Equal(t, "other", runs[1].ID)

	run.Meta.Tags[0] = "changed"
	assert.Equal(t, []string{"latest"}, s.Runs()[0].Meta.Tags)
}

func TestRuns_ElapsedIncludesTimeBetweenAttempts(t *testing.T) {
	start := time.Date(2026, time.October, 1, 10, 0, 0, 0, time.UTC)
	s := NewStats()
	s.add(Attempt{
		RunID: "run", Number: 1, Status: StatusFailed,
		Start: start, End: start.Add(time.Second), Duration: time.Second,
	})
	s.add(Attempt{
		RunID: "run", Number: 2, Status: StatusPassed,
		Start: start.Add(3 * time.Second), End: start.Add(4 * time.Second), Duration: time.Second,
	})

	runs := s.Runs()
	require.Len(t, runs, 1)
	assert.Equal(t, StatusFlaky, runs[0].Status)
	assert.Equal(t, start.Add(4*time.Second), runs[0].End)
	assert.Equal(t, 2*time.Second, runs[0].Duration)
	assert.Equal(t, 4*time.Second, runs[0].Elapsed)
}

func TestRuns_AttemptsWithoutRunIDAreSeparateRuns(t *testing.T) {
	s := NewStats()
	s.add(Attempt{Status: StatusFailed})
	s.add(Attempt{Status: StatusPassed})

	runs := s.Runs()
	require.Len(t, runs, 2)
	assert.Equal(t, StatusFailed, runs[0].Status)
	assert.Equal(t, StatusPassed, runs[1].Status)
}
