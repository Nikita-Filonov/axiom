# 📘 Log

`Log` represents a structured runtime message emitted during test execution.

Logs are **imperative and observational**: they do not affect control flow, retries, skips, or assertions. They exist to
be **consumed by runtime sinks** (console output, reporters, external systems).

Logs are emitted via `cfg.Log(...)` and routed through `Runtime` log sinks.

## Example

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

func TestLogExample(t *testing.T) {

	runner := axiom.NewRunner(
		axiom.WithRunnerRuntime(

			// Simple console log sink
			axiom.WithRuntimeLogSink(func(l axiom.Log) {
				fmt.Println("[", l.Level, "]", l.Text)
			}),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("logging demo"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Log(axiom.NewInfoLog("starting test"))

		cfg.Step("do work", func() {
			cfg.Log(axiom.NewDebugLog("inside step"))
		})

		cfg.Log(axiom.NewWarningLog("non-critical issue detected"))
	})
}

```
