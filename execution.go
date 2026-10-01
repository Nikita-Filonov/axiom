package axiom

import "crypto/rand"

// Execution identifies the [Runner.RunCase] invocation and attempt a [Config]
// belongs to. Plugins use it to correlate retries without relying on optional
// Case IDs or subtest names.
type Execution struct {
	// ID is shared by every Config built for one RunCase invocation.
	ID string
	// Attempt is the one-based attempt number. It is zero on the planning
	// Config, which decides skip, retry, and parallel policy before attempts run.
	Attempt int
}

// newExecution starts a RunCase invocation with a fresh ID at its planning
// attempt.
func newExecution() Execution {
	return Execution{ID: rand.Text()}
}

// nextAttempt returns the execution of the attempt that follows e.
func (e Execution) nextAttempt() Execution {
	e.Attempt++
	return e
}
