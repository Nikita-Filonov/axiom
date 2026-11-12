package teststats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStatsElapsed_ParallelAttemptsUseOuterBounds(t *testing.T) {
	start := time.Date(2026, time.November, 12, 10, 0, 0, 0, time.UTC)
	stats := NewStats()
	// Records from different processes may arrive out of order. The last
	// added record has neither the earliest start nor the latest end.
	for _, attempt := range []Attempt{
		{
			ProcessID: "one",
			RunID:     "short",
			Number:    1,
			Status:    StatusFailed,
			Start:     start.Add(2 * time.Second),
			End:       start.Add(4 * time.Second),
			Duration:  2 * time.Second,
		},
		{
			ProcessID: "two",
			RunID:     "long",
			Number:    1,
			Status:    StatusPassed,
			Start:     start,
			End:       start.Add(10 * time.Second),
			Duration:  10 * time.Second,
		},
		{
			ProcessID: "one",
			RunID:     "skipped",
			Number:    1,
			Status:    StatusSkipped,
			Start:     start.Add(6 * time.Second),
			End:       start.Add(6 * time.Second),
		},
	} {
		stats.add(attempt)
	}

	assert.Equal(t, 10*time.Second, stats.Elapsed(), "overlapping durations must not be summed")
}

func TestStatsElapsed_IncludesRetryAndRunGaps(t *testing.T) {
	start := time.Date(2026, time.November, 12, 10, 0, 0, 0, time.UTC)
	stats := NewStats()
	for _, attempt := range []Attempt{
		{
			RunID:    "retry",
			Number:   1,
			Status:   StatusFailed,
			Start:    start,
			End:      start.Add(time.Second),
			Duration: time.Second,
		},
		{
			RunID:    "retry",
			Number:   2,
			Status:   StatusPassed,
			Start:    start.Add(3 * time.Second),
			End:      start.Add(4 * time.Second),
			Duration: time.Second,
		},
		{
			RunID:    "other",
			Number:   1,
			Status:   StatusPassed,
			Start:    start.Add(7 * time.Second),
			End:      start.Add(8 * time.Second),
			Duration: time.Second,
		},
	} {
		stats.add(attempt)
	}

	assert.Equal(t, 8*time.Second, stats.Elapsed())
	retry := stats.Filter(func(attempt Attempt) bool { return attempt.RunID == "retry" })
	assert.Equal(t, 4*time.Second, retry.Elapsed(), "filtering changes the observed time range")
}
