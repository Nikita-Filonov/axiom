# 🏷 Tags Plugin (`testtags`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [What the plugin does](#what-the-plugin-does)
- [Configuration](#configuration)
- [Environment variables](#environment-variables)
- [Installation](#installation)
- [Example](#example)

---

## Overview

Filters test execution based on metadata tags using include and exclude rules.

The plugin evaluates test metadata during installation and excludes cases that do not match the filter. Explicit
test disabling remains a separate Axiom skip decision.

---

## What the plugin does

At runtime, the plugin:

- reads tags from test metadata (`cfg.Meta.Tags`)
- normalizes tags (trimmed, lowercased)
- applies include and exclude rules
- excludes tests when rules do not match

If a rule fails, the plugin sets:

```go
cfg.Attempt.Exclude("...")

```

Excluded tests remain visible as native Go skips, but produce no Allure, JUnit, or teststats test results. Lifecycle
consumers can observe `Attempt.Decision.Kind == DecisionExclude` or `EventTypeCaseExcluded`; they need not inspect
reason strings. A matching case with `WithCaseSkip` remains selected and is reported as skipped. See
[execution](../../docs/execution).

---

## Configuration

The plugin can be configured via code, environment variables, or both.

### Include / exclude rules

- `Include` — only tests with at least one matching tag are executed
- `Exclude` — tests with any matching tag are excluded

Rules are evaluated in the following order:

- exclude rules
- include rules

---

## Environment variables

The plugin supports environment-driven configuration:

- `AXIOM_TEST_TAGS_INCLUDE`
- `AXIOM_TEST_TAGS_EXCLUDE`

Values are comma-separated lists of tags.

Example:

```shell
export AXIOM_TEST_TAGS_INCLUDE=smoke,critical
export AXIOM_TEST_TAGS_EXCLUDE=slow
```

---

## Installation

The plugin is distributed as a regular Go module and installed using standard Go tooling.

Add the plugin dependency using `go get`:

```shell
go get github.com/Nikita-Filonov/axiom@v1.18.0 github.com/Nikita-Filonov/axiom/plugins/testtags@v0.30.0
```

This will add the plugin to your `go.mod` file:

```text
require (
	github.com/Nikita-Filonov/axiom v1.18.0
	github.com/Nikita-Filonov/axiom/plugins/testtags v0.30.0
)
```

Each plugin is versioned independently from the Axiom core.

---

## Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testtags"
)

func TestTagsExample(t *testing.T) {

	// Enable tag-based filtering.
	// Only tests tagged with "smoke" will be executed.
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testtags.Plugin(
				testtags.WithConfigInclude("smoke"),
				testtags.ConfigFromEnv(), // optional: merge env-based rules
			),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("tagged test"),

		// Tags are defined via test metadata
		axiom.WithCaseMeta(
			axiom.WithMetaTag("smoke"),
			axiom.WithMetaTag("auth"),
		),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		// If the tag filter does not match,
		// this test will be excluded before execution.
		cfg.Step("run", func() {
			// Test logic goes here
		})
	})
}
```
