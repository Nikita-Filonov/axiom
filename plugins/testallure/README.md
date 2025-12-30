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
`dailymotion/allure-go`.

The plugin integrates with the Axiom runtime and automatically maps:

- test execution → Allure tests
- steps → Allure steps
- metadata → Allure labels, tags, severity
- artefacts → Allure attachments

The plugin does not change test logic — it only observes and decorates execution.

---

## What the plugin does

At runtime, the plugin:

- wraps each test attempt in `allure.Test(...)`
- wraps each `cfg.Step(...)` in `allure.Step(...)`
- converts Axiom metadata into Allure options
- attaches emitted artefacts to the current test

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
	github.com/Nikita-Filonov/axiom v0.3.0
	github.com/Nikita-Filonov/axiom/plugins/testallure v0.1.0
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
	})
}
```
