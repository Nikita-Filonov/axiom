# 📘 Plugins

---

## 📑 Table of Contents

- [Overview](#overview)
- [Installing Plugins](#installing-plugins)
- [Writing a Plugin](#writing-a-plugin)
- [Built-in Plugins](#built-in-plugins)
    - [Allure Plugin (`testallure`)](#-allure-plugin-testallure)
    - [Stats Plugin (`teststats`)](#-stats-plugin-teststats)
    - [Tags Plugin (`testtags`)](#-tags-plugin-testtags)
- [Writing Your Own Plugin](#writing-your-own-plugin)

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
type Plugin func (cfg *axiom.Config)
```

A minimal plugin:

```go
package myplugin

import (
	"fmt"

	"github.com/Nikita-Filonov/axiom"
)

func Plugin() axiom.Plugin {
	return func(cfg *axiom.Config) {
		cfg.Runtime.EmitTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				fmt.Println("before test")
				next(c)
			}
		})
	}
}

```

Plugins commonly interact with:

- `cfg.Runtime.EmitTestWrap(...)` — wrap test execution
- `cfg.Runtime.EmitStepWrap(...)` — wrap step execution
- `cfg.Runtime.EmitLogSink(...)` — consume logs
- `cfg.Runtime.EmitArtefactSink(...)` — consume artefacts
- `cfg.Hooks.*` — lifecycle hooks
- `cfg.Skip` — skip logic
- `cfg.Context` — context injection
- `cfg.Meta` — metadata modification

---

## Built-in Plugins

Axiom ships with several reference plugins demonstrating how to build common tooling integrations.

### 🟣 Allure Plugin (`testallure`)

Generates Allure reporting via `dailymotion/allure-go`.

The plugin wraps:

- test execution via runtime test wraps
- step execution via runtime step wraps
- artefacts emitted during execution

Tags, epic, story, labels, severity, and other metadata are converted into Allure options.

#### Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testallure"
)

func TestAllureExample(t *testing.T) {

	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testallure.Plugin(),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("allure test"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Step("demo step", func() {
			// Allure automatically wraps this step
		})
	})
}
```

### 📊 Stats Plugin (`teststats`)

Collects execution statistics for each test:

- total attempts
- duration
- final status (passed/failed/skipped/flaky)
- metadata snapshot
- start/end timestamps

The plugin uses `BeforeSubTest` / `AfterSubTest` hooks to measure attempts and result finalization.

#### Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/teststats"
)

func TestStatsExample(t *testing.T) {

	stats := teststats.NewStats()

	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			teststats.Plugin(stats),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("stats test"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Step("work", func() {})
	})

	// Stats available after run
	_ = stats.Cases
}

```

### 🏷 Tags Plugin (`testtags`)

Filters tests based on metadata tags using include/exclude rules.

Features:

- include only tests with specified tags
- exclude tests with specified tags
- environment-driven config (`AXIOM_TEST_TAGS_INCLUDE`, `AXIOM_TEST_TAGS_EXCLUDE`)
- normalization, case-insensitive matching

If a tag rule fails, the plugin sets:

```go
cfg.Skip = axiom.Skip{Enabled: true, Reason: "..."}

```

#### Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testtags"
)

func TestTagsExample(t *testing.T) {

	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testtags.Plugin(
				testtags.WithConfigInclude("smoke"),
			),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("tagged test"),
		axiom.WithCaseMeta(
			axiom.WithMetaTag("smoke"),
		),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Step("run", func() {})
	})
}

```

---

## Writing Your Own Plugin

Here is a minimal but realistic plugin that measures step duration:

```go
package timestats

import (
	"fmt"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

func Plugin() axiom.Plugin {
	return func(cfg *axiom.Config) {

		cfg.Runtime.EmitStepWrap(func(name string, next axiom.StepAction) axiom.StepAction {
			return func() {
				start := time.Now()
				next()
				fmt.Println("step", name, "took", time.Since(start))
			}
		})
	}
}

```

Usage:

```go
package main

import (
	"github.com/Nikita-Filonov/axiom"
)

var runner = axiom.NewRunner(
	axiom.WithRunnerPlugins(timestats.Plugin()),
)

```
