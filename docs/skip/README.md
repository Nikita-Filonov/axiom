# 📘 Skip

`Skip` allows marking a test as skipped, either statically or dynamically. A skip definition may include an optional
reason, and may be applied at both `Runner` and `Case` level. Case-level skip overrides Runner-level skip.

A skipped test does not execute fixtures, steps, hooks, or plugins (except those involved in reporting skip state).

This model enables:

- declarative disabling of tests
- CI-conditional skipping
- environment-based skipping
- consistent merging (`Case` > `Runner`)

---

## Merge semantics

`Skip` distinguishes between **not set** and **explicitly set to false** through the `EnabledSet` flag:

| Builder                       | `Enabled` | `EnabledSet` | Reason       |
|-------------------------------|-----------|--------------|--------------|
| `SkipBecause("…")`            | `true`    | `true`       | `"…"`        |
| `WithSkipEnabled(true)`       | `true`    | `true`       | unchanged    |
| `WithSkipEnabled(false)`      | `false`   | `true`       | unchanged    |
| `WithSkipDisabled()`          | `false`   | `true`       | unchanged    |
| `WithSkipReason("…")` (only)  | unchanged | `false`      | `"…"`        |

`Skip.Join(other)` replaces `Enabled` only when `other.EnabledSet` is `true`. This means a `Case` can do all of:

- inherit `Runner` skip (do not call any skip builder)
- override `Runner` skip with a different reason (`SkipBecause(...)`)
- **explicitly turn off** an inherited Runner skip (`WithSkipDisabled()`)

Without `EnabledSet` the case-level `Enabled: false` would be indistinguishable from the zero value, so an override
back to "run this test" would be impossible. This mirrors how `Retry.TimesSet` and `Parallel.EnabledSet` work for the
same reason.

```go
runner := axiom.NewRunner(
    axiom.WithRunnerSkip(axiom.SkipBecause("maintenance window")),
)

// This case opts out of the runner-level skip even though Enabled=false is the
// zero value of bool — EnabledSet makes the override explicit.
c := axiom.NewCase(
    axiom.WithCaseName("smoke test that must always run"),
    axiom.WithCaseSkip(axiom.WithSkipDisabled()),
)
```

---

## Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

func TestSkipExample(t *testing.T) {

	// -------------------------------------------------------------------------
	// Runner-level skip (disabled by default)
	// -------------------------------------------------------------------------

	runner := axiom.NewRunner(
		axiom.WithRunnerSkip(
			// axiom.SkipBecause("maintenance window"),
		),
	)

	// -------------------------------------------------------------------------
	// Case-level skip overrides Runner settings
	// -------------------------------------------------------------------------

	c := axiom.NewCase(
		axiom.WithCaseName("skipped example"),
		axiom.WithCaseSkip(
			axiom.SkipBecause("feature temporarily disabled"),
		),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		// This block will not run
		cfg.Step("should not execute", func() {})
	})
}

```
