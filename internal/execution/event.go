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

	// AttemptFinished carries a selected attempt's final native outcome.
	AttemptFinished EventType = "attempt.finished"

	// AttemptExcluded records an engine exclusion without a test outcome.
	AttemptExcluded EventType = "attempt.excluded"

	// AttemptNotSelected means Go never entered the subtest callback (e.g. -run).
	AttemptNotSelected EventType = "attempt.not-selected"
)

// Event reports a lifecycle boundary needed by the caller. Decision is the
// submitted instruction. Message describes a recovered panic. Result is present
// only on a terminal event, after native completion.
//
// Observers must be concurrency-safe and must not run test actions or register
// cleanup. Parallel results may arrive after Run returns; the native T has then
// finished. Fatal process termination may prevent terminal delivery.
type Event struct {
	Type     EventType
	Result   *Result
	Message  string
	Decision Decision
}
