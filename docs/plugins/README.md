# 📘 Plugins

This guide is for authors of `axiom.Plugin` implementations. For available modules, installation instructions, and
runner resources, see the [plugin and resource index](../../plugins).

## 📑 Table of Contents

- [Plugin contract](#plugin-contract)
- [Example: timing a test body](#example-timing-a-test-body)
- [Example: decorating a case name](#example-decorating-a-case-name)
- [Choosing an extension point](#choosing-an-extension-point)
- [Lifecycle and state](#lifecycle-and-state)
- [Testing and documentation](#testing-and-documentation)

## Plugin contract

An `axiom.Plugin` is a `func(*axiom.Config)`. A Runner applies its plugins first, followed by the Case plugins, in
registration order. Each invocation configures the given `Config`: it can change merged settings or register Runtime
wrappers and sinks. Test actions, steps, and fixtures run later. Keep installation cheap and free of irreversible work.

Axiom applies plugins to a planning `Config` to decide skip, retry, and parallel policy. It then builds a fresh
`Config` and applies plugins again for each attempt. A policy skip may have no attempt `Config`. Each attempt gets its
own `Local` state and fixture cache; the Runner and any collectors captured by plugin closures may be shared.

## Example: timing a test body

This example registers a wrapper during installation. The clock starts only when an attempt executes, and `defer`
reports its duration even if the body panics. The duration ends when `next` returns; it does not include later
`testing.T.Cleanup` callbacks. Policy skips do not execute test wrappers.

```go
package example_test

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

func TimingPlugin() axiom.Plugin {
	return func(cfg *axiom.Config) {
		cfg.Runtime.EmitTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(current *axiom.Config) {
				start := time.Now()
				defer func() {
					if t := current.T(); t != nil {
						t.Logf("test body took %s", time.Since(start))
					}
				}()
				next(current)
			}
		})
	}
}

func TestTimed(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(TimingPlugin()),
	)

	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("timed")), func(cfg *axiom.Config) {
		cfg.Step("work", func() {})
	})
}
```

The plugin uses only Axiom core and Go's standard library. Integrations that need external packages belong in their
own Go modules under `plugins/`.

## Example: decorating a case name

A plugin can also change the current `Config` directly. This function can be added to the example above:

```go
func PrefixName(prefix string) axiom.Plugin {
    return func (cfg *axiom.Config) {
        cfg.Case.Name = prefix + cfg.Case.Name
    }
}
```

Use it with `axiom.WithCasePlugins(PrefixName("[smoke] "))`. Every build starts from the declared Case, so retries see
one prefix rather than an accumulating series of prefixes; the declared Case name stays unchanged.

## Choosing an extension point

| Need                                             | Use                                                           |
|--------------------------------------------------|---------------------------------------------------------------|
| Change metadata or skip policy before execution  | Mutate the current `Config` during installation.              |
| Surround a test, step, setup, or teardown action | Register the corresponding `cfg.Runtime.Emit*Wrap` callback.  |
| Observe events, logs, assertions, or artefacts   | Register the corresponding `cfg.Runtime.Emit*Sink` callback.  |
| Share a lazy value across a Runner's tests       | Define a [runner resource](../resource) rather than a plugin. |

The [Runtime guide](../runtime) describes the wrapper and sink signatures. Earlier wrappers are outermost; sinks
receive values in registration order. Calling `cfg.Step`, `cfg.Setup`, or other test actions during installation runs
them too early. A sink or wrapper can use them later, when the attempt is executing.

For complete implementations, see [teststats' event recorder](../../plugins/teststats/plugin.go) and
[testtracing's test wrapper](../../plugins/testtracing/plugin.go).

## Lifecycle and state

- **Planning versus attempts.** `cfg.Execution.Attempt` is zero on the planning `Config` and one-based on attempts.
  `cfg.T()` can return the root `*testing.T` during planning, so it does not identify a running attempt. An event sink
  can observe a policy `case.skip` on the planning `Config`; a test wrapper cannot observe it.
- **Repeated installation.** Applying the same plugin at Runner and Case level can register callbacks twice on one
  `Config`. Decide and document whether that is intentional. If one registration is required, keep a typed
  `Config.Local` marker in `installation.go`, named with the full plugin import path. Scope it to the collector too
  when separate collectors should receive independent records.
  See [testtracing's installation](../../plugins/testtracing/installation.go).
- **Ownership and concurrency.** Attempt-local mutable state belongs on its `Config`. Synchronize collectors shared
  by parallel attempts. `Config.Local` is not safe for concurrent mutation. Register cleanup when the attempt runs;
  `testing.T.Cleanup` executes callbacks in reverse registration order.
- **Timing and failures.** The body may panic or call `FailNow` or `SkipNow`. Use `defer` or cleanup for state that must
  be released. A failure reported after a plugin's final cleanup cannot change a result that it already published;
  reporting plugins should document that limit.

For plugins that observe both Runner and Case scopes, test the order and option precedence explicitly. The first
installation may win for one collector, while two different collectors may both be valid.

## Testing and documentation

Give a new plugin its own `go.mod`, tests, and README. Document what it observes, changes, and owns, including option
precedence, duplicate installation, cleanup, and concurrency behavior. Keep runnable examples alongside the code.

Use `testify/assert` and `testify/require` in tests as required by [CONTRIBUTING.md](../../CONTRIBUTING.md). Cover
planning and attempt Configs, Runner and Case application, skips, retries, failures, cleanup, and parallel execution
where relevant. Run `go vet ./...`, `go test -race ./...`, and `go test -cover ./...` from the plugin's module
directory; each production package must reach 100.0% statement coverage.
