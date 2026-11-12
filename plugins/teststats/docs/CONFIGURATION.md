# 📘 Configuration

## 📑 Table of Contents

- [Options](#options)
- [In-memory results](#in-memory-results)
- [JSON output](#json-output)
- [Repeated installation](#repeated-installation)
- [Local development](#local-development)

---

[Plugin overview](../README.md)

## Options

| Option                      | Default                               | Invalid argument                     |
|-----------------------------|---------------------------------------|--------------------------------------|
| `WithStats(stats)`          | One new collector per `Plugin()` call | `nil` panics                         |
| `WithOutputDir(dir)`        | File output disabled                  | Empty or whitespace-only path panics |
| `WithErrorHandler(handler)` | Log output errors to stderr           | `nil` panics                         |

Passing a nil option to `Plugin` also panics. Creating or installing the plugin does not write files.

## In-memory results

```go
stats := teststats.NewStats()
plugin := teststats.Plugin(teststats.WithStats(stats))
```

Pass `plugin` to `axiom.WithRunnerPlugins`. Read results through `stats.Attempts()`, `stats.Runs()`, or
`stats.Summary()`. See the [complete test](STATISTICS.md#complete-example).

## JSON output

```go
plugin := teststats.Plugin(teststats.WithOutputDir("./axiom-results"))
```

The plugin creates its own collector. You do not need a `Stats` variable to write files.
Combine `WithStats` and `WithOutputDir` when you also need in-memory queries.

Relative paths are resolved when `Plugin` is created. For multiple test packages, use the
[shared absolute directory example](JSON-OUTPUT.md#run-several-packages).

## Repeated installation

| On the same Config                       | Result                                                    |
|------------------------------------------|-----------------------------------------------------------|
| Same `Stats` on Runner and Case          | One stored attempt in that collector                      |
| Different collectors                     | Each receives its own copy                                |
| Same output directory on Runner and Case | One set of files; first installation's error handler wins |
| Different output directories             | Each receives the snapshots                               |

Directories are compared as absolute, cleaned paths. Symlink aliases are not resolved.

## Local development

From the Axiom repository root, create a workspace if none exists:

```shell
go work init . ./plugins/teststats
```

For an existing workspace:

```shell
go work use . ./plugins/teststats
```

This uses the local core and plugin together. Before releasing the plugin, release the core and update the
plugin's core requirement.
