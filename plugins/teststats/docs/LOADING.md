# 📘 Loading results

## 📑 Table of Contents

- [Read from another process](#read-from-another-process)
- [Result and errors](#result-and-errors)
- [Use in a notifier](#use-in-a-notifier)
- [Which files are loaded](#which-files-are-loaded)

---

[Plugin overview](../README.md)

## Read from another process

Save this as `read.go` in a module that depends on `teststats`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/Nikita-Filonov/axiom/plugins/teststats"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run read.go <results-directory>")
		os.Exit(2)
	}
	result, err := teststats.Load(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	counts := result.Stats.Summary().Runs
	fmt.Printf("runs=%d passed=%d failed=%d skipped=%d flaky=%d incomplete=%t\n",
		counts.Total,
		counts.Passed,
		counts.Failed,
		counts.Skipped,
		counts.Flaky,
		result.Incomplete(),
	)
	for _, unfinished := range result.Unfinished {
		fmt.Fprintf(os.Stderr, "%s: %s\n", unfinished.Path, unfinished.State)
	}
	for _, issue := range result.Issues {
		fmt.Fprintln(os.Stderr, issue)
	}
}
```

```shell
go run read.go ./axiom-results
```

For the [example files](JSON-FORMAT.md):

```text
runs=1 passed=1 failed=0 skipped=0 flaky=0 incomplete=false
```

## Result and errors

`Load(dir)` returns `(*LoadResult, error)`. When `err == nil`, `result.Stats` is always available.

| Result         | Contents                                                                                      |
|----------------|-----------------------------------------------------------------------------------------------|
| `Stats`        | Completed outcomes: `finished` and `finalized` attempts; queries use the existing `Stats` API |
| `Unfinished`   | Valid `running` and `finished` snapshots, with relative path, `AttemptState`, and `Attempt`   |
| `Issues`       | Per-file/directory errors and missing attempt ranges; each has `Path` and `Err`               |
| `Incomplete()` | True when there are issues, unfinalized attempts, or no usable attempts                       |

| Files on disk                                          | Stats                       | Other result                            |
|--------------------------------------------------------|-----------------------------|-----------------------------------------|
| Attempt 1 finalized/failed, attempt 2 finalized/passed | One flaky run, two attempts | `Incomplete() == false`                 |
| Attempt 1 finished/passed                              | One passed run              | One `Unfinished` entry; incomplete      |
| Attempt 1 running                                      | No completed attempts       | One `Unfinished` entry; incomplete      |
| Attempt 1 malformed, attempt 2 finalized/passed        | Only attempt 2              | Issue for attempt 1; incomplete         |
| Attempts 1 and 3 present                               | Both valid outcomes         | Issue for missing attempt 2; incomplete |
| Existing empty directory                               | Empty Stats                 | No issues; incomplete                   |
| Missing/unreadable root or unreadable process index    | No result                   | Non-nil `err`                           |

`LoadIssue` implements `error` and `Unwrap`, so `errors.Is(issue, fs.ErrPermission)` works.
Issues do not stop the rest of the directory from loading. No files are written or repaired.

## Use in a notifier

Use `result.Stats.Summary().Runs` to count cases, or `.Attempts` to count every retry.

| Notification field                     | Source                                                               |
|----------------------------------------|----------------------------------------------------------------------|
| Passed                                 | `counts.Passed`                                                      |
| Failed                                 | `counts.Failed`                                                      |
| Skipped                                | `counts.Skipped`                                                     |
| Flaky                                  | `counts.Flaky` — separate from Passed and Failed                     |
| Partial data                           | `result.Incomplete()`; show even when all observed counts are passed |
| Test-process outcome                   | Original `go test` exit code from CI                                 |
| CI run time, environment, report links | CI/application configuration                                         |

For a notifier that currently reads `TEST_PASSED`, `TEST_FAILED`, `TEST_SKIPPED`, and `TEST_BROKEN`, replace those
count inputs with one results directory. Axiom has no `broken` status: lifecycle failures count as failed attempts.
Add Flaky and Partial fields to the message; file-reading issues are diagnostics, not broken tests.

If `Load` returns an error, report statistics as unavailable. A zero test exit code plus `Incomplete() == true`
still means the saved statistics are incomplete. A nonzero test exit code remains a failure even if all saved
attempts passed. Keep the [original test exit code](JSON-OUTPUT.md#preserve-the-test-exit-code-in-ci) when the notifier exits.

Summed attempt durations are not CI wall time: parallel cases can overlap. Keep measuring CI run time separately.

## Which files are loaded

```text
<results>/processes/<process-id>/runs/<hex-run-id>/attempt-<number>.json
```

The loader reads this layout, checks version 1, required fields, states, outcomes, and identity against the path.
Malformed JSON, duplicate keys, unknown fields or versions, and null values are rejected as issues. Temporary files
and unrelated files are ignored. Symbolic links below the supplied root are not followed.

`run.json` is ignored, including missing, stale, or malformed snapshots. Runs and counts are rebuilt from attempts.
Gaps before the highest observed attempt number are reported without discarding the remaining attempts.

`Attempt.ProcessID` and `Run.ProcessID` preserve the source namespace. Grouping uses `(ProcessID, RunID)`, so identical
run IDs in different processes do not merge. Original run IDs are unchanged. Empty metadata collections omitted
from JSON load as nil collections.

Read after all writers exit. `Incomplete() == false` means the observed files are valid and finalized; it cannot
detect an entirely missing process, a missing last attempt, or tests that never started. There is no launch manifest
yet, so this check cannot establish that the whole CI invocation completed.
