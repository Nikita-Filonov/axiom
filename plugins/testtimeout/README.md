# ⏱ Timeout Plugin (`testtimeout`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [What the plugin does](#what-the-plugin-does)
- [Configuration](#configuration)
- [Environment variables](#environment-variables)
- [Semantics and caveats](#semantics-and-caveats)
- [Installation](#installation)
- [Example](#example)

---

## Overview

Enforces a per-case wall-clock deadline. When a case runs longer than the configured budget the plugin fails it with a
readable message instead of letting it hang until the global `go test -timeout` kills the whole binary.

The plugin decorates execution through a test wrap. It does not change test logic and is a no-op when no timeout is
configured, so it is safe to install unconditionally.

---

## What the plugin does

At runtime, for every case, the plugin:

- runs the test body under a wall-clock timer
- on timeout, marks the case as failed with a clear message (`testtimeout: case exceeded <budget>`)
- attaches a full goroutine dump as an artefact (enabled by default) so the hang can be diagnosed
- optionally gives the attempt's contexts the same deadline and cancels them when the attempt finishes
- re-raises any panic from the body on the test goroutine, preserving Axiom's lifecycle panic handling

Without `WithContextDeadline()`, a case that finishes within the budget has no observable change.

---

## Configuration

The plugin is configured with functional options:

- `WithTimeout(d)` — the per-case wall-clock budget. A non-positive value disables the plugin.
- `WithContextDeadline()` — give the attempt's Raw, DB, MQ, and RPC contexts the same deadline as `WithTimeout(d)`; disabled by default.
- `WithoutGoroutineDump()` — do not attach the goroutine dump artefact on timeout.
- `WithMessage(msg)` — override the default failure message.
- `ConfigFromEnv()` — read the budget from the environment.

---

## Environment variables

- `AXIOM_TEST_TIMEOUT` — per-case budget parsed with `time.ParseDuration` (for example `5s`, `500ms`).

An empty or invalid value leaves the current timeout unchanged.

```shell
export AXIOM_TEST_TIMEOUT=10s
```

---

## Semantics and caveats

- `WithTimeout(d)` marks the attempt as failed after `d`. Its goroutine may keep running.
- `WithContextDeadline()` gives `Raw`, `DB`, `MQ`, and `RPC` that same deadline, measured from the start of the attempt
  (after any parallel-test pause). Earlier parent deadlines still apply.
- The contexts are canceled when the attempt finishes or times out. This is a limit for the whole attempt, not a fresh
  timeout for each DB, MQ, or RPC call. Operations that ignore cancellation may keep running.

---

## Installation

The plugin is distributed as a regular Go module and installed using standard Go tooling.

```shell
go get github.com/Nikita-Filonov/axiom@v1.19.0 github.com/Nikita-Filonov/axiom/plugins/testtimeout@v0.14.0
```

This will add the plugin to your `go.mod` file:

```text
require (
	github.com/Nikita-Filonov/axiom v1.19.0
	github.com/Nikita-Filonov/axiom/plugins/testtimeout v0.14.0
)
```

Each plugin is versioned independently from the Axiom core.

---

## Example

```go
package example_test

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testtimeout"
)

func TestTimeoutExample(t *testing.T) {

	// Fail any case that runs longer than 5 seconds.
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testtimeout.Plugin(
				testtimeout.WithTimeout(5*time.Second),
				testtimeout.WithContextDeadline(), // optional: cancel attempt contexts at the same deadline
				testtimeout.ConfigFromEnv(), // optional: AXIOM_TEST_TIMEOUT overrides the budget
			),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("slow operation"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Step("call service", func() {
			// If this exceeds the budget, the case fails with a goroutine dump attached.
		})
	})
}
```
