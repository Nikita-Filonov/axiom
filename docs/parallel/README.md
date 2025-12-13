# 📘 Parallel

`Parallel` controls whether a test runs in Go’s parallel mode. `Parallel` settings may be defined at both `Runner` and
`Case` level. Case-level settings override Runner-level settings.

Parallelism affects only test _scheduling_ and is fully compatible with metadata, fixtures, hooks, plugins, and retry
logic.

A `Parallel` flag does **not** modify execution ordering inside a test — it only influences how Go schedules test cases
relative to each other.

This model enables:

- explicit opt-in parallel execution
- predictable merging (`Case` > `Runner`)
- consistent behavior across retries and subtests

---

## Example

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

func TestParallelExample(t *testing.T) {

	// -------------------------------------------------------------------------
	// Runner-level parallel (default for all tests)
	// -------------------------------------------------------------------------

	runner := axiom.NewRunner(
		axiom.WithRunnerParallel(), // same as WithParallelEnabled()
	)

	// -------------------------------------------------------------------------
	// Case overrides Runner setting
	// -------------------------------------------------------------------------

	c := axiom.NewCase(
		axiom.WithCaseName("runs sequentially"),
		axiom.WithCaseSequential(), // disables parallel mode for this test
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		// Uses cfg.Parallel.Enabled inside the engine
		fmt.Println("Parallel enabled:", cfg.Parallel.Enabled)

		cfg.Step("work", func() {
			fmt.Println("performing work...")
		})
	})
}

```
