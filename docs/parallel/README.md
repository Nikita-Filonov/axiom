# 📘 Parallel

`Parallel` controls whether a test runs in Go’s parallel mode. `Parallel` settings may be defined at `Runner`, `Case`,
and registered `Suite` test level. Case-level settings override Runner-level settings for case execution.

Parallelism affects only test _scheduling_ and is fully compatible with metadata, fixtures, hooks, plugins, and retry
logic.

A `Parallel` flag does **not** modify execution ordering inside a test — it only influences how Go schedules test cases
relative to each other.

This model enables:

- explicit opt-in parallel execution
- predictable merging (`Case` > `Runner`)
- consistent behavior across retries and subtests
- parallel suite tests without sharing mutable suite runtime state

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
		axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
	)

	// -------------------------------------------------------------------------
	// Case overrides Runner setting
	// -------------------------------------------------------------------------

	c := axiom.NewCase(
		axiom.WithCaseName("runs sequentially"),
		axiom.WithCaseParallel(axiom.WithParallelDisabled()),
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

---

## Parallel With Retry

When a Case enables both Parallel and Retry, Axiom applies `t.Parallel()` once to the Case wrapper. Retry attempts then
run sequentially inside that wrapper, so Axiom receives the result of one attempt before deciding whether to start the
next:

```text
parallel case
├── attempt 1
├── attempt 2
└── attempt 3
```

Different parallel Cases can overlap, while hooks, plugins, Config, local fixtures, and fixture cleanup remain scoped to
each concrete attempt.

A failure reported to Go's `testing.T` cannot be erased. A later successful attempt stops further retries, but the
earlier failure remains part of the final test result.

---

## Suite-Level Parallelism

Suite-level parallelism is intentionally separate from Runner/Case parallelism.

Use `NewSuiteFactory` with `WithSuiteConfigParallel` when every registered suite test should run in parallel:

```go
func TestUsersSuite(t *testing.T) {
	suite := axiom.NewSuiteFactory(
		t,
		func() *UsersSuite { return new(UsersSuite) },
		axiom.WithSuiteConfigParallel(),
	)

	suite.Test("user can log in", (*UsersSuite).UserCanLogin)
	suite.Test("admin can block user", (*UsersSuite).AdminCanBlockUser)
	suite.Run()
}
```

Use `WithSuiteTestParallel` when only one registered suite test should run in parallel:

```go
suite.Test(
	"user can log in",
	(*UsersSuite).UserCanLogin,
	axiom.WithSuiteTestParallel(),
)
```

Parallel suite tests require `NewSuiteFactory`. The regular `NewSuite` constructor uses one suite instance and stays
sequential so `SubT`, `Runner`, and any suite fields are not shared across parallel suite methods by accident.

Hooks keep the same lifecycle:

- `BeforeAll` and `AfterAll` run once per runner
- `BeforeTest` and `AfterTest` run for each case attempt
- test-level hooks may run concurrently when suite tests or cases are parallel
- runner resources shared by parallel tests must be safe for concurrent use
