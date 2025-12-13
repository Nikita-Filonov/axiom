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
