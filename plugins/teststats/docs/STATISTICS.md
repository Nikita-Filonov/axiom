# 📘 Statistics in memory

## 📑 Table of Contents

- [Complete example](#complete-example)
- [What each method returns](#what-each-method-returns)
- [A failed attempt followed by a pass](#a-failed-attempt-followed-by-a-pass)
- [Filter results](#filter-results)

---

[Plugin overview](../README.md)

## Complete example

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
		axiom.WithRunnerPlugins(
			teststats.Plugin(teststats.WithStats(stats)),
		),
	)

	// Register before RunCase: this runs after the plugin's parent cleanups.
	t.Cleanup(func() {
		summary := stats.Summary()
		fmt.Println("runs:", summary.Runs.Total, "passed:", summary.Runs.Passed)
	})

	login := axiom.NewCase(
		axiom.WithCaseID("C-1"),
		axiom.WithCaseName("login"),
	)
	runner.RunCase(t, login, func(cfg *axiom.Config) {
		// Test logic and assertions.
	})
}
```

After this passing case:

```text
runs: 1 passed: 1
```

The same reporting cleanup works with parallel cases. Reading immediately after a parallel `RunCase` can return
empty or partial results. See [when results are recorded](LIFECYCLE.md#when-files-and-stats-change).

## What each method returns

| Method             | Result                                                              | Order                                                  |
|--------------------|---------------------------------------------------------------------|--------------------------------------------------------|
| `stats.Attempts()` | One [Attempt](../attempt.go) per completed execution or policy skip | Start time, then ProcessID, RunID, and attempt number  |
| `stats.Runs()`     | [Runs](../run.go) grouping retries by ProcessID and RunID           | Earliest start of each run; attempts ordered by number |
| `stats.Summary()`  | [Counts](../summary.go) for runs and attempts                       | `Total`, `Passed`, `Failed`, `Skipped`, `Flaky`        |

Methods are safe for concurrent use and return copies. Two `RunCase` calls create different runs even when the
Case ID and name are the same. Metadata is captured when the attempt starts.
`ProcessID` also keeps runs from different test binaries separate when [loading saved results](LOADING.md).

## A failed attempt followed by a pass

| Attempt | Status | Start    | End      | Duration |
|---------|--------|----------|----------|----------|
| 1       | failed | 10:00:00 | 10:00:02 | 2s       |
| 2       | passed | 10:00:03 | 10:00:04 | 1s       |

```text
Run:      Status=flaky  Duration=3s  Elapsed=4s
Runs:     Total=1  Flaky=1
Attempts: Total=2  Failed=1  Passed=1  Flaky=0
```

`Duration` sums attempt durations. `Elapsed` also includes the retry delay.

| Attempt outcomes | Run status |
|------------------|------------|
| passed           | passed     |
| skipped          | skipped    |
| failed → passed  | flaky      |
| failed → failed  | failed     |
| failed → skipped | failed     |

A flaky run still leaves the Go test failed. The plugin does not change the test exit code.

## Filter results

After the example case finishes:

```go
loginStats := stats.Filter(func(attempt teststats.Attempt) bool {
	return attempt.CaseID == "C-1"
})
fmt.Println(loginStats.Summary().Runs.Total) // 1
```

`Filter` returns a new collector. The predicate receives a copy and can query the original collector.
Filter by case identity or metadata to preserve retry history. Filtering by `StatusPassed` alone removes the
earlier failures needed to classify a run as flaky.
