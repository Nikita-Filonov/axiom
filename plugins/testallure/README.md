# 🟣 Allure Plugin (`testallure`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [What the plugin does](#what-the-plugin-does)
  - [Execution model](#execution-model)
  - [Current limitations](#current-limitations)
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

The plugin projects Axiom runtime events into Allure:

- wraps each test attempt in `allure.Wrap(...)`
- wraps each `cfg.Step(...)` in an Allure step
- reports each `cfg.Setup(...)` and `cfg.Teardown(...)` as an Allure step
- converts Axiom metadata into Allure options
- attaches emitted artefacts to the current test

### Execution model

The official `commons/gotest` API uses an explicit per-test Allure context. The plugin keeps that
context isolated inside each Axiom attempt, including parallel cases and retries.

Every retry attempt produces a separate Allure result. Attempts with the same Axiom Case ID share
the same Allure test case and history IDs, allowing Allure to group them as retries.

### Current limitations

- **Concurrent steps:** parallel Cases are supported, but Steps within one Case must execute
  synchronously. `commons/gotest` maintains one Step stack per test result.
- **Background goroutines:** all test goroutines must finish before the test action returns. Events
  emitted after completion cannot be attached to the closed result.
- **Fixtures:** `commons/gotest` does not expose high-level before/after fixture helpers. Setup and
  teardown therefore appear as regular Steps.
- **Pre-execution skips:** a runtime skip is reported as an Allure `skipped` result, but
  `axiom.WithCaseSkip(...)` runs before the Allure lifecycle and produces no result.
- **Issue and TMS links:** `Meta.Issues` and `Meta.TestCases` are emitted with the standard `issue`
  and `tms` types. Their identifiers are stored as raw URLs because `allure-go` does not implement
  link-pattern expansion; vanilla Allure reports therefore do not turn them into full links.

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
