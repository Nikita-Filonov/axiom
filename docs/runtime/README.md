# 📘 Runtime

A `Runtime` defines how tests, steps, logs, artefacts, and raw events are executed and intercepted at runtime.

Unlike `Runner` and `Case`, which are declarative, a `Runtime` is purely behavioral. It contains execution middleware (
wraps) and sinks that observe or instrument test execution.

A `Runtime` is not executed directly. It is merged from `Runner` and `Case`, then invoked indirectly via `Config` during
test execution.

## 📑 Table of Contents

- [What Runtime Controls](#what-runtime-controls)
- [Lifecycle Placement](#lifecycle-placement)
- [Per-attempt lifecycle](#per-attempt-lifecycle)
- [Defining Runtime Behavior](#defining-runtime-behavior)
- [Example](#example)

---

## What Runtime Controls

A `Runtime` provides these extension points:

| Capability         | Description                                                        |
|--------------------|--------------------------------------------------------------------|
| **TestWraps**      | Middleware around the full per-attempt test lifecycle               |
| **StepWraps**      | Middleware around each `cfg.Step(...)`                              |
| **SetupWraps**     | Middleware around each immediate `cfg.Setup(...)` operation         |
| **TeardownWraps**  | Middleware around each immediate `cfg.Teardown(...)` operation      |
| **LogSinks**       | Receivers of structured logs emitted via `cfg.Log(...)`             |
| **EventSinks**     | Receivers of raw events emitted via `cfg.Event(...)`                |
| **AssertSinks**    | Receivers of structured assertions emitted via `cfg.Assert(...)`    |
| **ArtefactSinks**  | Receivers of artefacts emitted via `cfg.Artefact(...)`              |

All runtime behavior is **additive and ordered**. Multiple runtimes (`Runner` + `Case`) are merged deterministically.

## Lifecycle Placement

```text
Runner Runtime
      ↓
Case Runtime
      ↓
Merged Runtime
      ↓
Config
      ↓
cfg.Test / cfg.Step / cfg.Log / cfg.Event / cfg.Artefact
```

- `Runner` runtime is applied first
- `Case` runtime is applied after
- wraps are executed outer → inner
- sinks are invoked in registration order

## Per-attempt lifecycle

`TestWraps` enclose the whole executable lifecycle of one attempt, not only the user action:

```text
case.start event
TestWrap enter
  BeforeTest hooks
  test action
  AfterTest hooks
  fixture cleanups (LIFO)
TestWrap exit
case.finish event
```

This boundary allows a runtime plugin to keep attempt-scoped state active while hooks and fixture cleanups emit logs,
steps, assertions, or artefacts. Reporting plugins can therefore accept final attachments from a fixture cleanup before
they close the test result.

`cfg.Setup(name, fn)` and `cfg.Teardown(name, fn)` are execution primitives, not schedulers. Both run `fn` immediately.
To schedule teardown automatically, return a cleanup from a fixture and optionally decorate that cleanup with
`cfg.Teardown`:

```go
func ReportFixture(cfg *axiom.Config) (any, func(), error) {
	report := NewReport()

	return report, func() {
		cfg.Teardown("finalize report", func() {
			report.Finalize(cfg)
		})
	}, nil
}
```

If the test action or a `BeforeTest` hook panics, Axiom still runs `AfterTest` and fixture cleanups inside the active
wrap, then records the panic as a case failure. Panics raised by `AfterTest` or fixture cleanup also unwind through the
active wraps, but retain their historical behavior and propagate to the caller unchanged. `SkipNow` and `FailNow`
still execute the deferred lifecycle before the attempt goroutine exits.

## Defining Runtime Behavior

`Runtime` behavior is configured using `RuntimeOptions`:

```go
type RuntimeOption func(*Runtime)
```

These options are applied via:

- `axiom.WithRunnerRuntime(...)`
- `axiom.WithCaseRuntime(...)`

## Example

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

var runner = axiom.NewRunner(

	// -------------------------------------------------------------------------
	// Runtime configuration
	// -------------------------------------------------------------------------

	// Runner runtime is copied into every Config built by this runner.
	// Case runtime is merged after it, so case-level behavior can add more wraps
	// and sinks without replacing runner-level behavior.
	axiom.WithRunnerRuntime(

		// Register middleware around every test action.
		// The wrapper runs when cfg.Test(...) executes, not when the runner is built.
		axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				fmt.Println("[runtime] before test")
				next(c)
				fmt.Println("[runtime] after test")
			}
		}),

		// Register middleware around every cfg.Step(name, ...).
		// The step name is passed to the wrapper so reporters can preserve structure.
		axiom.WithRuntimeStepWrap(func(name string, next axiom.StepAction) axiom.StepAction {
			return func() {
				fmt.Println("[runtime] step:", name)
				next()
			}
		}),

		// Log sinks observe structured logs emitted via cfg.Log(...).
		axiom.WithRuntimeLogSink(func(l axiom.Log) {
			fmt.Println("[log]", l.Level, l.Text)
		}),

		// Event sinks observe raw facts emitted via cfg.Event(...) and by Axiom
		// lifecycle helpers such as cfg.Test(...) and cfg.Step(...).
		axiom.WithRuntimeEventSink(func(e axiom.Event) {
			fmt.Println("[event]", e.Type, e.Name, e.Message)
		}),

		// Artefact sinks receive structured test outputs emitted via cfg.Artefact(...).
		axiom.WithRuntimeArtefactSink(func(a axiom.Artefact) {
			fmt.Println("[artefact]", a.Type, a.Name)
		}),
	),
)

func TestRuntimeExample(t *testing.T) {
	// Case options are merged with runner options into a Config for this test.
	c := axiom.NewCase(
		axiom.WithCaseName("runtime demo"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		// This log reaches the log sink above and also emits a raw "log" event.
		cfg.Log(axiom.NewInfoLog("starting test"))

		cfg.Step("do work", func() {
			// This artefact reaches the artefact sink above and also emits a raw
			// "artefact" event.
			cfg.Artefact(
				axiom.NewTextArtefact("payload", "hello world"),
			)
		})
	})
}
```
