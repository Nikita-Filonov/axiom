# 📊 Stats Plugin (`teststats`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [What the plugin does](#what-the-plugin-does)
- [Installation](#installation)
- [Example](#example)

---

## Overview

Collects execution statistics for test cases by observing the Axiom runtime lifecycle.

The plugin records aggregated information about each test case, including retries, duration, final status, and
metadata, without affecting test execution.

The plugin does not control test flow — it only observes and records results.

---

## What the plugin does

At runtime, the plugin:

- tracks how many times a test case was executed (attempts)
- measures total execution duration
- determines the final test status:
    - passed
    - failed
    - skipped
    - flaky (passed after retries)
- captures test metadata and timestamps
- aggregates results into an in-memory statistics structure

---

## Installation

The plugin is distributed as a regular Go module and installed using standard Go tooling.

Add the plugin dependency using `go get`:

```shell
go get github.com/Nikita-Filonov/axiom/plugins/teststats
```

This will add the plugin to your `go.mod` file:

```text
require (
	github.com/Nikita-Filonov/axiom v0.3.0
	github.com/Nikita-Filonov/axiom/plugins/teststats v0.1.0
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
	"github.com/Nikita-Filonov/axiom/plugins/teststats"
)

func TestStatsExample(t *testing.T) {

	// Create a shared stats collector.
	// Results will be available after test execution.
	stats := teststats.NewStats()

	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			teststats.Plugin(stats),
		),

		// Enable retries to demonstrate flaky detection
		axiom.WithRunnerRetry(
			axiom.WithRetryTimes(2),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("stats example"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		cfg.Step("do work", func() {
			// Test logic goes here.
			// The stats plugin does not interfere with execution.
		})
	})

	// After the run, aggregated statistics are available.
	for _, result := range stats.Cases {
		_ = result.ID
		_ = result.Name
		_ = result.Attempts
		_ = result.Duration
		_ = result.Status
		_ = result.Meta
	}
}
```
