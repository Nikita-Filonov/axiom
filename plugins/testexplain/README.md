# 🧭 Explain Plugin (`testexplain`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [What the plugin does](#what-the-plugin-does)
- [Installation](#installation)
- [Example](#example)

---

## Overview

Captures a structured explanation of Axiom runner and config state.

The plugin is useful for debugging merged configuration, plugin order, registered hooks, runtime sinks, fixtures,
resources, retry policy, context values, and metadata.

---

## What the plugin does

At runtime, the plugin:

- records `ExplainConfig(cfg)` before test action execution
- stores explanations in an in-memory `Explainer`
- exposes snapshots without mutating recorded data

The package also provides `ExplainRunner(runner)` for inspecting runner-level configuration directly.

---

## Installation

The plugin is distributed as a regular Go module and installed using standard Go tooling.

Add the plugin dependency using `go get`:

```shell
go get github.com/Nikita-Filonov/axiom/plugins/testexplain
```

---

## Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testexplain"
)

func TestExplainExample(t *testing.T) {
	// Explainer stores copied snapshots of merged runtime configuration.
	// It is useful when a test behaves differently from what the declarations suggest.
	explainer := testexplain.NewExplainer()

	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			// The plugin records ExplainConfig(cfg) before the test action runs.
			// That means the explanation shows the effective Config after Runner
			// and Case options have been merged and plugins have been applied.
			testexplain.Plugin(explainer),
		),
	)

	c := axiom.NewCase(
		// This Case name will appear in the captured explanation, together with
		// metadata, runtime sinks, wraps, fixtures, resources, retry policy, and context.
		axiom.WithCaseName("explain example"),
	)

	// Running the case builds a Config and triggers the plugin's test wrapper.
	runner.RunCase(t, c, func(cfg *axiom.Config) {})

	// Snapshot returns copies, so callers can inspect or mutate the returned
	// explanations without changing what the Explainer has stored.
	for _, explanation := range explainer.Snapshot() {
		// Kind identifies what was explained. Case and Runtime show the merged
		// case data and executable runtime behavior for this Config.
		_ = explanation.Kind
		_ = explanation.Case
		_ = explanation.Runtime
	}
}
```
