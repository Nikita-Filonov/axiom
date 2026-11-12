# 📘 JSON format — v1

## 📑 Table of Contents

- [attempt-1.json](#attempt-1json)
- [run.json](#runjson)

---

[Plugin overview](../README.md) · [JSON Schema](../schema/v1.json)

The examples use readable IDs. Actual process and Axiom run IDs are generated at runtime.

```text
<output-directory>/processes/process-1/runs/72756e2d31/
├── attempt-1.json
└── run.json
```

`72756e2d31` is the lowercase hex encoding of `run-1`.

## attempt-1.json

One file per attempt. The same path is replaced as its [state changes](LIFECYCLE.md#when-files-and-stats-change).

```json
{
  "schema_version": 1,
  "kind": "attempt",
  "process_id": "process-1",
  "run_id": "run-1",
  "number": 1,
  "revision": 3,
  "state": "finalized",
  "case_id": "C-1",
  "name": "login",
  "test_name": "TestLogin/login",
  "meta": {
    "owner": "qa",
    "tags": ["smoke"],
    "labels": {"component": "login"}
  },
  "status": "passed",
  "started_at": "2026-11-11T10:00:00Z",
  "finished_at": "2026-11-11T10:00:01Z",
  "duration_ns": 1000000000
}
```

| Field | Meaning |
|-------|---------|
| `schema_version` | Format version; check before reading fields |
| `process_id` | Random namespace shared by plugin instances in one process; not an OS PID |
| `run_id` | `Config.Execution.ID`; shared by retries of one `RunCase` call |
| `number` | One-based attempt number; a policy skip is attempt 1 |
| `revision` | Increases on each snapshot; failed writes can leave gaps |
| `state` | `running`, `finished`, or `finalized`; independent of pass/fail outcome |
| `status` | `passed`, `failed`, or `skipped`; absent while running; failure takes precedence over skip |
| `meta` | Metadata captured at start, using the snake_case fields in the schema |
| `started_at`, `finished_at` | UTC RFC 3339 timestamps; finish is absent while running |
| `duration_ns` | Integer nanoseconds, measured with the monotonic clock; zero while running |

Optional fields: `case_id`, `skip_reason`, and `error`. `skip_reason` records a policy skip; Go does not expose the
text from `t.Skip`. `error` records the first Axiom lifecycle failure; it does not contain `t.Error` or `t.Fatal` logs.

For a manually constructed Config with no execution ID, the exporter generates a run ID. The in-memory attempt
keeps its empty RunID. Test names and Case IDs are never used as directory names.

## run.json

Lists observed attempts and summarizes the completed ones:

```json
{
  "schema_version": 1,
  "kind": "run",
  "process_id": "process-1",
  "run_id": "run-1",
  "attempts": [
    {
      "number": 1,
      "revision": 3,
      "state": "finalized"
    }
  ],
  "summary": {
    "case_id": "C-1",
    "name": "login",
    "meta": {
      "owner": "qa",
      "tags": ["smoke"],
      "labels": {"component": "login"}
    },
    "status": "passed",
    "started_at": "2026-11-11T10:00:00Z",
    "finished_at": "2026-11-11T10:00:01Z",
    "duration_ns": 1000000000,
    "elapsed_ns": 1000000000
  }
}
```

| Field | Rule |
|-------|------|
| `attempts` | Observed attempt numbers, revisions, and states, sorted by number |
| `summary` | `null` until at least one attempt finishes; includes `finished` and `finalized` attempts |
| `summary.status` | Same [retry classification](STATISTICS.md#a-failed-attempt-followed-by-a-pass) as `Stats.Runs()` |
| `summary.case_id`, `name`, `meta` | Taken from the last completed attempt |
| `summary.duration_ns` | Sum of completed attempt durations |
| `summary.elapsed_ns` | Earliest completed-attempt start to latest finish, including retry delays |

A running retry appears in `attempts` but adds no outcome or duration to `summary` yet. Decoded timestamp
subtraction need not exactly match the monotonic `duration_ns` measurement.

`run.json` can be stale after a crash. [Load](LOADING.md) rebuilds statistics from attempt files; see
[partial results](LIFECYCLE.md#partial-results). Neither a run summary nor a finalized attempt proves that the
whole test invocation completed.

These examples match the versioned [attempt fixture](../testdata/attempt-v1.json) and
[run fixture](../testdata/run-v1.json). Changes to Go structs do not automatically change the JSON schema.
