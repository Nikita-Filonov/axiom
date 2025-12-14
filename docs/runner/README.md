# 📘 Runner

A `Runner` defines the global execution environment for tests. It provides metadata, retry policy, fixtures, context,
hooks, plugins, and parallelization settings that apply to all cases executed through it.

A `Runner` does **not** describe a test — it executes Cases. Before running a test, a `Runner` merges its configuration
with a specific `Case`, producing a runtime `Config` object.

This layered model enables:

- consistent global behavior across tests
- predictable overrides at `Case` level
- shared fixtures and context
- unified plugin and reporting pipelines

---

## Example

The following example demonstrates every Runner option in a single, self-contained configuration.

```go
package example_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

// -----------------------------------------------------------------------------
// Example fixtures
// -----------------------------------------------------------------------------

func DBFixture(cfg *axiom.Config) (any, func(), error) {
	db := fmt.Sprintf("db-%s", cfg.ID)
	cleanup := func() { fmt.Println("closing:", db) }
	return db, cleanup, nil
}

// -----------------------------------------------------------------------------
// Example plugin
// -----------------------------------------------------------------------------

func LoggingPlugin() axiom.Plugin {
	return func(cfg *axiom.Config) {
		cfg.TestWraps = append(cfg.TestWraps, func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				start := time.Now()
				next(c)
				fmt.Println("duration:", time.Since(start))
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Example hooks
// -----------------------------------------------------------------------------

func beforeTest(c *axiom.Config)           { fmt.Println("→ before test") }
func afterTest(c *axiom.Config)            { fmt.Println("→ after test") }
func beforeStep(c *axiom.Config, _ string) { fmt.Println("→ before step") }
func afterStep(c *axiom.Config, _ string)  { fmt.Println("→ after step") }

// -----------------------------------------------------------------------------
// Runner definition
// -----------------------------------------------------------------------------

var runner = axiom.NewRunner(

	// Global metadata applied to all tests
	axiom.WithRunnerMeta(
		axiom.WithMetaEpic("authentication"),
		axiom.WithMetaFeature("login"),
		axiom.WithMetaTag("regression"),
	),

	// Global skip rules
	// axiom.WithRunnerSkip(axiom.WithSkipReason("maintenance")),

	// Global retry policy (used unless Case overrides)
	axiom.WithRunnerRetry(
		axiom.WithRetryTimes(3),
		axiom.WithRetryDelay(50),
	),

	// Global hooks
	axiom.WithRunnerHooks(
		axiom.WithBeforeTest(beforeTest),
		axiom.WithAfterTest(afterTest),
		axiom.WithBeforeStep(beforeStep),
		axiom.WithAfterStep(afterStep),
	),

	// Global context values
	axiom.WithRunnerContext(
		axiom.WithContextData("env", "staging"),
	),

	// Global plugins (applied before Case-specific plugins)
	axiom.WithRunnerPlugins(
		LoggingPlugin(),
	),

	// Enable parallel execution by default
	axiom.WithRunnerParallel(),

	// Global fixtures shared across all tests
	axiom.WithRunnerFixture("db", DBFixture),

	// Global runtime behavior
	axiom.WithRunnerRuntime(

		// Wrap every test
		axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				fmt.Println("[runner] before test")
				next(c)
				fmt.Println("[runner] after test")
			}
		}),

		// Wrap every step
		axiom.WithRuntimeStepWrap(func(name string, next axiom.StepAction) axiom.StepAction {
			return func() {
				fmt.Println("[runner] step:", name)
				next()
			}
		}),
	),

)

// -----------------------------------------------------------------------------
// Example test run
// -----------------------------------------------------------------------------

func TestRunnerExample(t *testing.T) {

	// Case definition (can override Runner settings)
	c := axiom.NewCase(
		axiom.WithCaseName("user can log in"),
		axiom.WithCaseMeta(axiom.WithMetaTag("smoke")),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		db := axiom.GetFixture[string](cfg, "db")
		fmt.Println("using fixture:", db)

		cfg.Step("login", func() {
			fmt.Println("perform login")
		})

		cfg.Step("validate", func() {
			fmt.Println("validate result")
		})
	})
}

```
