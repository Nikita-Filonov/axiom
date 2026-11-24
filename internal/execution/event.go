package execution

// EventType identifies an engine lifecycle fact. The caller may translate it
// into its own reporting protocol without exposing engine types to consumers.
type EventType string

const (
	// AttemptStarted opens a selected attempt, including a policy skip.
	AttemptStarted EventType = "attempt.started"

	// AttemptPanicRecovered records a panic converted into a native test failure.
	AttemptPanicRecovered EventType = "attempt.panic.recovered"

	// AttemptExecutionFinished closes the executable scope, before native cleanup.
	AttemptExecutionFinished EventType = "attempt.execution.finished"
)

// Event reports a non-terminal lifecycle boundary. Decision is the submitted
// instruction. Message describes a recovered panic.
//
// Observers must be concurrency-safe and must not run test actions or register
// cleanup.
type Event struct {
	Type     EventType
	Message  string
	Decision Decision
}
