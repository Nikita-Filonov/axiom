# 📘 Context

`Context` provides structured per-test contextual data used by plugins, fixtures, steps, and integrations. Axiom exposes
several typed channels (`Raw`, `HTTP`, `GRPC`, `Kafka`) along with an extensible key/value store (`Data`)
for arbitrary metadata.

`Context` may be defined at both `Runner` and `Case` level; `Case` overrides `Runner`. All context fields are merged and
normalized during `Config` creation.

This model enables:

- unified carriers for `HTTP`/`GRPC`/`Kafka` clients
- propagation of request IDs, environment markers, tenant identifiers
- plugin-level context injection

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
			axiom.WithContextHTTP(context.WithValue(context.Background(), "http-timeout", 5000)),
			axiom.WithContextData("request_id", "abc-123"),
		),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		// Typed data lookup
		env, _ := axiom.GetContextValue[string](&cfg.Context, "env")
		req := axiom.MustContextValue[string](&cfg.Context, "request_id")

		fmt.Println("env:", env)
		fmt.Println("request:", req)

		// Typed channel contexts
		fmt.Println("HTTP ctx:", cfg.Context.HTTP.Value("http-timeout"))

		cfg.Step("work", func() {
			fmt.Println("performing work...")
		})
	})
}

```
