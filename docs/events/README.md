# 📘 Events

---

## Overview

Events are a small raw fact stream emitted by Axiom when something happens that cannot be observed cleanly through hooks
or wraps alone.

The event bus is intentionally dumb:

- it does not calculate final test status
- it does not infer retry attempts
- it does not attach case metadata
- it does not decide which events are important

Consumers decide how to aggregate, filter, or interpret events.

---

## Event Shape

```go
type Event struct {
	// Time is filled by NewEvent unless the caller provides WithEventTime.
	Time    string

	// Name is optional scoped context: step name, fixture name, resource name, etc.
	Name    string

	// Type is the raw fact. Consumers should switch on this value.
	Type    EventType

	// Message is optional payload for failures, panics, logs, asserts, and artefacts.
	Message string
}
```

`Time` is filled automatically when an event is built with `NewEvent`. All other fields are explicit.

---

## Event Types

Events come in two flavours: **lifecycle events** tied to a specific phase, and **fact events** that can be emitted
from any phase.

### Lifecycle events

Lifecycle events follow the `subject.phase.outcome` shape:

- `case.start`, `case.finish`, `case.panic`
- `step.start`, `step.finish`, `step.panic`
- `setup.start`, `setup.finish`, `setup.panic`
- `teardown.start`, `teardown.finish`, `teardown.panic`
- `fixture.setup.start`, `fixture.setup.finish`, `fixture.setup.failed`
- `fixture.cleanup.start`, `fixture.cleanup.finish`, `fixture.cleanup.panic`
- `resource.setup.start`, `resource.setup.finish`, `resource.setup.failed`
- `resource.cleanup.start`, `resource.cleanup.finish`, `resource.cleanup.panic`
- `runner.before-all.start`, `runner.before-all.finish`, `runner.before-all.panic`
- `runner.after-all.start`, `runner.after-all.finish`, `runner.after-all.panic`

### Fact events

Fact events are single data points that can fire from the test body, hooks, fixture factories, steps, or
setup/teardown blocks. They are not bound to a phase, so they stay flat without a subject prefix:

- `log`
- `assert`
- `artefact`

Subscribing code can distinguish the two classes easily:

```go
switch {
case e.Type == axiom.EventTypeLog,
    e.Type == axiom.EventTypeAssert,
    e.Type == axiom.EventTypeArtefact:
    // fact event — happened somewhere inside the test, phase is implicit from
    // the surrounding lifecycle stream

case strings.HasPrefix(string(e.Type), "case."):
    // case lifecycle (start/finish/panic)
}
```

There is no generic `status` field. A failure or panic is its own event.

---

## Emitting Events

```go
// Emit a raw fact: the "create user" step has started.
cfg.Event(axiom.NewEvent(
	axiom.EventTypeStepStart,
	axiom.WithEventName("create user"),
))

// Emit another raw fact: the same step observed a panic-like failure payload.
// Axiom does not calculate final status here; consumers decide what this means.
cfg.Event(axiom.NewEvent(
	axiom.EventTypeStepPanic,
	axiom.WithEventName("create user"),
	axiom.WithEventMessage("database unavailable"),
))
```

Plugins consume events through runtime event sinks:

```go
axiom.WithRuntimeEventSink(func(e axiom.Event) {
	// Runtime sinks receive events exactly as emitted.
	// Keep them, print them, export them, aggregate them, or ignore them.
})
```
