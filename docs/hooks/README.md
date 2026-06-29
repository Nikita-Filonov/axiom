# 📘 Hooks

`Hooks` provide lifecycle callbacks that run **before and after tests and steps**. They allow you to implement logging,
tracing, reporting, metrics, debug output, dependency injection behaviors, and any other side effects
— **without modifying test code**.

`Hooks` may be defined at both `Runner` and `Case` level. Case-level hooks are appended after `Runner` hooks, forming a
unified ordered execution pipeline.

`Hooks` are intended for observing and extending execution. Axiom enters the corresponding after-phase even when a step
or test body panics, but panics inside hooks still propagate like ordinary test code.

---

## ✔ Available Hooks

### Suite-level hooks

These hooks fire **once per** `Runner`, regardless of the number of executed cases.

| Hook           | When it fires                                   |
|----------------|-------------------------------------------------|
| `BeforeAll(r)` | once, before the first test case in this runner |
| `AfterAll(r)`  | once, after the test function completes         |

Use these for:

- global environment setup (DB containers, mocks, servers)
- global teardown
- test suite–level metrics
- expensive shared resources

**Note:** `AfterAll` runs inside `t.Cleanup`, so it is triggered after the test function completes.

### Runner lifecycle boundary

`BeforeAll` and `AfterAll` are runner-level hooks. They run exactly once per `Runner` via `sync.Once`, but the runner
lifecycle itself must be **bound to something with a deterministic end**, otherwise `AfterAll` may fire earlier than
expected. There are three sound boundaries to choose from.

#### Option 1 — one top-level `TestXxx` with subtests

Use a single Go test function as the boundary. `BeforeAll` fires before any subtest, `AfterAll` fires through
`testing.T.Cleanup` after the test function (and all its subtests) completes.

```go
func TestUsers(t *testing.T) {
    t.Run("can login", func(st *testing.T) { runner.RunCase(st, login, body) })
    t.Run("can logout", func(st *testing.T) { runner.RunCase(st, logout, body) })
}
```

#### Option 2 — `Suite`

Use `axiom.NewSuite(...)` to register related cases under one shared boundary. See `docs/suite` for details.

#### Option 3 — `RunPackage` from `TestMain`

When one runner is shared by many independent top-level `TestXxx` functions in a package, bind its lifecycle to the
test binary itself via `TestMain`. `RunPackage` wires `BeforeAll`, `AfterAll`, and runner-scoped resource cleanups
around `m.Run()`. See [docs/package](../package) for the full reference.

```go
var runner = axiom.NewRunner(
    axiom.WithRunnerResource("db", DBResource),
    axiom.WithRunnerHooks(
        axiom.WithBeforeAll(prepareDB),
        axiom.WithAfterAll(dropDB),
    ),
)

func TestMain(m *testing.M) {
    os.Exit(axiom.RunPackage(m, runner))
}

func TestA(t *testing.T) { runner.RunCase(t, caseA, body) }
func TestB(t *testing.T) { runner.RunCase(t, caseB, body) }
func TestC(t *testing.T) { runner.RunCase(t, caseC, body) }
```

`RunPackage` guarantees:

- `BeforeAll` fires once before `m.Run()` starts (and thus before any `TestXxx`)
- `AfterAll` fires once after `m.Run()` returns — i.e. after the **last** `TestXxx` finishes, not after the first
- runner-scoped resource cleanups run once for the whole package, after `AfterAll`

How this works under the hood: while `entry` runs, the runner is marked as **managed** (the lifecycle is owned by an
outer manager, not by individual `t.Cleanup`s). In that mode every `RunCase` skips its own `t.Cleanup(r.ApplyFinish)`
registration, because otherwise the very first `TestXxx` to call `RunCase` would tear the runner down on its own
cleanup — defeating the whole point of having a package boundary. Once `RunPackage` returns, the flag is cleared and
`RunCase` falls back to its standalone behavior.

For custom harnesses that need to wrap the test entry point with additional behavior (signal handling, coverage
post-processing, etc.), `axiom.RunPackageWith(runner, fn)` accepts any `func() int` and applies the same lifecycle.

> ⚠️ If a `BeforeAll` hook panics, `entry` is not invoked and `AfterAll` does **not** run (the `defer r.ApplyFinish()`
> is registered after `r.ApplyStart()` succeeds). The original panic propagates verbatim. If you need cleanup of
> partially-initialized state, do it inside the failing `BeforeAll` itself via `defer`.

#### Anti-pattern

Without one of the three boundaries above, reusing a package-level `Runner` from multiple unrelated `TestXxx` functions
ties `AfterAll` to whichever `testing.T` happened to call `RunCase` first. When that test completes, the runner tears
down and any subsequent test sees closed resources.

### Test-level hooks

| Hook              | When it fires                        |
|-------------------|--------------------------------------|
| `BeforeTest(cfg)` | right before executing the test body |
| `AfterTest(cfg)`  | after finishing the test body        |

### Step-level hooks

| Hook                    | When it fires                                     |
|-------------------------|---------------------------------------------------|
| `BeforeStep(cfg, name)` | before executing a step                           |
| `AfterStep(cfg, name)`  | after executing a step (always, even if panicked) |

---

## Cleanup Boundary

Framework-owned cleanup is not implemented as user hooks.

This keeps hooks focused on user-defined behavior. Exact cleanup timing is documented by the feature that owns the
cleanup lifecycle.

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
