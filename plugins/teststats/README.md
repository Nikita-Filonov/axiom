# 📊 Stats Plugin (`teststats`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Example](#example)
- [Attempts](#attempts)
- [Runs](#runs)
- [Summary](#summary)
- [Filtering](#filtering)
- [Lifecycle and concurrency](#lifecycle-and-concurrency)

---

## Overview

Records the outcome of every Axiom case attempt without affecting test execution.

- one `Attempt` per executed or policy-skipped attempt, with timing that includes cleanups
- retries of one `RunCase` invocation grouped into a `Run` with a derived status, including flaky
- counts of runs and attempts by status
- safe for concurrent use, including parallel cases

---

## Installation

The plugin is distributed as a regular Go module and installed using standard Go tooling:

```shell
go get github.com/Nikita-Filonov/axiom github.com/Nikita-Filonov/axiom/plugins/teststats
```

This development version relies on `Config.Execution` and the `case.skip` event from the Axiom core in the same
checkout. Use a Go workspace while developing both modules:

```shell
go work init . ./plugins/teststats
```

Release the core first, then update this module's core requirement and release the plugin.

---

## Example

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/teststats"
)

func TestLogin(t *testing.T) {
	stats := teststats.NewStats()
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(teststats.Plugin(stats)),
		axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
	)

	t.Cleanup(func() {
		summary := stats.Summary()
		fmt.Println("runs:", summary.Runs.Total, "flaky:", summary.Runs.Flaky)
		for _, run := range stats.Runs() {
			fmt.Println(run.Name, run.Status, len(run.Attempts), run.Elapsed)
		}
	})

	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("user can log in")), func(cfg *axiom.Config) {
		// Test logic.
	})
}
```

Read results after the attempts have finished: after `RunCase` for sequential cases, and in a parent `t.Cleanup` or
`AfterAll` hook for parallel ones.

---

## Attempts

A failure followed by a successful retry produces two attempts:

| RunID | CaseID | Number | Status | Start    | End      |
|-------|--------|-------:|--------|----------|----------|
| run-A | login  |      1 | failed | 10:00:00 | 10:00:02 |
| run-A | login  |      2 | passed | 10:00:03 | 10:00:04 |

- `RunID` comes from `Config.Execution.ID` and is shared by the retries of one `RunCase` invocation. Running the same
  case twice produces two runs, even with the same name or `Case.ID`.
- `Number` is the one-based attempt number. A case skipped before its parallel retries start is recorded as attempt one.
- `CaseID` and `Name` come from the declared Case; `TestName` is the full Go test name.
- `Meta` is the merged metadata when the attempt started, including changes made by later plugins.
- `Status` is `passed`, `failed`, or `skipped`. A failure takes precedence over a skip.
- `SkipReason` is set for policy skips. Go does not expose the reason passed directly to `t.Skip`.
- `Error` is the first Axiom lifecycle failure: a panic in the test body, a step, setup or teardown, or a fixture
  failure. Go does not expose the messages passed to `t.Error` or `t.Fatal`.
- `Start` excludes the wait for parallel scheduling. `End` follows hooks, fixture cleanup, child subtests, and
  `t.Cleanup` callbacks. `Duration` uses the monotonic clock.

`Stats.Attempts()` returns copies ordered by start time.

---

## Runs

`Stats.Runs()` groups attempts by `RunID`, ordered by each run's first start:

```text
ID:       run-A
Status:   flaky
Attempts: [1 failed, 2 passed]
Duration: 3s  // sum of attempt durations
Elapsed:  4s  // first start to last end, including the retry delay
```

| Attempts in order | Run status |
|-------------------|------------|
| passed            | passed     |
| skipped           | skipped    |
| failed → passed   | flaky      |
| failed → failed   | failed     |
| failed → skipped  | failed     |

A flaky run is still a failed Go test; the plugin never changes the exit status of `go test`.

---

## Summary

`Stats.Summary()` counts runs by their derived status and attempts by outcome. For the history above:

```text
Runs:     Total=1 Flaky=1
Attempts: Total=2 Passed=1 Failed=1
```

---

## Filtering

`Filter` returns a new collector with the matching attempts and never changes the source:

```go
smoke := stats.Filter(func(a teststats.Attempt) bool {
	return slices.Contains(a.Meta.Tags, "smoke")
})
fmt.Println(smoke.Summary().Runs.Failed)
```

The predicate receives a copy and runs without holding the collector's lock, so it may query the source collector.
Filter by case identity or metadata to keep whole runs. Filtering by attempt status drops the attempts that make a
run flaky.

---

## Lifecycle and concurrency

The plugin listens to `case.start` and `case.skip` and registers a `t.Cleanup` on the attempt's test. Axiom registers
no cleanup earlier, so the attempt is recorded after everything it registers. A planning Config produces no record,
and neither do cases excluded by `go test -run`. Installing the same collector on both the Runner and the Case
records each attempt once.

Go marks a panicking `t.Cleanup` callback or a detected data race as a failure only after the attempt's cleanups have
run. The plugin then marks the recorded attempt failed from the parent test's cleanup, so final results are available
once the parent test has finished.

Runner resource cleanup is outside every attempt and does not affect its duration or status.

All `Stats` methods are safe for concurrent use and return copies.
