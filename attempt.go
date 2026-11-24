package axiom

import (
	"crypto/rand"
	"time"
)

// Attempt describes one prepared test attempt. RunID and Number identify it;
// Decision is fixed before the engine starts it. Outcome remains nil until the
// native test and all of its cleanup have finished. Excluded and unselected
// attempts therefore have no Outcome.
type Attempt struct {
	RunID    string
	Number   int
	Outcome  *AttemptOutcome
	Decision Decision
}

// AttemptStatus is the final native status. Failure takes precedence over skip.
type AttemptStatus string

const (
	// AttemptPassed means the native test neither failed nor skipped.
	AttemptPassed AttemptStatus = "passed"

	// AttemptFailed means the native test failed, even if it also skipped.
	AttemptFailed AttemptStatus = "failed"

	// AttemptSkipped means the native test skipped without a failure.
	AttemptSkipped AttemptStatus = "skipped"
)

// AttemptOutcome describes a completed native test. CleanupAt is sampled after
// callbacks registered by wrappers, hooks, the body, and child tests. Delivery
// may occur later at the parent boundary; that delay is not test duration.
type AttemptOutcome struct {
	Status    AttemptStatus
	TestName  string
	StartedAt time.Time
	CleanupAt time.Time
}

// newAttempt starts a RunCase invocation at its planning attempt.
func newAttempt() Attempt {
	return Attempt{RunID: rand.Text()}
}

// clone isolates the outcome between attempt consumers.
func (a Attempt) clone() Attempt {
	if a.Outcome != nil {
		outcome := *a.Outcome
		a.Outcome = &outcome
	}

	return a
}
