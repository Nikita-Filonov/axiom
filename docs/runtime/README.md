# 📘 Runtime

A `Runtime` defines how tests, steps, logs, and artefacts are executed and intercepted at runtime.

Unlike `Runner` and `Case`, which are declarative, a `Runtime` is purely behavioral. It contains execution middleware (
wraps) and sinks that observe or instrument test execution.

A `Runtime` is not executed directly. It is merged from `Runner` and `Case`, then invoked indirectly via `Config` during
test execution.

## What Runtime Controls

A `Runtime` provides four extension points:

| Capability        | Description                                             |
|-------------------|---------------------------------------------------------|
| **TestWraps**     | Middleware around the entire test execution             |
| **StepWraps**     | Middleware around each `cfg.Step(...)`                  |
| **LogSinks**      | Receivers of structured logs emitted via `cfg.Log(...)` |
| **ArtefactSinks** | Receivers of artefacts emitted via `cfg.Artefact(...)`  |

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
cfg.Test / cfg.Step / cfg.Log / cfg.Artefact
```

- `Runner` runtime is applied first
- `Case` runtime is applied after
- wraps are executed outer → inner
- sinks are invoked in registration order

## Defining Runtime Behavior

`Runtime` behavior is configured using `RuntimeOptions`:

```go
type RuntimeOption func (*Runtime)
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

	axiom.WithRunnerRuntime(

		// Wrap every test
		axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				fmt.Println("[runtime] before test")
				next(c)
				fmt.Println("[runtime] after test")
			}
		}),

		// Wrap every step
		axiom.WithRuntimeStepWrap(func(name string, next axiom.StepAction) axiom.StepAction {
			return func() {
				fmt.Println("[runtime] step:", name)
				next()
			}
		}),

		// Collect logs
		axiom.WithRuntimeLogSink(func(l axiom.Log) {
			fmt.Println("[log]", l.Level, l.Text)
		}),

		// Collect artefacts
		axiom.WithRuntimeArtefactSink(func(a axiom.Artefact) {
			fmt.Println("[artefact]", a.Type, a.Name)
		}),
	),
)

func TestRuntimeExample(t *testing.T) {

	c := axiom.NewCase(
		axiom.WithCaseName("runtime demo"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Log(axiom.NewInfoLog("starting test"))

		cfg.Step("do work", func() {
			cfg.Artefact(
				axiom.NewTextArtefact("payload", "hello world"),
			)
		})
	})
}

```
