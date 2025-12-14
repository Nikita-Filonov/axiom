# 📘 Artefacts

An `Artefact` represents a binary or structured output produced during test execution.

Artefacts are **data-oriented**: they capture what the test produced (payloads, responses, screenshots, dumps), not what
the test did.

Artefacts are emitted via `cfg.Artefact(...)` and routed through `Runtime` artefact sinks.

## Example

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

func TestArtefactExample(t *testing.T) {

	runner := axiom.NewRunner(
		axiom.WithRunnerRuntime(

			// Simple artefact sink
			axiom.WithRuntimeArtefactSink(func(a axiom.Artefact) {
				fmt.Println("artefact:", a.Name, "type:", a.Type, "size:", len(a.Data))
			}),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("artefact demo"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		cfg.Step("generate data", func() {

			payload := map[string]any{
				"id":   1,
				"name": "demo",
			}

			a, _ := axiom.NewJSONArtefact("payload", payload)
			cfg.Artefact(a)
		})

		cfg.Artefact(
			axiom.NewTextArtefact("note", "test finished successfully"),
		)
	})
}

```
