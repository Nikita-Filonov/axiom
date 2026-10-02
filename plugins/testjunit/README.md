# 📄 JUnit XML Plugin (`testjunit`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Example](#example)
- [Configuration](#configuration)
- [Report contents](#report-contents)
- [Lifecycle and concurrency](#lifecycle-and-concurrency)
- [Exporting in CI](#exporting-in-ci)
- [Limitations](#limitations)

---

## Overview

`testjunit` exports Axiom case attempts as JUnit XML without changing Go's test
result.

- One result per executed or policy-skipped attempt, including retries.
- Results grouped by suite name, with counts, timing, and failure messages.
- A shared reporter for sequential or parallel cases.
- Export to an `io.Writer` or a file after tests finish.

The plugin is a separate Go module with its own recorder. Production code depends
only on Axiom core and the Go standard library; tests use `testify`.

---

## Installation

```shell
go get github.com/Nikita-Filonov/axiom/plugins/testjunit
```

Register `testjunit.Plugin(reporter, options...)` with
`axiom.WithRunnerPlugins(...)` or `axiom.WithCasePlugins(...)`.

---

## Example

This [runnable test](example_test.go) writes a report after a sequential case.
For parallel cases and final cleanup status, see [when to export](#when-to-export).

```go
package testjunit_test

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testjunit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJUnitReportExample(t *testing.T) {
	// The reporter stores results; Plugin configures how attempts are recorded.
	reporter := testjunit.NewReporter()
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testjunit.Plugin(reporter, testjunit.WithSuiteName("example/package")),
		),
	)

	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("request succeeds")), func(cfg *axiom.Config) {
		// Test the application here. Failures still affect go test normally.
		cfg.T().Log("request succeeded")
	})

	// Sequential RunCase calls already have a result to export.
	// To include late cleanup failures and parallel cases, export after
	// the parent test or m.Run finishes.
	path := filepath.Join(t.TempDir(), "junit.xml")
	require.NoError(t, reporter.WriteFile(path))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var report struct {
		Tests int `xml:"tests,attr"`
	}
	require.NoError(t, xml.Unmarshal(data, &report))
	assert.Equal(t, 1, report.Tests)
}
```

---

## Configuration

| API                                      | Purpose                                                                        |
| ---------------------------------------- | ------------------------------------------------------------------------------ |
| `testjunit.NewReporter()`                | Create an empty collector that stores results and exports XML.                 |
| `testjunit.Plugin(reporter, options...)` | Record attempts in the supplied reporter. Options belong to this installation. |
| `testjunit.WithSuiteName(name)`          | Set the suite name and testcase `classname`; the default is `axiom`.           |

- Different runners can share a reporter and use different suite names.
- Installing the same reporter twice on one Config records each attempt once,
  using the first installation's options.

---

## Report contents

### XML structure

| Element or attribute           | Value                                                                   |
| ------------------------------ | ----------------------------------------------------------------------- |
| `<testsuites>`                 | Report root with totals across all suites.                              |
| `<testsuite>`                  | One group per suite name.                                               |
| `<testcase>`                   | One executed or policy-skipped attempt.                                 |
| Testcase `name`                | Full Go test name, distinguishing retry attempts.                       |
| Testcase `classname`           | Configured suite name.                                                  |
| `tests`, `failures`, `skipped` | Counts on the root and each suite.                                      |
| `time`                         | Attempt duration in seconds; root and suite values sum those durations. |

- Suites and their cases follow attempt start order.
- XML control characters in names and messages are replaced. Attempts with the
  same resulting suite name are grouped together.
- An empty reporter produces a `<testsuites>` document with zero totals and no
  child suites.

### Outcomes

| Attempt outcome   | XML result                                                                                           |
| ----------------- | ---------------------------------------------------------------------------------------------------- |
| Passed            | A testcase without `<failure>` or `<skipped>`.                                                       |
| Failed            | `<failure>` with an available Axiom lifecycle message, or `test failed` for direct Go test failures. |
| Skipped           | `<skipped>` with the policy skip reason, or `skipped` when no reason is available.                   |

A lifecycle failure takes precedence over a skip, even when its message is empty.
A failed attempt followed by a successful retry produces both results; the
previous failure remains in the report.

---

## Lifecycle and concurrency

### Recording results

1. On `case.start` or `case.skip`, the recorder registers an attempt cleanup.
2. That cleanup records the outcome after the body, hooks, fixtures, child
   subtests, and cleanup callbacks registered after it.
3. The parent test's cleanup rechecks the outcome for late failures, including
   earlier-registered cleanups, cleanup panics, and race-detector failures.

Duration ends at the recorder's cleanup. It excludes callbacks that run later
and runner resource cleanup outside the attempt.

### When to export

| Export point                               | Results available                                                                                                         |
| ------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------- |
| After a sequential `RunCase`               | The recorded attempt; late status corrections may still be pending.                                                       |
| During test execution                      | Attempts collected so far; parallel cases may still be running.                                                           |
| In a parent test's cleanup                 | Final child outcomes **if the export callback was registered before running the cases**, so it runs after reconciliation. |
| After the parent test or `m.Run()` returns | Final outcomes, including late failures.                                                                                  |

### Concurrency

- Collection and snapshots are synchronized; a reporter can be shared by parallel cases.
- Concurrent exports need separate writers or a writer with its own synchronization.
- A Reporter's zero value is ready to use. Do not copy it after first use.

---

## Exporting in CI

### Writing a report

| Method                     | Behavior                                                                                  |
| -------------------------- | ----------------------------------------------------------------------------------------- |
| `reporter.Write(w)`        | Write a complete XML snapshot to an `io.Writer`.                                          |
| `reporter.WriteFile(path)` | Create parent directories, write a temporary file beside the destination, then rename it. |

Both methods return export errors to the caller. Replacing an existing file
with `WriteFile` depends on the platform's `os.Rename` behavior.

### Package-wide export

Keep the reporter at package scope and export after `m.Run()` in `TestMain`.
Preserve the test exit code and turn an export error into a failed job:

```go
package example_test

import (
	"log"
	"os"
	"testing"

	"github.com/Nikita-Filonov/axiom/plugins/testjunit"
)

var junit = testjunit.NewReporter()

func TestMain(m *testing.M) {
	code := m.Run()
	if err := junit.WriteFile("reports/junit.xml"); err != nil {
		log.Printf("JUnit export: %v", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}
```

Install `testjunit.Plugin(junit, testjunit.WithSuiteName("example/package"))`
on the Runner used by the package's tests. Use a distinct output path for each
package to prevent overwrites, and a distinct suite name to distinguish packages.

### Publishing the report

| CI system                                                                           | Integration                                                                                                                 |
| ----------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| [GitLab](https://docs.gitlab.com/ci/testing/unit_test_reports/)                     | Upload `reports/junit.xml` with `artifacts:reports:junit`; use `artifacts:when: always` to retain reports from failed jobs. |
| [Jenkins](https://plugins.jenkins.io/junit/)                                        | Publish the file with the JUnit plugin.                                                                                     |
| [GitHub Actions](https://docs.github.com/en/actions/tutorials/store-and-share-data) | Upload the file as a workflow artifact. Displaying JUnit results in the GitHub UI requires a separate report publisher.     |

---

## Limitations

- **Failure text:** Go does not expose messages passed directly to `t.Error` or
  `t.Fatal` to plugins. Those failures use the generic `test failed` message.
- **Logs:** stdout, stderr, and arbitrary test logs are not captured.
- **Source locations:** reports have no `file` attribute because Axiom does not
  record a case's source file. CI features requiring that attribute are unavailable.
- **Test selection:** tests that never enter Axiom and cases excluded by
  `go test -run` are absent from the report.
- **Process crashes:** a crash before the export call prevents XML generation.
