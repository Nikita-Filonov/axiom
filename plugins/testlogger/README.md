# 📝 Logger Plugin (`testlogger`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Example](#example)

---

## Overview

Provides structured logging for tests using Go’s standard `log/slog` package. The plugin consumes logs emitted via
`cfg.Log(...)` and forwards them to a `slog.Logger` with appropriate log levels.

This allows plugins and test code to emit structured logs without coupling to a specific logging backend.

---

## Installation

The plugin is distributed as a regular Go module and installed using standard Go tooling.

Add the plugin dependency using `go get`:

```shell
go get github.com/Nikita-Filonov/axiom/plugins/testlogger
```

This will add the plugin to your `go.mod` file:

```text
require (
	github.com/Nikita-Filonov/axiom v0.3.0
	github.com/Nikita-Filonov/axiom/plugins/testlogger v0.1.0
)
```

Each plugin is versioned independently from the Axiom core.

---

## Features

- maps `axiom.LogLevel` to `slog.Level`
- logs are emitted through the runtime log pipeline
- respects test context (`cfg.Context.Raw`)
- zero configuration: uses standard text logger to stdout

---

## Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testlogger"
)

func TestLoggerExample(t *testing.T) {

	// Enable structured logging via slog
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testlogger.Plugin(),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("logging example"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		// Emit log events into the runtime.
		// The logger plugin consumes them and forwards to slog.
		cfg.Log(axiom.NewInfoLog("starting test"))
		cfg.Log(axiom.NewWarningLog("something looks odd"))

		cfg.Step("do work", func() {

			// Logs emitted inside steps are automatically
			// associated with the current test context.
			cfg.Log(axiom.NewDebugLog("inside step"))
		})
	})
}
```
