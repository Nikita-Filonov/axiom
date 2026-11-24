package execution

import "time"

// Plan describes a logical run. MaxAttempts must be at least one and
// RetryDelay non-negative. Prepare must supply fresh callbacks for each attempt;
// retries never reuse a previous attempt's mutable state. Parallel controls the
// container for parallel retries; Spec.Parallel controls individual attempts.
type Plan struct {
	Name        string
	Parallel    bool
	RetryDelay  time.Duration
	MaxAttempts int

	Prepare func(PrepareContext) Spec
}

// PrepareContext describes the attempt the engine asks the caller to prepare.
// It is passed by value and contains no mutable execution state.
type PrepareContext struct {
	// Number is the one-based attempt number.
	Number int
}
