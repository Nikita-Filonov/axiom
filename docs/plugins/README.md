# 📘 Plugins

---

## 📑 Table of Contents

- [Overview](#overview)
- [Installing Plugins](#installing-plugins)
- [Writing a Plugin](#writing-a-plugin)
- [Built-in Plugins](#built-in-plugins)

---

## Overview

A `Plugin` is a function that configures test execution via `Config` and its `Runtime`. Plugins extend Axiom without
changing its core. They may attach hooks, wraps, context values, reporting integrations, filtering logic, or custom
instrumentation.

A plugin is applied:

1. At `Runner` level (global, applied first)
2. At `Case` level (applied after Runner plugins)

Plugins form a deterministic mutation pipeline.

A plugin does **not** execute tests or steps — it only decorates execution by registering behavior in `Config` and
`Runtime`.

---

## Installing Plugins

Axiom plugins are distributed as **regular Go modules**. There is no plugin manager, registry, or custom installation
mechanism.

Plugins are installed and versioned using standard Go tooling.

### Installing a plugin

Use `go get` with the plugin module path:

```bash
go get github.com/Nikita-Filonov/axiom/plugins/testtags@v0.1.0
```

This will add the plugin as a dependency to your `go.mod` file:

```text
require (
	github.com/Nikita-Filonov/axiom v0.3.0
	github.com/Nikita-Filonov/axiom/plugins/testtags v0.1.0
)
```

Each plugin is **versioned** independently of the Axiom core.

---

## Writing a Plugin

A plugin has the type:

```go
type Plugin func(cfg *axiom.Config)
```

A minimal plugin:

```go
package myplugin

import (
	"fmt"

	"github.com/Nikita-Filonov/axiom"
)

func Plugin() axiom.Plugin {
	// A plugin is applied to each built Config. It should register behavior on
	// Config/Runtime and return; it should not run the test by itself.
	return func(cfg *axiom.Config) {
		// Test wraps are middleware. This registration only changes how the test
		// will execute later, when cfg.Test(...) reaches Runtime.Test(...).
		cfg.Runtime.EmitTestWrap(func(next axiom.TestAction) axiom.TestAction {
			// The outer function receives the next action in the chain.
			// The returned action becomes the decorated test body.
			return func(c *axiom.Config) {
				fmt.Println("before test")
				next(c)
				fmt.Println("after test")
			}
		})
	}
}
```

Plugins commonly interact with:

- `cfg.Runtime.EmitTestWrap(...)` — wrap test execution
- `cfg.Runtime.EmitStepWrap(...)` — wrap step execution
- `cfg.Runtime.EmitLogSink(...)` — consume logs
- `cfg.Runtime.EmitAssertSink(...)` — consume asserts
- `cfg.Runtime.EmitArtefactSink(...)` — consume artefacts
- `cfg.Hooks.*` — lifecycle hooks
- `cfg.Skip` — skip logic
- `cfg.Context` — context injection
- `cfg.Meta` — metadata modification

---

## Built-in Plugins

Axiom ships with several built-in plugins that demonstrate common patterns for extending the runtime. Each plugin is
fully self-contained and documented in its own README.

These plugins are intended both for direct use and as reference implementations when writing custom plugins.

- **🟣 Allure Plugin:** [testallure](../../plugins/testallure). Generates Allure reports by projecting Axiom runtime
  events (tests, steps, artefacts, metadata) into the Allure execution model.
- **📝 Logger Plugin:** [testlogger](../../plugins/testlogger). Consumes structured log events emitted via `cfg.Log(...)`
  and forwards them to Go’s `log/slog` logging infrastructure.
- **📊 Stats Plugin:** [teststats](../../plugins/teststats). Collects execution statistics for test cases, including
  attempts, duration, final status, and metadata snapshots.
- **🔎 Tracing Plugin:** [testtracing](../../plugins/testtracing). Records raw config-scoped runtime events into an
  in-memory trace for later inspection or export.
- **🧭 Explain Plugin:** [testexplain](../../plugins/testexplain). Captures a structured explanation of the merged
  runner/case configuration before test execution.
- **🏷 Tags Plugin:** [testtags](../../plugins/testtags). Filters test execution based on metadata tags using include /
  exclude rules. Can be configured via code or environment variables.
- **✅ Assert Plugin:** [testassert](../../plugins/testassert). Bridges Axiom’s structured runtime assertions with
  `stretchr/testify/assert`. Allows test code to emit declarative assertion events without coupling to a specific
  assertion backend.
