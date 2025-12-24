# 📘 Assert

`Assert` represents a **structured assertion event** emitted during test execution.

Assertions in Axiom are **declarative and observational**: they describe _what was asserted_, not
_how the assertion was evaluated_ or _how test execution should react_.

An `Assert` **does not perform a check by itself** and **does not control execution flow**. It is emitted as a
**runtime event** and consumed by **assert sinks** (reporters, statistics collectors, external systems).

Assertions are emitted via `cfg.Assert(...)` and routed through `Runtime` assert sinks.

## Key Principles

- `Assert` is **data**, not logic
- `Assert` does **not fail a test**
- `Assert` does **not trigger retries**
- `Assert` does **not depend on** `testing.T`
- Interpretation is performed by **helpers**, **plugins**, or **sinks**

This design allows Axiom to integrate with existing assertion libraries while providing a **unified**,
**structured assertion event stream**.

## Example

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

func TestAssertExample(t *testing.T) {

	// Runner defines the global execution environment.
	// Here we configure a runtime assert sink that will
	// receive all emitted assertion events.
	runner := axiom.NewRunner(
		axiom.WithRunnerRuntime(

			// Simple assert sink.
			//
			// Assert sinks observe assertion events and can:
			//   - report results
			//   - collect statistics
			//   - forward data to external systems
			//
			// They do NOT control execution flow.
			axiom.WithRuntimeAssertSink(func(a axiom.Assert) {
				fmt.Println(
					"assert:",
					"type:", a.Type,
					"message:", a.Message,
					"expected:", a.Expected,
					"actual:", a.Actual,
					"error:", a.Error,
				)
			}),
		),
	)

	// Case defines a single test case configuration.
	c := axiom.NewCase(
		axiom.WithCaseName("assert demo"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		// Steps provide structured execution and reporting.
		cfg.Step("validate response", func() {

			// Emit an equality assertion event.
			//
			// This does NOT perform a comparison and does NOT fail the test.
			// It only describes what was asserted.
			cfg.Assert(
				axiom.NewEqualAssert(200, 200, "status code"),
			)

			// Emit a "no error expected" assertion.
			//
			// Whether this assertion passes or fails is determined
			// by helpers, plugins, or sinks — not by Axiom core.
			cfg.Assert(
				axiom.NewNoErrorAssert(nil, "request execution"),
			)

			// Emit a non-nil assertion.
			//
			// This assertion describes an invariant:
			// the response payload is expected to exist.
			cfg.Assert(
				axiom.NewNotNilAssert("payload", "response payload"),
			)
		})
	})
}

```
