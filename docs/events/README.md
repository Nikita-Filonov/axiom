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

Lifecycle events are represented by their type:

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
- `log`, `assert`, `artefact`

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
