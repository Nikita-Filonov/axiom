# 🔭 OpenTelemetry Plugin (`testotel`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [What the plugin exports](#what-the-plugin-exports)
- [Trace context](#trace-context)
- [Configuration and lifetime](#configuration-and-lifetime)
- [Installation](#installation)
- [Example](#example)

---

## Overview

`testotel` records Axiom test attempts as OpenTelemetry spans. It is a separate
Go module; Axiom core has no OpenTelemetry dependency. The plugin uses the
caller’s tracer provider and does not create an SDK, exporter, collector, or
backend. Existing OpenTelemetry setup can send the spans to any compatible
destination.

This plugin complements [testtracing](../testtracing), which keeps raw Axiom
events in memory. `testotel` translates selected lifecycle events to telemetry
for an external tracing pipeline.

---

## What the plugin exports

- One `axiom.test` span per attempt, including retries. All attempts in one
  `RunCase` share the `axiom.execution.id` attribute and have distinct
  `axiom.execution.attempt` numbers. The span starts when the attempt runs and
  ends after its `testing.T.Cleanup` callbacks.
- A policy skip creates one span marked `skipped`, even when Axiom skips on its
  planning Config before starting retries. A passing attempt is marked
  `passed`; a failing attempt is marked `failed` and gets OTel error status.
- Step, setup, teardown, and fixture lifecycle events become span events.
  Their names are exported as `axiom.event.name` when present. Case ID, case
  name, and Go test name are span attributes.

The plugin **does not export** event messages, log bodies, assertion text,
artefact content, or skip reasons. These may contain secrets. Case IDs and
names, Go test names, and step and fixture names **are** exported, so avoid
putting secrets in them. Runner-wide `BeforeAll`/`AfterAll` events and resource
operations are outside the attempt's Config and are not captured.

`testotel` currently exports traces only. It uses `axiom.*` attributes because
OpenTelemetry's [test attributes](https://opentelemetry.io/docs/specs/semconv/registry/attributes/test/)
are still marked Development. The plugin does not infer a flaky run outcome;
each attempt has its own result. Use [teststats](../teststats) for run-level
flaky classification and counts.

Status is read when the attempt's final cleanup runs. A failure that Go reports
only afterward, such as some cleanup panics or race-detector failures, cannot
change an already ended span. Go's test result remains authoritative.

---

## Trace context

Call `testotel.Context(cfg)` inside a test, hook, or fixture to get the active
attempt span's `context.Context`. Pass it to instrumented HTTP clients,
database calls, or other operations to make their spans children of the test
span. If no attempt is active, it returns `cfg.Context.Raw`, or
`context.Background()` when that field is unset. A nil Config panics.

The plugin leaves `cfg.Context.Raw`, `DB`, `MQ`, and `RPC` unchanged. Merely
installing it does not propagate trace context through outgoing requests; the
test must pass the returned context. Cross-process correlation also requires
the client's instrumentation and propagation setup.

---

## Configuration and lifetime

Register `testotel.Plugin()` in `axiom.WithRunnerPlugins(...)` or
`axiom.WithCasePlugins(...)`. By default it uses the global OTel tracer
provider. Configure the SDK before creating the plugin, or pass a provider
explicitly with `testotel.WithTracerProvider(provider)`. A nil option or
provider panics. Reapplying the plugin to one Config does not duplicate its
span.

The caller owns the provider. Flush and shut it down after the test suite has
finished, usually in `TestMain`. The plugin never shuts down a provider it did
not create. If no recording provider is configured, the global provider may
produce no exported spans.

---

## Installation

The plugin is an independently versioned Go module. Install it with standard
Go tooling:

```shell
go get github.com/Nikita-Filonov/axiom/plugins/testotel
```

---

## Example

The SDK and exporter belong to the test application. This example uses a
span recorder for local inspection; replace it with an OTLP exporter when
connecting to a collector. See the [runnable example](example_test.go).

```go
package testotel_test

import (
	"context"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testotel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTracedCase(t *testing.T) {
	// This in-memory recorder stands in for an exporter.
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	// The test owns the provider and shuts it down after all subtests finish.
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Error(err)
		}
	})

	// Installing the plugin on the Runner traces every case attempt it runs.
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testotel.Plugin(testotel.WithTracerProvider(provider)),
		),
	)
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("request works")), func(cfg *axiom.Config) {
		// Pass the attempt context to instrumented clients. This child span
		// stands in for a span created by an HTTP or database client.
		ctx := testotel.Context(cfg)
		_, child := provider.Tracer("application").Start(ctx, "request")
		child.End()

		// Axiom step transitions appear as events on the attempt span.
		cfg.Step("check response", func() {})
	})

	// RunCase waits for the attempt's cleanup, so both spans have ended.
	if got := len(recorder.Ended()); got != 2 {
		t.Fatalf("got %d completed spans, want 2", got)
	}
}
```
