# 📘 Stats Plugin (`teststats`)

## 📑 Table of Contents

- [Install](#install)
- [Connect the plugin](#connect-the-plugin)
- [Guides](#guides)

---

Collects Axiom test results in memory, saves them as JSON, and loads saved results in another process.
One `RunCase` call produces a run; each retry produces an attempt.

## Install

```shell
go get github.com/Nikita-Filonov/axiom github.com/Nikita-Filonov/axiom/plugins/teststats
```

The examples below use the `v0.36.0` API. To work from this checkout, see
[local development](docs/CONFIGURATION.md#local-development).

## Connect the plugin

```go
stats := teststats.NewStats()

runner := axiom.NewRunner(
	axiom.WithRunnerPlugins(
		teststats.Plugin(
			teststats.WithStats(stats),
			teststats.WithOutputDir("./axiom-results"),
		),
	),
)
```

`WithStats` gives you a collector to query. `WithOutputDir` enables JSON files. Both options are optional.

**Migration:** replace `teststats.Plugin(stats)` with `teststats.Plugin(teststats.WithStats(stats))`.

## Guides

| Task                                                       | Guide                                      |
|------------------------------------------------------------|--------------------------------------------|
| Choose options and defaults                                | [Configuration](docs/CONFIGURATION.md)     |
| Read attempts, runs, counts, and filtered results          | [Statistics in memory](docs/STATISTICS.md) |
| Save JSON from one or several test packages                | [JSON output](docs/JSON-OUTPUT.md)         |
| Load files in a notifier or another process                | [Loading results](docs/LOADING.md)         |
| Inspect `attempt-1.json` and `run.json`                    | [JSON format](docs/JSON-FORMAT.md)         |
| Understand cleanup failures, timeouts, and partial results | [Lifecycle](docs/LIFECYCLE.md)             |

The plugin exports and loads observed cases. Launch/package/process manifests and notification adapters
are not implemented yet.
