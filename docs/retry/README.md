# 📘 Retry

## 📑 Table of Contents

- [Semantics](#semantics)
- [Parallel Cases](#parallel-cases)
- [Re-running failed tests with gotestsum](#re-running-failed-tests-with-gotestsum)
- [Example](#example)

---

`Retry` defines how many times a test may re-run and how long to wait between attempts. `Retry` configuration may be
applied at both Runner and `Case` level. Case-level settings override Runner-level defaults.

A retry is applied when an attempt fails. Each retry creates a fresh `Config`, causing fixtures to
re-evaluate and hooks to re-run, ensuring isolated, deterministic attempts.

> Axiom retries happen within one `go test` run. Once an attempt reports a failure to `testing.T`, that run remains
> failed even if a later attempt passes. Use a tool that starts a new `go test` run, such as
> [gotestsum](https://github.com/gotestyourself/gotestsum#re-running-failed-tests), when a passing rerun should determine
> the final command result.

This model enables:

- repeated attempt lifecycles for diagnostics
- predictable overrides (`Case` > `Runner`)
- isolated fixture lifecycles per attempt
- configurable retry delays

---

## Semantics

- Retry is **disabled by default** (`Times = 1`, `Delay = 0`)
- `Times` is always normalized to a minimum of `1`
- `Delay` is always normalized to a minimum of `0`
- Case-level retry settings override Runner-level settings **per field**
- Unset fields inherit values from the Runner
- When Retry and Parallel are both enabled, the Case runs in parallel while its attempts run sequentially
- A failure already reported to Go's `testing.T` remains part of the final test result even if a later attempt passes

Normalization guarantees that retries are always safe and deterministic, even when invalid values are explicitly
provided.

## Parallel Cases

Retry attempts must finish before Axiom can decide whether to run the next attempt. For a parallel Case, Axiom therefore
applies `t.Parallel()` once to the Case wrapper and executes the attempt subtests sequentially inside it:

Schematically:

```text
parallel case
├── attempt 1
├── attempt 2
└── attempt 3
```

Different Cases can still run concurrently. Hooks, plugins, Config, local fixtures, and fixture cleanup are recreated or
applied for every attempt as they are for a sequential retry.

Go does not allow a failed `testing.T` to become successful again. Retry therefore re-executes the Case lifecycle, but
does not erase a failure already reported by an earlier attempt.

---

## Re-running failed tests with `gotestsum`

`gotestsum` runs standard `go test` commands and can start a fresh run for failed tests. Axiom tests need no special
integration because they are ordinary Go tests. From a Go module directory, run:

```sh
gotestsum --rerun-fails
```

This is a command-level retry: it reruns a failed Go test in a new invocation. Axiom's `Retry` controls attempts within
one invocation and preserves every failure reported to `testing.T`. See the
[gotestsum rerun documentation](https://github.com/gotestyourself/gotestsum#re-running-failed-tests) for options and
argument handling.

---

## Example

```go
package example_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

func TestRetryExample(t *testing.T) {

	// -------------------------------------------------------------------------
	// Runner-level retry policy
	// -------------------------------------------------------------------------

	runner := axiom.NewRunner(
		axiom.WithRunnerRetry(
			axiom.WithRetryTimes(2),
			axiom.WithRetryDelay(1*time.Second),
		),
	)

	// -------------------------------------------------------------------------
	// Case overrides retry settings
	// -------------------------------------------------------------------------

	c := axiom.NewCase(
		axiom.WithCaseName("retry example"),
		axiom.WithCaseRetry(
			axiom.WithRetryTimes(3), // overrides Runner value
			axiom.WithRetryDelay(500*time.Millisecond),
		),
	)

	attempt := 0

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		attempt++
		fmt.Println("attempt:", attempt)

		if attempt < 3 {
			cfg.T().Fail() // fail the current attempt and trigger retry
		}

		cfg.Step("finalize", func() {
			fmt.Println("action completed on attempt", attempt)
		})
	})
}

```

This example deliberately produces a failing `go test` result: attempts 1 and 2 call `Fail`, and attempt 3 completes
the action. It demonstrates repeated attempt lifecycles, not recovery of the final test result.
