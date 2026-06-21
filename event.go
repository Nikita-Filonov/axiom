package axiom

import (
	"fmt"
	"time"
)

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

type Event struct {
	Time    string    `json:"time,omitempty"`
	Name    string    `json:"name,omitempty"`
	Type    EventType `json:"type"`
	Message string    `json:"message,omitempty"`
}

type EventOption func(*Event)

func NewEvent(eventType EventType, options ...EventOption) Event {
	e := Event{Type: eventType}
	for _, option := range options {
		option(&e)
	}

	e.Normalize()
	return e
}

func WithEventTime(t string) EventOption {
	return func(e *Event) { e.Time = t }
}

func WithEventName(name string) EventOption {
	return func(e *Event) { e.Name = name }
}

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

func (e *Event) Normalize() {
	if e.Time == "" {
		e.Time = time.Now().Format(time.RFC3339Nano)
	}
}
