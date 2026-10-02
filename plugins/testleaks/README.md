# 🧵 Leak Checks Plugin (`testleaks`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [Goroutine checks](#goroutine-checks)
- [Resource tracking](#resource-tracking)
- [Configuration and limits](#configuration-and-limits)
- [Installation](#installation)
- [Example](#example)

---

## Overview

`testleaks` checks each executed Axiom attempt for goroutines still running and
explicitly tracked resources still open after the attempt body and its
later-registered `testing.T.Cleanup` callbacks. It reports a failure through
the attempt's `*testing.T`. Retry attempts are checked separately; a case
skipped during planning does not run a check.

The plugin registers its cleanup callback before invoking the attempt body.
Cleanups registered by that body run first, in Go's reverse registration order.
The default 200 ms grace period gives workers a chance to exit before the final
snapshot.

---

## Goroutine checks

The test wrap applies an attempt-specific pprof label while running the body,
hooks, and fixtures. Goroutines started there inherit the label, so concurrent
attempts can be inspected independently. On failure, the report includes the
remaining goroutine count and up to eight stack groups, with up to sixteen
frames per group.

Use `WithIgnoreFunction(name)` for a deliberately long-lived goroutine. A group
is excluded when a stack frame's fully qualified function name exactly matches
`name`. `WithoutGoroutines()` disables this check while keeping resource
tracking active. The plugin uses Go's regular goroutine profile; no
`GOEXPERIMENT` setting is required.

---

## Resource tracking

Go does not provide a per-test inventory of open files, connections, timers,
and similar resources. Register resources explicitly with `Track(cfg, name)`
and call the returned handle's `Release` after cleanup completes. The report
includes the name and registration site of each unreleased resource. `Release`
is safe to call repeatedly or concurrently.

For an `io.ReadCloser`, `TrackReadCloser(cfg, name, value)` returns a wrapper
whose `Close` also releases its handle. It records that `Close` was called even
if the underlying `Close` returns an error; the caller must handle that error.
The plugin never closes resources for you. Runner-owned resources that outlive
an attempt should not be tracked on that attempt.

---

## Configuration and limits

Pass options to `testleaks.Plugin(...)`:

- `WithGracePeriod(d)` waits up to `d` after cleanup; zero checks immediately.
- `WithIgnoreFunction(name)` excludes goroutine stack groups containing that
  exact function name. It may be supplied more than once.
- `WithoutGoroutines()` checks only explicitly tracked resources.

The goroutine check sees only goroutines that inherit the label. It cannot
attribute work started by an existing worker pool, a callback after the wrap
returns, or code that replaces pprof labels. A goroutine alive at the deadline
may still exit later; this is a leak signal, not proof of permanence. Cleanup
callbacks registered before the plugin's callback may run after the check.

The check calls `t.Errorf` during `testing.T.Cleanup`. Go's test result reflects
that failure, but a reporter that finalizes when its test wrap returns may have
already finished. This matters when combining `testleaks` with `testallure`.

---

## Installation

The plugin is a separate Go module:

```shell
go get github.com/Nikita-Filonov/axiom/plugins/testleaks
```

When developing against the core module in this repository, include both
modules in a Go workspace, as with the other plugins.

---

## Example

This example checks a worker goroutine, a response body, and a manually tracked
resource. Its cleanups run before the leak check. The code is also kept as a
[runnable test](example_test.go).

```go
package testleaks_test

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testleaks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeakChecksExample(t *testing.T) {
	// Install the checker for every attempt executed by this Runner.
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testleaks.Plugin(
				testleaks.WithGracePeriod(300 * time.Millisecond),
			),
		),
	)

	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("read response")), func(cfg *axiom.Config) {
		// Workers started inside the attempt inherit its pprof label.
		stop := make(chan struct{})
		stopped := make(chan struct{})
		go func() {
			defer close(stopped)
			<-stop
		}()
		cfg.T().Cleanup(func() {
			close(stop)
			<-stopped
		})

		// Closing a tracked ReadCloser releases its resource handle.
		body := testleaks.TrackReadCloser(cfg, "response body", io.NopCloser(strings.NewReader("ok")))
		cfg.T().Cleanup(func() {
			assert.NoError(cfg.T(), body.Close())
		})
		payload, err := io.ReadAll(body)
		require.NoError(cfg.T(), err)
		assert.Equal(cfg.T(), "ok", string(payload))

		// Track any other resource until its actual cleanup completes.
		subscription := testleaks.Track(cfg, "temporary subscription")
		cfg.T().Cleanup(subscription.Release)
	})
}
```
