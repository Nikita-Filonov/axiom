# Changelog

User-facing changes to Axiom since 2026-03-26, reconstructed from the repository's commits and release tags. Core
versions and plugin versions are independent. The first core release in this period was `v0.15.0` on 2026-04-16.

## Core versions

### v1.20.0 — 2026-10-31

- Clarified that a failed retry attempt keeps the Go test run failed even if a later attempt passes, and documented
  command-level reruns with `gotestsum`.
- Clarified that `Config.Setup` and `Config.Teardown` execute at the call site; use hooks or fixture cleanup to schedule
  work around a test attempt.
- Expanded the plugin authoring and installation guides, including plugin lifecycle and state handling.
- This release changes documentation only; the core API and runtime behavior are unchanged from v1.19.0.

### v1.19.0 — 2026-10-01

- **Breaking:** `Runner.BuildConfig`, `Runner.ApplyStart`, and `Runner.ApplyFinish` are no longer exported. `RunCase`,
  `Suite`, and `RunPackage` own the runner lifecycle and build every `Config`; a plugin receives its `Config` from
  them.
- **Breaking:** other execution internals are no longer exported: `Config.Test`, `Config.ApplyPlugins`, the
  `Hooks.ApplyBefore*` and `Hooks.ApplyAfter*` methods, `Fixtures.Teardown`, `Resources.Teardown`, the `Runtime`
  dispatch methods (`Test`, `Step`, `Setup`, `Teardown`, `Log`, `Event`, `Assert`, `Artefact`), `Event.Normalize`,
  `NewLogEvent`, `NewAssertEvent`, `NewArtefactEvent`, `SuiteRunner.BuildSuite`, `NewSuiteConfig`, and
  `NewSuiteTestConfig`. Use the `Config` methods to emit logs, steps, events, assertions, and artefacts. Calling
  `cfg.Test` inside a test body re-ran `BeforeTest` and `AfterTest` hooks and fixture cleanup; write that code in the
  body directly.
- **Breaking:** `TestingSuite` can be satisfied only by embedding `Suite`. `Suite.SetRootT`, `Suite.SetSubT`, and
  `Suite.SetRunner` are no longer exported; the `RootT`, `SubT`, and `Runner` fields and `Suite.T` are unchanged.
- **Breaking:** removed the unused `Copy`, `Join`, and `Normalize` interfaces. The `Copy`, `Join`, and `Normalize`
  methods on configuration types are unchanged.
- **Breaking:** the `Cache` and `Cleanups` fields of `Fixtures` and `Resources` are no longer exported, and the
  `FixtureResult`, `ResourceResult`, `FixtureCleanup`, and `ResourceCleanup` types are removed. Values are built and
  cached only by `GetFixture` and `GetResource`, so writing the resource cache can no longer bypass its lock.
  `Registry` and the `New*`/`With*` constructors are unchanged. `GetFixture` now works on a zero `Fixtures` value.
- Added `Config.Execution` (`ID` and `Attempt`) for correlating the retries of one
  `RunCase` invocation, and a `case.skip` event for policy skips.
- CI checks plugins against the checked-out core through a Go workspace.

### v1.18.0 — 2026-10-01

- Added `Meta.Owner` and `WithMetaOwner` for a test's responsible person or team. Case metadata can override a Runner
  owner.

### v1.17.0 — 2026-09-26

- `ParamFixture.Default(params)` returns a `RunnerFixtureRegistrar` for registration with
  `axiom.WithRunnerFixtures(fixture.Default(params))`. Cases can select their own parameters with
  `axiom.WithCaseFixtures(fixture.For(params))`.

### v1.16.0 — 2026-09-25

- `Runner.Join` records its parent and overlay so tools can explain where merged runner settings came from.

### v1.15.0 — 2026-09-24

- Expanded documentation for the public API and plugins. This release primarily documents existing behavior.

### v1.14.0 — 2026-09-21

- Added `Cache` and typed `CacheKey[T]` with `Get`, `Set`, `Delete`, and coordinated `GetOrCreate`. The caller controls
  cache lifetime and cleanup.

### v1.13.0 — 2026-09-17

- Added case-level lifecycle hooks through `WithCaseHooks`.

### v1.12.0 — 2026-09-15

- Added `ParamFixture[P, T]`: a case selects a fixture variant with `For(params)`, and a runner can supply a default
  with `Default(params)`.
- Added `WithCaseFixtures` for registering typed fixture definitions on a case.

### v1.11.0 — 2026-09-11

- Added typed `ContextKey[T]` for storing and reading runner and case context values.

### v1.10.0 — 2026-09-11

- Added typed fixture and resource keys and definitions: `FixtureKey[T]`, `ResourceKey[T]`, `DefineFixture`, and
  `DefineResource`.

### v1.9.0 — 2026-08-22

- Expanded regression tests for fixtures, resources, parameters, metadata, and runtime execution.

### v1.8.0 — 2026-08-21

- Kept after-test hooks and fixture cleanup inside the test runtime wrapper, so reporting plugins can observe them
  before the attempt closes.
- Added lifecycle coverage for fixture cleanup, including failures and panics.

### v1.7.0 — 2026-07-24

- Moved case execution into a dedicated attempt flow. Each retry gets a fresh `Config`; skip and parallel policies are
  applied at the appropriate subtest boundary.

### v1.6.0 — 2026-06-29

- Added `RunPackage` and `RunPackageWith` to manage a runner's hooks and resources across multiple top-level Go tests.

### v1.5.0 — 2026-06-27

- Moved fixture and resource cleanup into their own lifecycle lists with reverse-order teardown.
- Added an explicit skip override so a case can disable a runner's skip setting.

### v1.4.0 — 2026-06-21

- Hardened fixture and resource lookup, runtime events, logs, assertions, and artefact handling.
- Added the independently versioned `testexplain` and `testtracing` plugins.

### v1.3.0 — 2026-06-17

- Added parallel configuration for suites and individual suite tests.

### v1.2.0 — 2026-06-17

- Added per-test suite configuration, including runner selection.

### v1.1.0 — 2026-06-17

- Extended typed toolset construction and its documentation.

### v1.0.0 — 2026-06-17

- Added typed `Local` state and `Toolset` support for per-attempt helpers.

### v0.18.0 — 2026-06-16

- Added `Config.T()` and suite access to the active `*testing.T`.

### v0.17.0 — 2026-06-15

- Reworked suites around explicitly registered tests and suite configuration.

### v0.16.0 — 2026-06-13

- Introduced the suite execution API and runner lifecycle cleanup for suite tests.

### v0.15.0 — 2026-04-16

- Added explicit parallel settings so a case can override a runner's parallel default, including disabling it.

## Independently versioned plugins

### teststats v0.37.0 — 2026-11-12

- Added `Stats.Elapsed()` to measure the span from the earliest completed attempt start to the latest end.
  Overlapping tests are counted once; gaps between retries and runs are included. Empty statistics return zero.
  The method also works with loaded and filtered results. Partial results use only retained completed attempts;
  this does not measure compilation, process startup, or the full CI job. The v1 JSON format is unchanged.

### teststats v0.36.0 — 2026-11-12

- **Breaking:** `Plugin` now accepts options only. Replace `Plugin(stats)` with `Plugin(WithStats(stats))`.
  Omitting `WithStats` creates one collector per plugin instance; in-memory queries and retry classification are unchanged.
- Added `WithOutputDir` for incremental, versioned JSON attempt and run snapshots, isolated by process and run.
  Attempts are published at start, after cleanup, and after parent cleanup reconciles late failures. Publication
  uses a synced temporary file and rename; completed snapshots survive later test-process termination.
- Added `WithErrorHandler` for export failures, which default to stderr logging and do not fail tests. Documented
  the v1 JSON Schema, partial results, derived run snapshots, and filesystem guarantees. Launch/package/process
  manifests and notification adapters are outside this export; no core changes are required.
- Added `Load(dir)` to restore statistics from attempt files in another process. `LoadResult` returns usable Stats,
  unfinalized attempts, per-file diagnostics, and an `Incomplete()` check. Stale run snapshots are ignored; malformed
  attempts and numbering gaps remain visible without discarding other results.
- Added `ProcessID` to Attempt and Run so equal run IDs from different processes stay separate. The original run IDs
  and v1 JSON format are preserved. Exported `AttemptState` and its lifecycle constants for loaded snapshots.
- Reorganized the plugin documentation into a short README and task-specific guides with Go examples, JSON files,
  lifecycle tables, and CI commands that preserve the test exit code.
- Separated JSON validation, attempt validation, and file loading. Added regression tests for malformed artifacts,
  default error logging, path resolution, and file failures that must preserve existing snapshots.

### testtimeout v0.14.0 — 2026-10-05

- Added `WithContextDeadline()` to give `Raw`, `DB`, `MQ`, and `RPC` contexts the same deadline as the case timeout.
  The contexts are canceled when the attempt finishes or times out; earlier parent deadlines still apply.
- An attempt that returns after its deadline is reported as timed out, including when it returns on context cancellation.

### testtracing v0.18.0 — 2026-10-02

- Repeated installation with the same `Trace` and `Config`, including on both
  Runner and Case, now registers one collector instead of creating duplicate
  records. Different traces and retry attempts remain independent, and repeated
  events are preserved. Use separate Trace collectors for independent copies of
  the event stream. Nil Trace and Config arguments now panic during installation.

### testjunit v0.1.0 — 2026-10-02

- Added the independent `testjunit` module. It exports executed and
  policy-skipped attempts as JUnit XML, including retry outcomes, timing, and
  available lifecycle failure messages. Its own recorder reconciles late test
  failures during parent cleanup and depends only on Axiom core. Reports can be
  written to an `io.Writer` or to a file after tests finish. Register it with
  `testjunit.Plugin(reporter, options...)`; suite options belong to each plugin
  installation, and a shared reporter groups results by suite name.

### testleaks v0.1.0 — 2026-10-02

- Added the independent `testleaks` module. It checks attempt-labeled goroutines
  after the test body and its later-registered cleanups, and reports explicitly
  tracked resources that were not released. It supports a grace period, exact
  function-name exclusions, and an `io.ReadCloser` tracking wrapper.

### testotel v0.1.0 — 2026-10-02

- Added the independent `testotel` module. It emits one OpenTelemetry span per
  case attempt or policy skip, correlates retries, records selected lifecycle
  events, and provides `Context(cfg)` for child spans. The caller owns the SDK
  and exporter.

### testenv v0.2.0 — 2026-10-02

- Moved resource configuration and its builder into `config.go`, and the
  environment snapshot into `envs.go`. The public API and behavior are unchanged.

### testflags v0.4.0 — 2026-10-02

- Moved flag and resource configuration into `config.go`. The public API and
  behavior are unchanged.

### testenv v0.1.0 — 2026-10-02

- Added the independent `testenv` module for typed environment variables shared
  through a lazy runner resource. Values are captured on first access; typed
  keys support required variables and defaults.

### testflags v0.3.0 — 2026-10-02

- Clarified the plugin documentation and moved resource construction into a
  dedicated method. The public API and behavior are unchanged.

### teststats v0.31.0 — 2026-10-01

- Replaced the old `CaseResult` and public counter fields with `Attempt`, `Run`,
  and `Summary`. The plugin now records policy skips, lifecycle failures, and
  timing through cleanup; groups retries by run ID and derives flaky status.
  `Stats.Record` and `Stats.Cases` are removed; use `Stats.Attempts()`,
  `Stats.Runs()`, and `Stats.Summary()` to read results. The collector is safe
  for concurrent use.

### testflags v0.1.0 — 2026-10-01

- Added typed CLI flag declarations with functional options and `FlagKey[T]` access.
- `axiom.WithRunnerResources(testflags.Resource())` registers a lazy shared snapshot for resource constructors,
  hooks, fixtures, and cases. Snapshots expose defaults and explicitly set values.

### Plugin release index

The table shows the first plugin version since 2026-03-26 and the latest version
documented here as of 2026-11-12. Plugin versions are independent of the core version.

| Plugin           | First version in period | Latest version     |
|------------------|-------------------------|--------------------|
| `testallure`     | `v0.13.0`               | `v0.41.0`          |
| `testassert`     | `v0.10.0`               | `v0.32.0`          |
| `testenv`        | `v0.1.0`                | `v0.5.0`           |
| `testexplain`    | `v0.1.0`                | `v0.20.0`          |
| `testflags`      | `v0.1.0`                | `v0.7.0`           |
| `testjunit`      | `v0.1.0`                | `v0.3.0`           |
| `testleaks`      | `v0.1.0`                | `v0.3.0`           |
| `testlogger`     | `v0.12.0`               | `v0.34.0`          |
| `testotel`       | `v0.1.0`                | `v0.4.0`           |
| `testquarantine` | `v0.1.0`                | `v0.14.0`          |
| `teststats`      | `v0.13.0`               | `v0.37.0`          |
| `testtags`       | `v0.13.0`               | `v0.35.0`          |
| `testtimeout`    | `v0.1.0`                | `v0.15.0`          |
| `testtracing`    | `v0.1.0`                | `v0.20.0`          |

Notable plugin changes in this period:

- `testexplain` and `testtracing` first shipped on 2026-06-21. `testexplain` gained more detailed explanations of merged
  runners and case execution in `v0.12.0` (2026-09-25).
- `testallure` switched to the official `allure-framework/allure-go` integration in `v0.20.0` (2026-07-28). Later
  releases added lifecycle coverage and assertion failure reporting through `testallure.T(cfg)`.
- `testquarantine` and `testtimeout` first shipped in `v0.1.0` on 2026-09-11.

For plugin-specific configuration and limitations, see each plugin's `README.md` under `plugins/`.
