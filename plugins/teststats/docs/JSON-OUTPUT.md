# 📘 JSON output

## 📑 Table of Contents

- [Enable files](#enable-files)
- [Run several packages](#run-several-packages)
- [Handle write errors](#handle-write-errors)
- [Preserve the test exit code in CI](#preserve-the-test-exit-code-in-ci)

---

[Plugin overview](../README.md)

## Enable files

```go
package example_test

import (
	"os"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/teststats"
)

func TestLogin(t *testing.T) {
	resultsDir := os.Getenv("AXIOM_RESULTS_DIR")
	if resultsDir == "" {
		t.Fatal("set AXIOM_RESULTS_DIR before running tests")
	}

	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			teststats.Plugin(teststats.WithOutputDir(resultsDir)),
		),
	)
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("login")), func(cfg *axiom.Config) {
		// Test logic and assertions.
	})
}
```

`AXIOM_RESULTS_DIR` is an environment variable chosen for this example. The plugin does not read it automatically.
No `TestMain` or final flush is needed.

## Run several packages

Run from your project root:

```bash
export AXIOM_RESULTS_DIR
AXIOM_RESULTS_DIR="$(mktemp -d "$PWD/axiom-results.XXXXXX")"
go test -count=1 ./...
```

| Setting | Reason |
|---------|--------|
| Same absolute directory for every package | Go test binaries normally run from different package directories |
| Fresh directory per invocation | The plugin does not remove old files |
| `-count=1` | Cached tests do not execute the plugin or produce fresh files |

After a case with two attempts:

```text
axiom-results.A1B2C3/
└── processes/
    └── <process-id>/
        └── runs/
            └── <run-key>/
                ├── attempt-1.json
                ├── attempt-2.json
                └── run.json
```

Each test binary has a separate process ID. See [the file contents](JSON-FORMAT.md).

## Handle write errors

The default handler logs to stderr. To use your logger:

```go
teststats.Plugin(
	teststats.WithOutputDir(resultsDir),
	teststats.WithErrorHandler(func(err error) {
		logger.Printf("statistics export: %v", err)
	}),
)
```

Handlers run synchronously and can be called concurrently by different plugin instances. They must return normally
and should not call `t.Error`, `t.Fatal`, or `t.Skip`. Output failures do not otherwise fail the test.

## Preserve the test exit code in CI

With `AXIOM_RESULTS_DIR` set as above:

```bash
test_exit_code=0
go test -count=1 ./... || test_exit_code=$?

# Upload "$AXIOM_RESULTS_DIR" here using your CI's artifact mechanism.
# An optional publisher must not stop this script before the final exit.

exit "$test_exit_code"
```

There is no built-in publisher or public JSON loading API yet. If you consume the files directly, follow the
[partial-results rules](LIFECYCLE.md#partial-results).
