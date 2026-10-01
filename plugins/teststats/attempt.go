package teststats

import (
	"time"

	"github.com/Nikita-Filonov/axiom"
)

// Status is the outcome of an attempt or a run.
type Status string

// Attempts are passed, failed, or skipped; flaky is derived only for runs.
const (
	StatusPassed  Status = "passed"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
	StatusFlaky   Status = "flaky"
)

// Attempt is the outcome of one case attempt.
type Attempt struct {
	// RunID groups the attempts of one RunCase invocation. An attempt without
	// a RunID is a run of its own.
	RunID string
	// Number is the one-based attempt number within the run.
	Number int
	// CaseID and Name come from the declared Case; TestName is the full Go
	// test name of the attempt.
	CaseID   string
	Name     string
	TestName string
	// Meta is the merged metadata when the attempt started.
	Meta axiom.Meta

	Status Status
	// SkipReason is set for a policy skip. Go does not expose the reason
	// passed directly to t.Skip.
	SkipReason string
	// Error is the first Axiom lifecycle failure: a panic in the test body, a
	// step, setup, or teardown, or a fixture failure. Go does not expose the
	// messages passed to t.Error or t.Fatal.
	Error string

	// Start is when the attempt began executing, after any wait for parallel
	// scheduling. End follows hooks, fixture cleanup, child subtests, and
	// t.Cleanup callbacks.
	Start    time.Time
	End      time.Time
	Duration time.Duration
}

func (a Attempt) clone() Attempt {
	a.Meta = a.Meta.Copy()
	return a
}
