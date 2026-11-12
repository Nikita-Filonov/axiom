# 📘 Lifecycle and partial results

## 📑 Table of Contents

- [When files and Stats change](#when-files-and-stats-change)
- [Partial results](#partial-results)
- [File replacement](#file-replacement)
- [What no files can mean](#what-no-files-can-mean)

---

[Plugin overview](../README.md)

## When files and Stats change

```text
case.start / case.skip
    │
    ├── attempt-1.json: running, revision 1
    │
    ├── test body, child tests, hooks, fixture/test cleanup
    │
    ├── attempt-1.json: finished, revision 2
    │   Stats now contains the attempt
    │
    └── parent test cleanup
        attempt-1.json: finalized, revision 3
        Stats and JSON receive any late failure correction
```

After each attempt write, the plugin rebuilds `run.json` from the saved attempts. Writes are synchronous.

| State | Available data |
|-------|----------------|
| `running` | Start and metadata; no outcome or finish time |
| `finished` | Outcome and duration after the plugin's attempt cleanup |
| `finalized` | Outcome rechecked during parent cleanup; no further revisions |

The attempt cleanup runs after cleanups registered later. A cleanup registered earlier by another plugin runs
after it. Go can also mark cleanup panics and data races failed after attempt cleanups have run.

Example: a late cleanup fails an attempt that was initially recorded as passed:

| Snapshot of the same file | `state` | `status` | `revision` |
|---------------------------|---------|----------|------------|
| After attempt cleanup | finished | passed | 2 |
| After parent cleanup | finalized | failed | 3 |

For a bare Config whose active test is its RootT, there is no later parent check: the file remains `finished`.
Runner resource cleanup is outside every attempt and does not affect its status or duration.

## Partial results

Assuming earlier writes succeeded:

| Interruption | Files left on disk |
|--------------|--------------------|
| Timeout or process kill during attempt 2 | Attempt 1's saved snapshot; attempt 2 still `running` |
| Exit before parent cleanup | Completed attempts can remain `finished` |
| Exit after writing an attempt, before updating the run | New attempt file; missing or older `run.json` |
| Crash during a temporary-file write | Previous destination, if any; possibly `.teststats-*.tmp` |

A stale run snapshot can look like this:

```text
attempt-1.json: revision=3, status=failed
run.json:      attempts[0].revision=2, summary.status=passed
```

For an external reader:

1. Read after test processes exit; ignore `.teststats-*.tmp` files.
2. Check schema versions and validate attempt files. Keep malformed or missing data visible as incomplete.
3. Rebuild summaries from attempt files. To reuse `run.json`, first match the complete attempt list and revisions.
4. Preserve `running` and `finished` as unconfirmed states. Do not turn them into a complete successful invocation.
5. Use the original `go test` exit code for the process result, including failures outside Axiom cases.

If an existing attempt file is malformed or incompatible, the writer reports an error and leaves `run.json`
unchanged. A newly saved attempt is retained even if the run update fails.

## File replacement

```text
serialize JSON → create temporary file in target directory → write → sync → close → rename
```

| Guarantee | Scope |
|-----------|-------|
| Old or new whole file | Local Unix filesystems with atomic rename; other platforms follow `os.Rename` guarantees |
| Failed replacement preserves the previous destination | Until rename succeeds; temporary files are removed on ordinary errors |
| Completed writes survive later test-process termination | No guarantee against machine power loss; the directory is not synced |
| Attempt and run updated together | Not guaranteed: these are two separate file replacements |

## What no files can mean

Cases excluded by `go test -run` and planning Configs produce no records. Policy-skipped cases do produce records.
Start timing excludes the wait for parallel scheduling.

An empty directory cannot distinguish no selected cases from a compilation error, initialization panic, write
failure, or crash before the first Axiom case. The plugin does not write `launch.json`, package manifests, or
`process.json`; it has no complete-launch signal.
