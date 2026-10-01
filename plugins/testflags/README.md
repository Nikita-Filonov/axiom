# CLI flags (`testflags`)

---

## 📑 Table of Contents

- [Overview](#overview)
- [Configuration](#configuration)
- [Shared setup and TestMain](#shared-setup-and-testmain)
- [Inspecting flags](#inspecting-flags)
- [Lifetime and errors](#lifetime-and-errors)
- [Installation](#installation)
- [Example](#example)

---

## Overview

Typed command-line flags shared by a runner's resources, hooks, fixtures, and tests.
Register their snapshot in the **resources** group: `axiom.WithRunnerResources(testflags.Resource())`.

Go parses arguments before ordinary tests. Declare flags at package level, before
parsing; declaring the runner there is safe because its flag snapshot loads on first read.

---

## Configuration

| Option               | Meaning                                             |
|----------------------|-----------------------------------------------------|
| `WithName(name)`     | Required name, without leading dashes.              |
| `WithDefault(value)` | Default value; omitted means the type's zero value. |
| `WithUsage(text)`    | Description shown in Go's flag help.                |
| `WithFlagSet(fs)`    | Set to register on; defaults to `flag.CommandLine`. |

Constructors: `Bool`, `Int`, `Int64`, `Uint`, `Uint64`, `Float64`, `String`, and
`Duration`. Each returns a `FlagKey[T]` whose `Get(runner)` returns `T`.
Defaults must have the constructor's exact type, for example `int64(10)` for
`Int64`, or `5*time.Second` for `Duration`; mismatches panic at declaration time.

For a flag registered with the standard `flag` package, use
`testflags.NewFlagKey[T](name)`. This creates a key without registering another flag.

For an isolated `*flag.FlagSet`, use `WithFlagSet(fs)` on declarations and
`testflags.Resource(testflags.WithSource(fs))` on the runner. Check `fs.Parse(args)`
for errors before reading. See the [runnable examples](example_test.go).

---

## Shared setup and TestMain

Resource constructors and `BeforeAll` receive `*axiom.Runner`, so they can call
`custom.Get(runner)` directly. Fixtures, test actions, and plugins use
`custom.Get(cfg.Runner)`. The flags resource loads on demand; no loading hook is needed.

If you use `TestMain`, call `flag.Parse()` before `axiom.RunPackage(m, runner)`:
`RunPackage` runs `BeforeAll` hooks before `m.Run`.

---

## Inspecting flags

`testflags.Get(runner)` returns the snapshot. Its `Get(name)` returns a value and
presence flag; `Lookup(name)` returns an entry, and `All()` returns entries
sorted by name.

Each entry has `Name`, `Value`, `Usage`, `Default` (text), and `Set`.
`Set` distinguishes explicitly supplied `0`, `false`, or `""` from an omitted flag;
it also includes calls to `flag.Set`. All flags on the source are included,
including Go's `test.*` flags. Custom `flag.Getter` values keep their dynamic type;
other custom flag values expose their `String()` representation.

---

## Lifetime and errors

- Concurrent reads share one snapshot. Finish parsing and source mutation before
  concurrent access. Custom getter values are shallow copies; keep shared objects immutable.
- `Runner.Copy` and `Runner.Join` follow the usual [resource rules](../../docs/resource#join-semantics):
  uncached definitions create fresh state; cached snapshots are shared. Configure before execution.
- `Get` panics on lookup errors. `TryGet` returns an error for missing registration,
  an unparsed source, missing flags, or a type mismatch. An early read can succeed
  after parsing on the same runner. Invalid declarations and zero-value keys panic.
- With `go test ./... -args -axiom.custom=55`, every selected package with tests must
  register the flag, usually through a shared test-configuration package. `-args`
  forwards arguments; it does not make unknown flags valid.

---

## Installation

The plugin is distributed as a regular Go module and installed using standard Go tooling.

Add the plugin dependency using `go get`:

```shell
go get github.com/Nikita-Filonov/axiom/plugins/testflags
```

---

## Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testflags"
)

// Register the flag before go test parses its arguments.
var custom = testflags.Int(
	testflags.WithName("axiom.custom"),
	testflags.WithDefault(0),
	testflags.WithUsage("custom value"),
)

// Resource constructors can read the same typed flag as test cases.
var configuredValue = axiom.DefineResource("configured-value", func(r *axiom.Runner) (int, func(), error) {
	return custom.Get(r), nil, nil
})

var runner = axiom.NewRunner(
	// Register the flag snapshot and the resource on the same runner.
	axiom.WithRunnerResources(testflags.Resource(), configuredValue),
)

func TestCustomFlag(t *testing.T) {
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("custom flag")), func(cfg *axiom.Config) {
		// Get returns int; no cast or string key is needed at the call site.
		value := custom.Get(cfg.Runner)
		fromResource := configuredValue.Get(cfg.Runner)
		cfg.T().Logf("flag=%d resource=%d", value, fromResource)

		// Set tells whether the value was supplied explicitly, even if it is 0.
		entry, _ := testflags.Get(cfg.Runner).Lookup("axiom.custom")
		cfg.T().Logf("explicitly set: %t", entry.Set)
	})
}
```

Run with `go test -v -axiom.custom=55`. The test prints `flag=55 resource=55`and `explicitly set: true`.
