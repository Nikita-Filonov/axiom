# 📘 Retry

`Retry` defines how many times a test may re-run and how long to wait between attempts. `Retry` configuration may be
applied at both Runner and `Case` level. Case-level settings override Runner-level defaults.

A retry is applied only when the test body returns a failure. Each retry creates a fresh `Config`, causing fixtures to
re-evaluate and hooks to re-run, ensuring isolated, deterministic attempts.

This model enables:

- consistent flaky-test handling
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

Normalization guarantees that retries are always safe and deterministic, even when invalid values are explicitly
provided.

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
			t.Fail() // trigger retry
		}

		cfg.Step("finalize", func() {
			fmt.Println("success on attempt", attempt)
		})
	})
}

```
