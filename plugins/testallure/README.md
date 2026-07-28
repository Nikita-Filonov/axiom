# 🟣 Allure Plugin (`testallure`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [What the plugin does](#what-the-plugin-does)
- [Installation](#installation)
- [Example](#example)

---

## Overview

Generates Allure reports by projecting Axiom runtime events into the Allure execution model using
the official [`allure-framework/allure-go`](https://github.com/allure-framework/allure-go)
`commons/gotest` integration.

The plugin integrates with the Axiom runtime and automatically maps:

- test execution → Allure tests
- steps → Allure steps
- metadata → Allure labels, tags, severity, issue and TMS links
- artefacts → Allure attachments

The plugin does not change test logic — it only observes and decorates execution.

---

## What the plugin does

At runtime, the plugin:

- wraps each test attempt in `allure.Wrap(...)`
- wraps each `cfg.Step(...)` in an Allure step
- reports each `cfg.Setup(...)` and `cfg.Teardown(...)` as an Allure step
- converts Axiom metadata into Allure options
- attaches emitted artefacts to the current test

The official `commons/gotest` API uses an explicit per-test Allure context. The plugin keeps that
context isolated inside each Axiom attempt, including parallel cases and retries.

Steps within one case must execute synchronously. Parallel cases are supported, but concurrent
steps started by different goroutines within the same case are not: `commons/gotest` represents
step nesting as one stack per test result. All test goroutines must also finish before the test
action returns; events emitted after the test has completed cannot be attached to the closed
result.

`commons/gotest` does not expose high-level before/after fixture helpers. Setup and teardown calls
therefore remain visible in the report as regular steps with their original names.

`Meta.Issues` and `Meta.TestCases` are emitted as links with the standard `issue` and `tms` types.
Their short identifiers are used as both the link name and raw URL, allowing Allure link patterns
to expand them into full URLs while generating the report.

---

## Installation

The plugin is distributed as a regular Go module and installed using standard Go tooling.

Add the plugin dependency using `go get`:

```shell
go get github.com/Nikita-Filonov/axiom/plugins/testallure
```

This will add the plugin to your `go.mod` file:

```text
require (
	github.com/Nikita-Filonov/axiom v1.7.0
	github.com/Nikita-Filonov/axiom/plugins/testallure v0.20.0
)
```

Each plugin is versioned independently from the Axiom core.

---

## Example

```go
package example_test

import (
	"encoding/json"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testallure"
)

func TestAllureExample(t *testing.T) {

	// Enable Allure reporting
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testallure.Plugin(),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("user can login"),

		// Test metadata is automatically mapped to Allure
		axiom.WithCaseMeta(
			axiom.WithMetaEpic("authentication"),
			axiom.WithMetaFeature("login"),
			axiom.WithMetaStory("valid credentials"),
			axiom.WithMetaSeverity(axiom.SeverityCritical),
			axiom.WithMetaTag("smoke"),
			axiom.WithMetaLabel("component", "auth-service"),
		),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Setup("prepare test data", func() {
			// Appears in Allure as a step
		})

		cfg.Step("prepare request", func() {
			// This step appears as an Allure step
		})

		cfg.Step("send request", func() {
			// Nested execution is automatically tracked
		})

		cfg.Step("validate response", func() {

			// Emit an artefact into the runtime.
			// The Allure plugin observes this event and attaches it to the test.
			payload, _ := json.Marshal(map[string]any{
				"status": "ok",
				"user":   "demo",
			})

			artefact, _ := axiom.NewJSONArtefact("response.json", payload)
			cfg.Artefact(artefact)
		})

		cfg.Teardown("cleanup test data", func() {
			// Appears in Allure as a step
		})
	})
}
```

By default, results are written to `./allure-results`. Set `ALLURE_RESULTS_DIR` to select another
directory.
