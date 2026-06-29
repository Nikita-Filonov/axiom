# 📘 Package

`RunPackage` is the standard helper for binding a shared `Runner` to the **whole Go test binary** through `TestMain`. It
guarantees that runner-level `BeforeAll`, `AfterAll`, and runner-scoped resource cleanups fire **once per package**, not
once per top-level `TestXxx`.

This is the right boundary when a single `Runner` is reused across many unrelated `TestXxx` functions in the same
package — typically because all of them share the same expensive infrastructure (DB container, gRPC client pool,
external service stub).

---

## Why it exists

Without an explicit lifecycle boundary, a package-level `Runner` is tied to whichever `*testing.T` happened to call
`RunCase` first. When that test completes, its `t.Cleanup` flushes `r.ApplyFinish` through `sync.Once`, and every
subsequent `TestXxx` sees a runner that has already torn down its `AfterAll` and resources.

`RunPackage` solves this by wrapping `m.Run()` with the runner lifecycle directly:

```go
var runner = axiom.NewRunner( /* ... */ )

func TestMain(m *testing.M) {
    os.Exit(axiom.RunPackage(m, runner))
}

func TestA(t *testing.T) { runner.RunCase(t, caseA, body) }
func TestB(t *testing.T) { runner.RunCase(t, caseB, body) }
func TestC(t *testing.T) { runner.RunCase(t, caseC, body) }
```

Effective execution order:

```text
runner BeforeAll
  TestA
  TestB
  TestC
runner AfterAll
  resource cleanups (LIFO)
```

---

## API

```go
func RunPackage(m *testing.M, r *Runner) int
func RunPackageWith(r *Runner, entry func() int) int
```

- `RunPackage(m, r)` — the standard form for `TestMain`. Calls `RunPackageWith(r, m.Run)`.
- `RunPackageWith(r, entry)` — building block for custom test harnesses (signal handling, coverage post-processing,
  benchmarking). Accepts any `func() int` and applies the same lifecycle around it.

### Guarantees

- `BeforeAll` hooks fire **exactly once**, before `entry` starts
- `AfterAll` hooks fire **exactly once**, after `entry` returns or panics
- runner-scoped resource cleanups run **exactly once**, in LIFO order, after `AfterAll`
- the panic from `entry` is **propagated verbatim** after `AfterAll` and resource cleanups complete
- the exit code returned by `entry` is returned verbatim

### Panics

`RunPackage` panics with a clear message in three input-validation cases:

| Call                                | Panic message                       |
|-------------------------------------|-------------------------------------|
| `RunPackage(nil, r)`                | `runpackage: nil *testing.M`        |
| `RunPackageWith(nil, fn)`           | `runpackage: nil *Runner`           |
| `RunPackageWith(r, nil)`            | `runpackage: nil entry function`    |

---

## How it works under the hood

While `entry` runs, the runner is internally marked as **managed** — meaning the lifecycle is owned by an outer
manager (`RunPackageWith` here) rather than by individual `t.Cleanup`s. In that mode every `RunCase` skips its own
`t.Cleanup(r.ApplyFinish)` registration, because otherwise the very first `TestXxx` to call `RunCase` would tear the
runner down on its own cleanup — the exact problem `RunPackage` is meant to solve.

Once `RunPackageWith` returns, the flag is cleared and `RunCase` falls back to its standalone behavior (registering
`r.ApplyFinish` via `t.Cleanup` like before), so calling `RunCase` outside a `TestMain` context continues to work
exactly as it did.

In code terms (`runner.go`):

```go
func (r *Runner) RunCase(t *testing.T, c Case, action TestAction) {
    r.ApplyStart()
    if !r.managed.Load() {
        t.Cleanup(r.ApplyFinish)
    }
    r.runCase(t, c, action)
}
```

And `package.go`:

```go
func RunPackageWith(r *Runner, entry func() int) int {
    r.managed.Store(true)
    defer r.managed.Store(false)

    r.ApplyStart()
    defer r.ApplyFinish()
    return entry()
}
```

---

## Lifecycle edge cases

| Situation                            | Outcome                                                                                |
|--------------------------------------|----------------------------------------------------------------------------------------|
| `entry` returns `0`                  | `AfterAll` runs, resources are torn down, exit code is `0`                             |
| `entry` returns non-zero             | same as above; the non-zero exit code is propagated                                    |
| `entry` panics                       | `AfterAll` and resource cleanups run (via `defer`), then the panic propagates verbatim |
| `AfterAll` itself panics             | the panic propagates after resource cleanups                                           |
| `BeforeAll` panics                   | `entry` is **not** invoked and `AfterAll` does **not** run                             |

> ⚠️ The last row is intentional. `defer r.ApplyFinish()` is registered **after** `r.ApplyStart()` succeeds, so if
> `BeforeAll` panics before completing, no `AfterAll` is queued. If your `BeforeAll` allocates resources before the
> point where it can panic, clean them up inside the failing hook via its own `defer`.

---

## When to use `RunPackage`

Use `RunPackage` when **all** of the following hold:

- the package has more than one top-level `TestXxx` that share the same `Runner`
- that shared `Runner` owns expensive setup (resources, containers, clients)
- you want `BeforeAll`/`AfterAll` to wrap the whole package, not the first test

If you only have one top-level `TestXxx` with subtests, you already have a deterministic boundary — no `TestMain`
needed. If your tests are organized as a `Suite`, the suite already manages its own lifecycle — `RunPackage` is
unnecessary.

---

## When **not** to use `RunPackage`

Skip `RunPackage` when:

- each `TestXxx` uses its own `Runner` (no shared lifecycle to manage)
- the package has exactly one `TestXxx` that drives everything via subtests
- the test entry point is already managed by `axiom.NewSuite` / `axiom.NewSuiteFactory`

In these cases the `Runner` lifecycle is already bound to a deterministic `*testing.T`, and adding `RunPackage` on top
only introduces an extra layer with no observable benefit.

---

## Comparison with the other lifecycle boundaries

| Boundary               | Use it when                                                                                          |
|------------------------|------------------------------------------------------------------------------------------------------|
| `RunPackage`           | Many top-level `TestXxx` share one runner across the **whole package**.                              |
| `Suite`                | Several related cases form a logical group inside one test binary; see [docs/suite](../suite).       |
| Single `TestXxx`       | One top-level test runs everything as subtests; the simplest case, no extra plumbing required.       |

All three converge on the same guarantee — `AfterAll` fires once, after the chosen boundary completes — but each
expresses that boundary at a different scope.

---

## Example

```go
package users_test

import (
    "os"
    "testing"

    "github.com/Nikita-Filonov/axiom"
)

// Package-wide runner with expensive shared setup.
var runner = axiom.NewRunner(
    axiom.WithRunnerResource("db", DBResource),
    axiom.WithRunnerHooks(
        axiom.WithBeforeAll(func(r *axiom.Runner) {
            // expensive bootstrap that should run exactly once per package
        }),
        axiom.WithAfterAll(func(r *axiom.Runner) {
            // global teardown, observed before resource cleanup runs
        }),
    ),
)

func TestMain(m *testing.M) {
    os.Exit(axiom.RunPackage(m, runner))
}

func TestCreateUser(t *testing.T) {
    runner.RunCase(t, createUserCase, func(cfg *axiom.Config) {
        cfg.Step("create", func() { /* ... */ })
    })
}

func TestBlockUser(t *testing.T) {
    runner.RunCase(t, blockUserCase, func(cfg *axiom.Config) {
        cfg.Step("block", func() { /* ... */ })
    })
}
```

`BeforeAll` runs once before `TestCreateUser`, `AfterAll` runs once after `TestBlockUser`, and the DB resource is
torn down exactly once at the very end.
