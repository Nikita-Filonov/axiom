# Axiom Plugins and Resources

The modules in this directory extend Axiom without adding dependencies to the core. They are separate Go modules with
their own versions and README files. Most register an `axiom.Plugin` through `axiom.WithRunnerPlugins(...)` or
`axiom.WithCasePlugins(...)`. `testenv` and `testflags` provide runner resources instead; register them with
`axiom.WithRunnerResources(...)`.

## 📑 Table of Contents

- [Modules](#modules)
- [Installation](#installation)
- [Writing a plugin](#writing-a-plugin)

## Modules

- **🟣 Allure Plugin:** [testallure](./testallure). Generates Allure reports by projecting Axiom runtime
  events (tests, steps, artefacts, metadata) into the Allure execution model.
- **📝 Logger Plugin:** [testlogger](./testlogger). Consumes structured log events emitted via `cfg.Log(...)`
  and forwards them to Go’s `log/slog` logging infrastructure.
- **📊 Stats Plugin:** [teststats](./teststats). Records every attempt, including skips and cleanup
  failures, groups retries into runs with flaky detection, and counts runs and attempts.
- **🔎 Tracing Plugin:** [testtracing](./testtracing). Records raw config-scoped runtime events into an
  in-memory trace for later inspection or export.
- **🔭 OpenTelemetry Plugin:** [testotel](./testotel). Exports per-attempt spans and selected lifecycle
  events through a caller-provided OpenTelemetry tracer provider.
- **🧭 Explain Plugin:** [testexplain](./testexplain). Captures a structured explanation of the merged
  runner/case configuration before test execution.
- **🏷 Tags Plugin:** [testtags](./testtags). Filters test execution based on metadata tags using include /
  exclude rules. Can be configured via code or environment variables.
- **✅ Assert Plugin:** [testassert](./testassert). Bridges Axiom’s structured runtime assertions with
  `stretchr/testify/assert`. Allows test code to emit declarative assertion events without coupling to a specific
  assertion backend.
- **⏱ Timeout Plugin:** [testtimeout](./testtimeout). Enforces a per-case wall-clock deadline, failing a
  hanging case with a readable message and an attached goroutine dump instead of stalling the whole test binary.
- **🧟 Quarantine Plugin:** [testquarantine](./testquarantine). Quarantines known-flaky cases by skipping
  them before execution with a recorded reason, so they stay visible without gating the suite. Can be configured to run
  them anyway in non-gating jobs.
- **🚩 Flags Resource:** [testflags](./testflags). Shares typed CLI flags with runner resources, hooks,
  fixtures, and tests.
- **🌱 Environment Resource:** [testenv](./testenv). Shares a typed snapshot of environment variables with
  runner resources, hooks, fixtures, plugins, and tests.
- **🧵 Leak Checks Plugin:** [testleaks](./testleaks). Checks attempt-labeled goroutines after the body
  and its later-registered cleanups, plus explicitly tracked resources that were not released.
- **📄 JUnit XML Plugin:** [testjunit](./testjunit). Exports finished case attempts as JUnit XML for CI
  test reports.

## Installation

Install a module with standard Go tooling, for example:

```shell
go get github.com/Nikita-Filonov/axiom/plugins/testtags
```

Each module is versioned independently of the Axiom core. Pin a version with `@vX.Y.Z` when needed. Each module's
README shows how to register and configure it.

## Writing a plugin

See the [authoring guide](../docs/plugins) for the plugin contract, a runnable example, lifecycle rules, and tests.
