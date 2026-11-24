package execution

import "time"

// Status is the final native test status. Failure takes precedence over skip.
type Status string

const (
	// AttemptPassed means the native test neither failed nor skipped.
	AttemptPassed Status = "passed"

	// AttemptFailed means the native test failed, even if it also skipped.
	AttemptFailed Status = "failed"

	// AttemptSkipped means the native test skipped without a failure.
	AttemptSkipped Status = "skipped"
)

// Outcome is observed after the native subtest has finished. CleanupAt
// is sampled in the engine's earliest-registered cleanup, after callbacks
// registered by wrappers, hooks, the body, and child tests. Final delivery may
// occur later at the parent boundary; that delay is not test duration.
type Outcome struct {
	Status    Status
	TestName  string
	StartedAt time.Time
	CleanupAt time.Time
}
