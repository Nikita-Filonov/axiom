package axiom

import (
	"fmt"
	"time"
)

// EventType identifies a raw lifecycle or emitted fact in Axiom's event stream.
type EventType string

// EventType values identify lifecycle and emitted facts in the event stream.
const (
	// EventTypeRunnerBeforeAllStart records the start of a runner's BeforeAll phase.
	EventTypeRunnerBeforeAllStart EventType = "runner.before-all.start"
	// EventTypeRunnerBeforeAllFinish records completion of the BeforeAll hooks.
	EventTypeRunnerBeforeAllFinish EventType = "runner.before-all.finish"
	// EventTypeRunnerBeforeAllPanic records a panic during the BeforeAll hooks.
	EventTypeRunnerBeforeAllPanic EventType = "runner.before-all.panic"
	// EventTypeRunnerAfterAllStart records the start of a runner's AfterAll phase.
	EventTypeRunnerAfterAllStart EventType = "runner.after-all.start"
	// EventTypeRunnerAfterAllFinish records completion of AfterAll hooks and resource cleanup.
	EventTypeRunnerAfterAllFinish EventType = "runner.after-all.finish"
	// EventTypeRunnerAfterAllPanic records a panic during AfterAll hooks or resource cleanup.
	EventTypeRunnerAfterAllPanic EventType = "runner.after-all.panic"

	// EventTypeCaseStart records the start of one test case attempt.
	EventTypeCaseStart EventType = "case.start"
	// EventTypeCaseFinish records the end of an attempt, including one with a recovered panic.
	EventTypeCaseFinish EventType = "case.finish"
	// EventTypeCasePanic records a panic recovered while running a case attempt.
	EventTypeCasePanic EventType = "case.panic"
	// EventTypeCaseSkip records a policy skip, which happens instead of the
	// attempt and therefore without case.start or case.finish.
	EventTypeCaseSkip EventType = "case.skip"

	// EventTypeStepStart records entry into a named step, before its hooks run.
	EventTypeStepStart EventType = "step.start"
	// EventTypeStepFinish records the end of a named step, even after a recovered panic.
	EventTypeStepFinish EventType = "step.finish"
	// EventTypeStepPanic records a panic recovered while running a named step.
	EventTypeStepPanic EventType = "step.panic"
	// EventTypeSetupStart records entry into an immediate named setup operation.
	EventTypeSetupStart EventType = "setup.start"
	// EventTypeSetupFinish records the end of a setup operation, even after a recovered panic.
	EventTypeSetupFinish EventType = "setup.finish"
	// EventTypeSetupPanic records a panic recovered during a setup operation.
	EventTypeSetupPanic EventType = "setup.panic"
	// EventTypeTeardownStart records entry into an immediate named teardown operation.
	EventTypeTeardownStart EventType = "teardown.start"
	// EventTypeTeardownFinish records the end of a teardown operation, even after a recovered panic.
	EventTypeTeardownFinish EventType = "teardown.finish"
	// EventTypeTeardownPanic records a panic recovered during a teardown operation.
	EventTypeTeardownPanic EventType = "teardown.panic"

	// EventTypeFixtureSetupStart records the start of fixture construction.
	EventTypeFixtureSetupStart EventType = "fixture.setup.start"
	// EventTypeFixtureSetupFinish records successful fixture construction and caching.
	EventTypeFixtureSetupFinish EventType = "fixture.setup.finish"
	// EventTypeFixtureSetupFailed records a fixture lookup, construction, or type failure.
	// Lookup and cached-value type failures can occur without a setup start event.
	EventTypeFixtureSetupFailed EventType = "fixture.setup.failed"
	// EventTypeFixtureCleanupStart records the start of a fixture cleanup callback.
	EventTypeFixtureCleanupStart EventType = "fixture.cleanup.start"
	// EventTypeFixtureCleanupFinish records successful return from fixture cleanup.
	EventTypeFixtureCleanupFinish EventType = "fixture.cleanup.finish"
	// EventTypeFixtureCleanupPanic records a panic in fixture cleanup before it is rethrown.
	EventTypeFixtureCleanupPanic EventType = "fixture.cleanup.panic"
	// EventTypeResourceSetupStart records the start of resource construction.
	EventTypeResourceSetupStart EventType = "resource.setup.start"
	// EventTypeResourceSetupFinish records successful resource construction and caching.
	EventTypeResourceSetupFinish EventType = "resource.setup.finish"
	// EventTypeResourceSetupFailed records a resource lookup or construction failure.
	// A lookup failure can occur without a setup start event.
	EventTypeResourceSetupFailed EventType = "resource.setup.failed"
	// EventTypeResourceCleanupStart records the start of a resource cleanup callback.
	EventTypeResourceCleanupStart EventType = "resource.cleanup.start"
	// EventTypeResourceCleanupFinish records successful return from resource cleanup.
	EventTypeResourceCleanupFinish EventType = "resource.cleanup.finish"
	// EventTypeResourceCleanupPanic records a panic in resource cleanup before it is rethrown.
	EventTypeResourceCleanupPanic EventType = "resource.cleanup.panic"

	// EventTypeLog records a log entry with its level and text.
	EventTypeLog EventType = "log"
	// EventTypeAssert records an assertion's type and message.
	EventTypeAssert EventType = "assert"
	// EventTypeArtefact records an artefact's type and name.
	EventTypeArtefact EventType = "artefact"
)

// String returns the event type name.
func (t EventType) String() string {
	return string(t)
}

// Event records a raw execution fact for runtime event sinks. It does not
// determine final test status or carry merged Case metadata.
type Event struct {
	// Time is filled by NewEvent in RFC3339Nano format when omitted.
	Time string `json:"time,omitempty"`
	// Name optionally identifies the step, fixture, or resource involved.
	Name string `json:"name,omitempty"`
	// Type identifies the fact or lifecycle transition.
	Type EventType `json:"type"`
	// Message holds optional text associated with the event.
	Message string `json:"message,omitempty"`
}

// EventOption configures an Event built by NewEvent.
type EventOption func(*Event)

// NewEvent returns an Event of eventType, assigning the current time unless
// an option supplies one.
func NewEvent(eventType EventType, options ...EventOption) Event {
	e := Event{Type: eventType}
	for _, option := range options {
		option(&e)
	}

	e.normalize()
	return e
}

// WithEventTime sets the event timestamp. NewEvent replaces an empty value with
// the current time.
func WithEventTime(t string) EventOption {
	return func(e *Event) { e.Time = t }
}

// WithEventName sets the optional step, fixture, or resource name.
func WithEventName(name string) EventOption {
	return func(e *Event) { e.Name = name }
}

// WithEventMessage formats message as text for Event.Message.
func WithEventMessage(message any) EventOption {
	return func(e *Event) { e.Message = fmt.Sprint(message) }
}

// newLogEvent records a structured log as a raw event.
func newLogEvent(l Log) Event {
	return NewEvent(
		EventTypeLog,
		WithEventName(l.Level.String()),
		WithEventMessage(l.Text),
	)
}

// newAssertEvent records an assertion as a raw event.
func newAssertEvent(a Assert) Event {
	return NewEvent(
		EventTypeAssert,
		WithEventName(a.Type.String()),
		WithEventMessage(a.Message),
	)
}

// newArtefactEvent records an artefact as a raw event.
func newArtefactEvent(a Artefact) Event {
	return NewEvent(
		EventTypeArtefact,
		WithEventName(a.Type.String()),
		WithEventMessage(a.Name),
	)
}

// normalize fills an empty Time with the current RFC3339Nano timestamp.
func (e *Event) normalize() {
	if e.Time == "" {
		e.Time = time.Now().Format(time.RFC3339Nano)
	}
}
