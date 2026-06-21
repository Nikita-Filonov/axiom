# 🔎 Tracing Plugin (`testtracing`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [What the plugin does](#what-the-plugin-does)
- [Installation](#installation)
- [Example](#example)

---

## Overview

Collects raw config-scoped Axiom runtime events into an in-memory trace.

The plugin does not calculate statuses or attempts. It stores events exactly as they are emitted by the runtime and
groups config-scoped events into consumer-level `TraceRecord` values. Consumers can decide which events to keep, export,
aggregate, or ignore.

---

## What the plugin does

At runtime, the plugin:

- subscribes to the applied config `Runtime` event sink
- appends config-scoped events to a shared `Trace`
- groups config-scoped events into `TraceRecord` snapshots
- preserves events as-is
- stops collecting events for a test attempt after `testing.T.Cleanup`

---

## Installation

The plugin is distributed as a regular Go module and installed using standard Go tooling.

Add the plugin dependency using `go get`:

```shell
go get github.com/Nikita-Filonov/axiom/plugins/testtracing
```

---

## Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testtracing"
)

func TestTracingExample(t *testing.T) {
	// Trace is a shared in-memory collector. The plugin writes records into it
	// while tests run, and the test or reporter can inspect it afterwards.
	trace := testtracing.NewTrace()

	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			// Installing the plugin at Runner level applies it to every Config
			// built by this runner. Each Config gets its own TraceRecord.
			testtracing.Plugin(trace),
		),
	)

	c := axiom.NewCase(
		// Case data is copied into TraceRecord.Case. It is not injected into
		// individual events; events stay raw.
		axiom.WithCaseName("tracing example"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		// These calls emit config-scoped runtime events. The tracing plugin
		// appends them to the TraceRecord for this Config.
		cfg.Step("do work", func() {})
		cfg.Log(axiom.NewInfoLog("done"))
	})

	// Snapshot returns a copy of collected records. One record corresponds to
	// one Config attempt; retries produce additional records with the same Case.
	for _, record := range trace.Snapshot() {
		// Case and Meta are consumer-level context captured by the plugin.
		// They help callers group and export events without making events smart.
		_ = record.Case
		_ = record.Meta

		for _, event := range record.Events {
			// Events are preserved as emitted. Consumers decide how to filter,
			// aggregate, or interpret Type, Name, and Message.
			_ = event.Type
			_ = event.Name
			_ = event.Message
		}
	}
}
```
