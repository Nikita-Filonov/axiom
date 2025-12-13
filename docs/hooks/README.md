# 📘 Hooks

`Hooks` provide lifecycle callbacks that run **before and after tests and steps**. They allow you to implement logging,
tracing, reporting, metrics, debug output, dependency injection behaviors, and any other side effects
— **without modifying test code**.

`Hooks` may be defined at both `Runner` and `Case` level. Case-level hooks are appended after `Runner` hooks, forming a
unified ordered execution pipeline.

`Hooks` do **not** change control flow — they observe execution. They always fire, even if a step or test panics (Axiom
guarantees this via internal `defer` recovery).

---

## ✔ Available Hooks

### Suite-level hooks

These hooks fire **once per** `Runner`, regardless of the number of executed cases.

| Hook           | When it fires                                   |
|----------------|-------------------------------------------------|
| `BeforeAll(r)` | once, before the first test case in this runner |
| `AfterAll(r)`  | once, after the last test case (via t.Cleanup)  |

Use these for:

- global environment setup (DB containers, mocks, servers)
- global teardown
- test suite–level metrics
- expensive shared resources

**Note:** `AfterAll` runs inside `t.Cleanup`, so it is triggered after the test function completes. This matches Go’s
testing lifecycle and ensures deterministic teardown.

### Test-level hooks

| Hook              | When it fires                                       |
|-------------------|-----------------------------------------------------|
| `BeforeTest(cfg)` | right before executing the test body                |
| `AfterTest(cfg)`  | after finishing the test body (even if it panicked) |

### Step-level hooks

| Hook                    | When it fires                                     |
|-------------------------|---------------------------------------------------|
| `BeforeStep(cfg, name)` | before executing a step                           |
| `AfterStep(cfg, name)`  | after executing a step (always, even if panicked) |

---

## Example

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

// -----------------------------------------------------------------------------
// Hook implementations
// -----------------------------------------------------------------------------

func beforeAll(r *axiom.Runner)            { fmt.Println("→ before all (suite setup)") }
func afterAll(r *axiom.Runner)             { fmt.Println("→ after all (suite teardown)") }
func beforeTest(c *axiom.Config)           { fmt.Println("→ before test") }
func afterTest(c *axiom.Config)            { fmt.Println("→ after test") }
func beforeStep(c *axiom.Config, n string) { fmt.Println("→ before step:", n) }
func afterStep(c *axiom.Config, n string)  { fmt.Println("→ after step:", n) }

// -----------------------------------------------------------------------------
// Runner with global hooks
// -----------------------------------------------------------------------------

var runner = axiom.NewRunner(
	axiom.WithRunnerHooks(
		axiom.WithBeforeAll(beforeAll),
		axiom.WithAfterAll(afterAll),
		axiom.WithBeforeTest(beforeTest),
		axiom.WithAfterTest(afterTest),
		axiom.WithBeforeStep(beforeStep),
		axiom.WithAfterStep(afterStep),
	),
)

// -----------------------------------------------------------------------------
// Test demonstrating hook behavior
// -----------------------------------------------------------------------------

func TestHooksExample(t *testing.T) {

	c := axiom.NewCase(
		axiom.WithCaseName("hooks example"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		cfg.Step("prepare", func() {
			fmt.Println("doing prepare...")
		})

		cfg.Test(func(inner *axiom.Config) {
			fmt.Println("inside test body")
		})

		cfg.Step("finish", func() {
			fmt.Println("finishing...")
		})
	})
}

```

---

## Execution Order Overview

For two test cases inside one runner:

```text
→ BEFORE ALL (suite setup)

Case 1:
  → before test
    → before step prepare
    → after step prepare
    → inside test body
    → before step finish
    → after step finish
  → after test

Case 2:
  → before test
  ...

→ AFTER ALL (suite teardown)
```
