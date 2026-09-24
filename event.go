package axiom

import (
	"fmt"
	"time"
)

// EventType identifies a raw lifecycle or emitted fact in Axiom's event stream.
type EventType string

const (
	EventTypeRunnerBeforeAllStart  EventType = "runner.before-all.start"
	EventTypeRunnerBeforeAllFinish EventType = "runner.before-all.finish"
	EventTypeRunnerBeforeAllPanic  EventType = "runner.before-all.panic"
	EventTypeRunnerAfterAllStart   EventType = "runner.after-all.start"
	EventTypeRunnerAfterAllFinish  EventType = "runner.after-all.finish"
	EventTypeRunnerAfterAllPanic   EventType = "runner.after-all.panic"

	EventTypeCaseStart  EventType = "case.start"
	EventTypeCaseFinish EventType = "case.finish"
	EventTypeCasePanic  EventType = "case.panic"

	EventTypeStepStart      EventType = "step.start"
	EventTypeStepFinish     EventType = "step.finish"
	EventTypeStepPanic      EventType = "step.panic"
	EventTypeSetupStart     EventType = "setup.start"
	EventTypeSetupFinish    EventType = "setup.finish"
	EventTypeSetupPanic     EventType = "setup.panic"
	EventTypeTeardownStart  EventType = "teardown.start"
	EventTypeTeardownFinish EventType = "teardown.finish"
	EventTypeTeardownPanic  EventType = "teardown.panic"

	EventTypeFixtureSetupStart     EventType = "fixture.setup.start"
	EventTypeFixtureSetupFinish    EventType = "fixture.setup.finish"
	EventTypeFixtureSetupFailed    EventType = "fixture.setup.failed"
	EventTypeFixtureCleanupStart   EventType = "fixture.cleanup.start"
	EventTypeFixtureCleanupFinish  EventType = "fixture.cleanup.finish"
	EventTypeFixtureCleanupPanic   EventType = "fixture.cleanup.panic"
	EventTypeResourceSetupStart    EventType = "resource.setup.start"
	EventTypeResourceSetupFinish   EventType = "resource.setup.finish"
	EventTypeResourceSetupFailed   EventType = "resource.setup.failed"
	EventTypeResourceCleanupStart  EventType = "resource.cleanup.start"
	EventTypeResourceCleanupFinish EventType = "resource.cleanup.finish"
	EventTypeResourceCleanupPanic  EventType = "resource.cleanup.panic"

	EventTypeLog      EventType = "log"
	EventTypeAssert   EventType = "assert"
	EventTypeArtefact EventType = "artefact"
)

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

// EventOption configures an Event before normalization.
type EventOption func(*Event)

// NewEvent returns an Event of eventType, assigning the current time unless
// an option supplies one.
func NewEvent(eventType EventType, options ...EventOption) Event {
	e := Event{Type: eventType}
	for _, option := range options {
		option(&e)
	}

	e.Normalize()
	return e
}

// WithEventTime sets the event timestamp. An empty value is replaced with the
// current time when the Event is normalized.
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

func NewLogEvent(l Log) Event {
	return NewEvent(
		EventTypeLog,
		WithEventName(l.Level.String()),
		WithEventMessage(l.Text),
	)
}

func NewAssertEvent(a Assert) Event {
	return NewEvent(
		EventTypeAssert,
		WithEventName(a.Type.String()),
		WithEventMessage(a.Message),
	)
}

func NewArtefactEvent(a Artefact) Event {
	return NewEvent(
		EventTypeArtefact,
		WithEventName(a.Type.String()),
		WithEventMessage(a.Name),
	)
}

// Normalize fills an empty Time with the current RFC3339Nano timestamp.
func (e *Event) Normalize() {
	if e.Time == "" {
		e.Time = time.Now().Format(time.RFC3339Nano)
	}
}
