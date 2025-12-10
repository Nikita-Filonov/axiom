# 📘 Config

`Config` is the **runtime execution model** for a single test attempt (including retries).

Where:

- `Runner` defines global infrastructure,
- `Case` describes the logical test scenario,

`Config` is the **fully merged, normalized snapshot** of everything needed to execute the test: metadata, context,
policies, hooks, fixtures, and middleware.

Every retry attempt receives a **fresh** `Config`, ensuring deterministic test behavior and clean fixture lifecycle.

`Config` is the primary object passed to the test body and to all plugins, steps, hooks, and wrap functions.

## Why `Config` exists

Axiom separates _declarative configuration_ from _runtime behavior_:

| Layer      | Purpose                                             |
|------------|-----------------------------------------------------|
| **Runner** | global, reusable infrastructure                     |
| **Case**   | per-test declarative configuration                  |
| **Config** | final merged runtime for a single execution attempt |

`Config` solves key problems:

- unified access to metadata, context, fixtures, retry policies, and hooks
- deterministic retry (clean state for each attempt)
- middleware-based extension model (test wraps, step wraps)
- structured step execution
- safe and typed resource access

Everything that happens **during** a test run flows through `Config`.

## Example

```go
package example_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

func TestConfigExample(t *testing.T) {

	// ---------------------------------------------------------
	// Global (Runner-level) configuration
	// ---------------------------------------------------------

	runner := axiom.NewRunner(
		axiom.WithRunnerMeta(
			axiom.WithMetaTag("example"),
		),
		axiom.WithRunnerFixture("db", func(cfg *axiom.Config) (any, func(), error) {
			db := "db-" + cfg.ID
			return db, func() { fmt.Println("cleanup:", db) }, nil
		}),
		axiom.WithRunnerRetry(
			axiom.WithRetryTimes(2),
			axiom.WithRetryDelay(100*time.Millisecond),
		),
	)

	// ---------------------------------------------------------
	// Case configuration
	// ---------------------------------------------------------

	c := axiom.NewCase(
		axiom.WithCaseName("config demo"),
		axiom.WithCaseContext(
			axiom.WithContextData("request_id", "abc-123"),
		),
	)

	// ---------------------------------------------------------
	// Test execution
	// ---------------------------------------------------------

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		fmt.Println("Meta tags:", cfg.Meta.Tags)

		// Context values
		reqID := axiom.MustContextValue[string](&cfg.Context, "request_id")
		fmt.Println("RequestID:", reqID)

		// Fixture access
		db := axiom.GetFixture[string](cfg, "db")
		fmt.Println("DB:", db)

		// Step execution
		cfg.Step("perform operation", func() {
			fmt.Println("→ doing work")
		})

		// SubTest execution (middleware-aware)
		cfg.SubTest(func(c *axiom.Config) {
			fmt.Println("→ inside subtest")
		})
	})
}

```

## How `Config` Is Built (Merging Model)

`Config` is constructed inside `Runner.RunCase`:

```text
Runner.Meta     + Case.Meta     → Config.Meta
Runner.Context  + Case.Context  → Config.Context
Runner.Retry    + Case.Retry    → Config.Retry
Runner.Hooks    + Case.Hooks    → Config.Hooks
Runner.Fixtures + Case.Fixtures → Config.Fixtures
Runner.Parallel + Case.Parallel → Config.Parallel
```

Then plugins are applied:

```text
Config → Plugin1 → Plugin2 → ...
```

Then retry loop begins, and each attempt creates a fresh `Config`.
