# 📘 Hooks

`Hooks` provide lifecycle callbacks that run before and after key execution stages: tests, steps, and subtests. `Hooks`
enable logging, reporting, tracing, instrumentation, metrics, and other side-effect integrations without modifying
test code.

Hooks may be defined at both Runner and Case level. Case-level hooks are appended after Runner hooks, forming a single
ordered pipeline.

A hook does **not** alter control flow — it only observes execution.

Available hook types:

- `BeforeTest(cfg)`
- `AfterTest(cfg)`
- `BeforeStep(cfg, name)`
- `AfterStep(cfg, name)`
- `BeforeSubTest(cfg)`
- `AfterSubTest(cfg)`

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

func beforeTest(c *axiom.Config)           { fmt.Println("→ before test") }
func afterTest(c *axiom.Config)            { fmt.Println("→ after test") }
func beforeStep(c *axiom.Config, n string) { fmt.Println("→ before step:", n) }
func afterStep(c *axiom.Config, n string)  { fmt.Println("→ after step:", n) }
func beforeSub(c *axiom.Config)            { fmt.Println("→ before subtest") }
func afterSub(c *axiom.Config)             { fmt.Println("→ after subtest") }

// -----------------------------------------------------------------------------
// Runner with global hooks
// -----------------------------------------------------------------------------

var runner = axiom.NewRunner(
	axiom.WithRunnerHooks(
		axiom.WithBeforeTest(beforeTest),
		axiom.WithAfterTest(afterTest),
		axiom.WithBeforeStep(beforeStep),
		axiom.WithAfterStep(afterStep),
		axiom.WithBeforeSubTest(beforeSub),
		axiom.WithAfterSubTest(afterSub),
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

		cfg.SubTest(func(sub *axiom.Config) {
			fmt.Println("inside subtest")
		})

		cfg.Step("finish", func() {
			fmt.Println("finishing...")
		})
	})
}

```
