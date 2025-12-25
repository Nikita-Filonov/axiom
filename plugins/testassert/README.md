# ✅ Assert Plugin (`testassert`)

Provides assertion handling for Axiom tests by bridging Axiom’s structured assertion events with
`stretchr/testify/assert`.

The plugin consumes assertions emitted via the Axiom runtime assertion pipeline and executes the corresponding
`testify/assert` calls against the active `*testing.T`.

This allows test code and plugins to emit **declarative**, **structured assertions** without directly depending on a
specific assertion library.

---

## Overview

Axiom assertions are represented as runtime events (`axiom.Assert`) and emitted during test execution. The `testassert`
plugin acts as an assertion sink, translating these events into concrete `testify/assert` invocations.

Key properties:

- assertions are handled **only at runtime**
- assertion logic is **decoupled from test code**
- assertion backend can be replaced or extended via plugins

---

## Supported Assertions

The plugin currently supports the following assertion types:

- `AssertEqual`
- `AssertTrue`
- `AssertFalse`
- `AssertError`
- `AssertNoError`
- `AssertNil`
- `AssertNotNil`

Each assertion includes:

- assertion type
- expected / actual values (where applicable)
- optional message

---

## Example

```go
package example_test

import (
	"errors"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testassert"
)

func TestAssertExample(t *testing.T) {

	// Enable assertion handling via testify/assert
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testassert.Plugin(),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("assert example"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		cfg.Step("validate values", func() {

			// Emit an equality assertion into the runtime
			cfg.Assert(
				axiom.NewEqualAssert(42, 42, "values must match"),
			)

			// Emit boolean assertions
			cfg.Assert(
				axiom.NewTrueAssert(true, "condition must be true"),
			)
			cfg.Assert(
				axiom.NewFalseAssert(false, "condition must be false"),
			)
		})

		cfg.Step("validate errors", func() {

			err := errors.New("boom")

			// Assert that an error is present
			cfg.Assert(
				axiom.NewErrorAssert(err, "error must be present"),
			)

			// Assert that no error occurred
			cfg.Assert(
				axiom.NewNoErrorAssert(nil, "no error expected"),
			)
		})

		cfg.Step("validate nils", func() {

			var v any = nil
			obj := struct{}{}

			// Assert nil / not-nil values
			cfg.Assert(
				axiom.NewNilAssert(v, "value must be nil"),
			)
			cfg.Assert(
				axiom.NewNotNilAssert(obj, "object must not be nil"),
			)
		})
	})
}

```
