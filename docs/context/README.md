# 📘 Context

`Context` provides structured per-test **execution context** used by plugins, fixtures, steps, and integrations. It
represents **lifecycle and cancellation boundaries**, not concrete technologies.

Axiom exposes several **typed context channels**:

- `Raw` — root context (base cancellation & deadlines)
- `DB` — database lifecycle (connections, transactions)
- `MQ` — message queues / streams (consumers, producers)
- `RPC` — outbound calls (HTTP, gRPC, etc.)

In addition, `Context` provides an extensible key/value store (`Data`) for arbitrary runtime metadata.

`Context` may be defined at both `Runner` and `Case` level; `Case` overrides `Runner`. All context fields are merged and
normalized during `Config` creation.

This model enables:

- clear separation between execution lifecycle and infrastructure
- unified cancellation and timeout propagation
- predictable context overrides at runner / case scope
- plugin- and fixture-level context injection without `context.WithValue`

---

## Design principles

- `Context` is not a dependency injection container
- concrete clients (HTTP, gRPC, Kafka, DB, etc.) should be created in **fixtures**
- `Context` only defines lifecycle boundaries
- `Data` is for lightweight runtime values (IDs, flags, environment markers)

---

## Example

```go
package example_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

func TestContextExample(t *testing.T) {

	// -------------------------------------------------------------------------
	// Runner-level context
	// -------------------------------------------------------------------------

	runner := axiom.NewRunner(
		axiom.WithRunnerContext(
			axiom.WithContextRaw(context.WithValue(context.Background(), "global", "yes")),
			axiom.WithContextData("env", "staging"),
		),
	)

	// -------------------------------------------------------------------------
	// Case-level context (overrides runner)
	// -------------------------------------------------------------------------

	c := axiom.NewCase(
		axiom.WithCaseName("context example"),
		axiom.WithCaseContext(
			axiom.WithContextRPC(
				context.WithValue(context.Background(), "timeout_ms", 5000),
			),
			axiom.WithContextData("request_id", "abc-123"),
		),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		// Typed data lookup
		env, _ := axiom.GetContextValue[string](&cfg.Context, "env")
		req := axiom.MustContextValue[string](&cfg.Context, "request_id")

		fmt.Println("env:", env)
		fmt.Println("request:", req)

		// Typed execution contexts
		fmt.Println("RPC ctx timeout:", cfg.Context.RPC.Value("timeout_ms"))

		cfg.Step("work", func() {
			fmt.Println("performing work...")
		})
	})
}

```
