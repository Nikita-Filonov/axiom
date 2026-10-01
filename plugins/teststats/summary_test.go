package teststats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSummary_CountsRunsAndAttempts(t *testing.T) {
	s := NewStats()
	for _, a := range []Attempt{
		{RunID: "flaky", Number: 1, Status: StatusFailed},
		{RunID: "flaky", Number: 2, Status: StatusPassed},
		{RunID: "failed", Number: 1, Status: StatusFailed},
		{RunID: "passed", Number: 1, Status: StatusPassed},
		{RunID: "skipped", Number: 1, Status: StatusSkipped},
	} {
		s.add(a)
	}

	assert.Equal(t, Summary{
		Runs:     Counts{Total: 4, Passed: 1, Failed: 1, Skipped: 1, Flaky: 1},
		Attempts: Counts{Total: 5, Passed: 2, Failed: 2, Skipped: 1},
	}, s.Summary())
}
