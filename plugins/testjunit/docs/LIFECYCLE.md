# 🔄 JUnit Lifecycle

## 📑 Table of Contents

- [Publication](#publication)
- [Shared reporters](#shared-reporters)
- [Files and concurrency](#files-and-concurrency)
- [Partial runs](#partial-runs)

## Publication

```text
Plugin(options...)        -> prepare settings and writer; no Reporter mutation or file I/O
plugin installation      -> register output and recorder; no file I/O
case.start / case.skip    -> register attempt cleanup
attempt cleanup          -> record outcome -> publish XML
terminal Attempt         -> update the same record -> publish XML if changed
```

The attempt cleanup runs after the body, hooks, fixtures, child subtests, and later-registered cleanups.
This first snapshot is provisional: earlier-registered cleanup callbacks can still fail. The recorder subscribes to
`EmitAttemptSink` to confirm the native status and timing after completion. It does not register parent cleanup or
re-read the completed `testing.T`. Confirmation updates the existing collection entry, preserving skip reasons and
lifecycle error messages; it never adds a second entry. Lifecycle errors still take precedence over a passed status.

Final duration comes from `AttemptOutcome.StartedAt` and `CleanupAt`. It includes native cleanup, including the early
report publication, but excludes waiting for the parent to deliver the final result and runner resource cleanup.
A timing change also rewrites XML. Exclusions and native deselections have no record and are ignored by the sink.

For parallel attempts, the final result may arrive only at the parent boundary. Keeping the early publication is
therefore essential: completed attempts remain on disk if another case later crashes or hits the process timeout.
If the process stops before final delivery, the early status and duration remain. Configs driven manually through
raw events without Axiom execution also retain only their early snapshot.

Writes are synchronous. Every publication rewrites the full report; total output work grows roughly quadratically
with the number of attempts. Explicit final export remains available for large suites that prefer one write.

## Shared reporters

```go
reporter := testjunit.NewReporter()
first := testjunit.Plugin(
    testjunit.WithReporter(reporter),
    testjunit.WithSuiteName("first"),
    testjunit.WithOutputFile("junit-results/combined.xml"),
)
second := testjunit.Plugin(
    testjunit.WithReporter(reporter),
    testjunit.WithSuiteName("second"),
)
```

Once `first` is installed, both plugins update `combined.xml`. Output registration belongs to the reporter and lasts
for its lifetime. Constructing a plugin alone does not register its output.
Multiple output paths are supported; every publication updates all registered paths.
Installation does not publish existing results; the next completed attempt or status correction includes them.

- One default reporter is created per `Plugin(...)` call, not per attempt.
- Reusing the same plugin shares its reporter.
- Installing the same reporter twice on one Axiom Config records the attempt once; the first suite name wins.
- A repeated installation can add another output path without adding another recorder.
- Repeated registration of the same absolute cleaned output path uses the first installed error handler.
- Separate reporters writing the same path are unsupported. Symlink aliases are not deduplicated.
- A reporter's zero value works; do not copy it after first use.

## Files and concurrency

`Reporter.WriteFile` serializes automatic and manual exports on that reporter. It acquires the export lock
before taking the result snapshot, preventing an older concurrent export from replacing a newer snapshot.
The collection lock is released before encoding and file I/O. Error handlers run after export locks are released.

```text
lock export -> snapshot -> encode into sibling temporary file -> close -> rename -> unlock
```

An unsuccessful write before rename leaves the previous report intact. Replacement uses `os.Rename` and inherits
its platform/filesystem guarantees; this is not a power-loss durability guarantee. Temporary files are removed.

`Reporter.Write` requires separate writers or caller-provided synchronization. Different processes must use different
output paths; process-local locks do not merge reports from separate Go test binaries.

## Partial runs

| Situation                                | Automatic output                               |
|------------------------------------------|------------------------------------------------|
| No attempt completes                     | No new file                                    |
| Compilation/startup failure before Axiom | No new file                                    |
| Process stops after some publications    | Last successfully published snapshot           |
| Late failure reconciled normally         | Updated failed testcase                        |
| Process stops before reconciliation      | Earlier status may remain                      |
| Output write fails                       | Handler called; later publications can recover |

A valid XML file does not prove that the entire test run completed. In-flight attempts and unobserved failures are
absent. Keep the original `go test` exit code alongside JUnit results and avoid reusing stale output directories.
