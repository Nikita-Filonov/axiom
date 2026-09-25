# Changelog

User-facing changes to Axiom since 2026-03-26, reconstructed from the repository's commits and release tags. Core
versions and plugin versions are independent. The first core release in this period was `v0.15.0` on 2026-04-16.

## Core releases

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

The table shows the first plugin tag since 2026-03-26 and the latest published tag as of 2026-09-26. It is a release
index, not a claim that every intermediate tag changed plugin behavior; many tags update the required Axiom version.

| Plugin           | First tag in period | Latest published tag |
|------------------|---------------------|----------------------|
| `testallure`     | `v0.13.0`           | `v0.33.0`            |
| `testassert`     | `v0.10.0`           | `v0.25.0`            |
| `testexplain`    | `v0.1.0`            | `v0.13.0`            |
| `testlogger`     | `v0.12.0`           | `v0.27.0`            |
| `testquarantine` | `v0.1.0`            | `v0.7.0`             |
| `teststats`      | `v0.13.0`           | `v0.28.0`            |
| `testtags`       | `v0.13.0`           | `v0.28.0`            |
| `testtimeout`    | `v0.1.0`            | `v0.7.0`             |
| `testtracing`    | `v0.1.0`            | `v0.13.0`            |

Notable plugin changes in this period:

- `testexplain` and `testtracing` first shipped on 2026-06-21. `testexplain` gained more detailed explanations of merged
  runners and case execution in `v0.12.0` (2026-09-25).
- `testallure` switched to the official `allure-framework/allure-go` integration in `v0.20.0` (2026-07-28). Later
  releases added lifecycle coverage and assertion failure reporting through `testallure.T(cfg)`.
- `testquarantine` and `testtimeout` first shipped in `v0.1.0` on 2026-09-11.

For plugin-specific configuration and limitations, see each plugin's `README.md` under `plugins/`.
