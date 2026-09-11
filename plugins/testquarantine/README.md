# 🧟 Quarantine Plugin (`testquarantine`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [What the plugin does](#what-the-plugin-does)
- [Semantics: why pre-emptive skip](#semantics-why-pre-emptive-skip)
- [Configuration](#configuration)
- [Environment variables](#environment-variables)
- [Installation](#installation)
- [Example](#example)

---

## Overview

Quarantines known-flaky cases so they do not gate a suite. A quarantined case is skipped before execution with a
`quarantined: <reason>` skip reason, which keeps it visible in reports without failing the run.

Selection is tag-based by default (`quarantine`) and can be replaced with a custom predicate.

---

## What the plugin does

At runtime, for every case, the plugin:

- decides whether the case is quarantined (by metadata tag or a custom predicate)
- if quarantined, sets a skip with reason `quarantined: <reason>`:

```go
cfg.Skip = cfg.Skip.Join(axiom.NewSkip(axiom.SkipBecause("quarantined: flaky")))
```

Skipped cases are still visible to other plugins (stats, reporting, etc.).

With `WithRun(true)` (or the environment toggle) quarantined cases are executed instead of skipped — useful for a
non-gating job that still observes whether they pass.

---

## Semantics: why pre-emptive skip

Quarantine is a **deliberate pre-emptive skip**, not a post-hoc "fail then downgrade to skip".

Go cannot un-fail a test once it has failed: assertions call `t.Fail`/`t.FailNow`, which propagate to the parent test
and cannot be reverted. Even Axiom's own retry keeps a failed process failed. Because of that, the only robust way to
stop a known-flaky case from gating a suite is to skip it before it runs, with a recorded reason, while keeping it
visible.

To still observe a quarantined case (for example in a nightly, non-gating job), enable `WithRun(true)` so it executes
normally.

---

## Configuration

The plugin is configured with functional options:

- `WithTag(tag)` — metadata tag that marks a case as quarantined (default `quarantine`).
- `WithReason(reason)` — fallback skip reason (default `flaky`).
- `WithRun(run)` — execute quarantined cases instead of skipping them.
- `WithPredicate(fn)` — replace tag-based selection with custom logic returning `(reason, quarantined)`.
- `ConfigFromEnv()` — read the run toggle from the environment.

---

## Environment variables

- `AXIOM_TEST_QUARANTINE_RUN` — execute quarantined cases when set to `1`/`true`/`yes`/`on`; skip them on
  `0`/`false`/`no`/`off`.

An empty or unrecognized value leaves the current setting unchanged.

```shell
export AXIOM_TEST_QUARANTINE_RUN=1
```

---

## Installation

The plugin is distributed as a regular Go module and installed using standard Go tooling.

```shell
go get github.com/Nikita-Filonov/axiom/plugins/testquarantine
```

This will add the plugin to your `go.mod` file:

```text
require (
	github.com/Nikita-Filonov/axiom v1.10.0
	github.com/Nikita-Filonov/axiom/plugins/testquarantine v0.1.0
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
	"github.com/Nikita-Filonov/axiom/plugins/testquarantine"
)

func TestQuarantineExample(t *testing.T) {

	// Cases tagged "quarantine" are skipped so they do not gate the suite.
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testquarantine.Plugin(
				testquarantine.WithReason("JIRA-123 flaky"),
				testquarantine.ConfigFromEnv(), // optional: AXIOM_TEST_QUARANTINE_RUN=1 runs them anyway
			),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("flaky login flow"),

		// The quarantine tag marks this case as known-flaky.
		axiom.WithCaseMeta(
			axiom.WithMetaTag("quarantine"),
		),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		// Skipped before execution unless quarantined runs are enabled.
		cfg.Step("run", func() {
			// Test logic goes here
		})
	})
}
```
