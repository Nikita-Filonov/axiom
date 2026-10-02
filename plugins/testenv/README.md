# 🌱 Environment Variables Plugin (`testenv`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [What the plugin does](#what-the-plugin-does)
- [Configuration](#configuration)
- [Snapshot and lifetime](#snapshot-and-lifetime)
- [Installation](#installation)
- [Example](#example)

---

## Overview

Typed environment variables shared by a runner's resources, hooks, fixtures,
plugins, and tests. Register the lazy snapshot in the **resources** group with
`axiom.WithRunnerResources(testenv.Resource())`.

Unlike the case-filtering plugins, `testenv` does not skip or execute tests. It
provides configuration values to the code that owns those decisions.

---

## What the plugin does

On first access, `Resource()` captures the process environment once per runner.
Typed keys then read and parse values from that snapshot. Each key has an exact,
case-sensitive name; an explicitly set empty string differs from an unset
variable.

Values may contain secrets. `testenv` never logs them, and parse errors omit the
raw value. Take care when logging a value returned by a key or `Lookup`.

---

## Configuration

| Option               | Meaning                                                     |
|----------------------|-------------------------------------------------------------|
| `WithName(name)`     | Required variable name.                                     |
| `WithDefault(value)` | Value for an unset variable; its type must match the key.   |
| `WithSource(source)` | Source for `Resource`; defaults to `os.Environ`.            |

Constructors: `String`, `Bool`, `Int`, `Int64`, `Uint`, `Uint64`, `Float64`, and
`Duration`. Each returns an `EnvKey[T]` whose `Get(runner)` returns `T`.
`Duration` accepts Go duration strings such as `250ms` and `2m`.

An unset variable is **required** unless `WithDefault` is supplied. `TryGet`
returns an error for missing or malformed values; `Get` panics with that error.
Defaults apply only when the variable is unset. For example, an explicitly set
empty string is valid for `String`, but invalid for `Int` and `Duration`.

For a test or custom environment, use `Resource(testenv.WithSource(source))`.
The source returns `[]string` entries in `KEY=value` format and is called once
when the snapshot is created. See the [runnable example](example_test.go).

---

## Snapshot and lifetime

Runner construction does not read the environment. Set variables before the
first key or `testenv.Get(runner)` call. Later environment changes do not affect
that runner's snapshot. Resource constructors and `BeforeAll` hooks can call
`key.Get(runner)`; plugins, fixtures, and cases use `key.Get(cfg.Runner)`.

`testenv.Get(runner).Lookup(name)` returns a raw value and a presence flag.
Runner copies and joins follow Axiom's resource rules: copies made before the
first read build their own snapshot; copies made afterward share the cached one.

---

## Installation

The plugin is an independently versioned Go module. After its first release,
install it with standard Go tooling:

```shell
go get github.com/Nikita-Filonov/axiom/plugins/testenv
```

---

## Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testenv"
)

var target = testenv.String(
	testenv.WithName("TEST_ENV"),
	testenv.WithDefault("local"),
)

var runner = axiom.NewRunner(
	axiom.WithRunnerResources(testenv.Resource()),
)

func TestEnvironment(t *testing.T) {
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("configured environment")), func(cfg *axiom.Config) {
		// String returns the exact environment value, or "local" when unset.
		cfg.T().Logf("target environment: %s", target.Get(cfg.Runner))
	})
}
```

Run with `TEST_ENV=stable go test -v`. The key then returns `"stable"`; without
the variable it returns `"local"`.
