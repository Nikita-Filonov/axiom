# 📘 Runtime

`Runtime` connects test execution to wrappers and sinks. Wrappers surround an action; sinks receive logs, assertions,
artefacts, or events. Use them to add reporting, logging, or instrumentation to a test.

## 📑 Table of Contents

- [Configuring runtime](#configuring-runtime)
- [Wrappers and sinks](#wrappers-and-sinks)
- [Execution order](#execution-order)
- [Operation values](#operation-values)
- [Scope and ownership](#scope-and-ownership)

## Configuring runtime

Register shared behavior on a `Runner` and case-specific behavior on a `Case`:

```go
runner := axiom.NewRunner(
	axiom.WithRunnerRuntime(
		axiom.WithRuntimeStepWrap(func(step axiom.Operation, next axiom.StepAction) axiom.StepAction {
			return func() {
				fmt.Println("step:", step.Name())
				next()
			}
		}),
		axiom.WithRuntimeLogSink(func(log axiom.Log) {
			fmt.Println(log.Level, log.Text)
		}),
	),
)

runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("create user")), func(cfg *axiom.Config) {
	cfg.Step("Send request", func() {
		cfg.Log(axiom.NewInfoLog("request sent"))
	})
})
```

`WithCaseRuntime` accepts the same options. A plugin registers callbacks on the attempt's `cfg.Runtime` using methods
such as `EmitStepWrap` and `EmitLogSink`.

## Wrappers and sinks

| Registration              | Receives                           | Purpose                                     |
|---------------------------|------------------------------------|---------------------------------------------|
| `WithRuntimeTestWrap`     | `next TestAction`                  | Surround the attempt's executable lifecycle |
| `WithRuntimeStepWrap`     | `Operation`, `next StepAction`     | Surround a `cfg.Step` action                |
| `WithRuntimeSetupWrap`    | `Operation`, `next SetupAction`    | Surround a `cfg.Setup` action               |
| `WithRuntimeTeardownWrap` | `Operation`, `next TeardownAction` | Surround a `cfg.Teardown` action            |
| `WithRuntimeLogSink`      | `Log`                              | Receive structured messages                 |
| `WithRuntimeAssertSink`   | `Assert`                           | Receive assertion facts                     |
| `WithRuntimeArtefactSink` | `Artefact`                         | Receive attachments and other outputs       |
| `WithRuntimeEventSink`    | `Event`                            | Receive raw execution facts                 |

A wrapper returns an action and calls `next` inside it. Use `defer` for work that must run when the action exits,
including during panic or `FailNow`:

```go
axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
	return func(cfg *axiom.Config) {
		started := time.Now()
		defer func() { fmt.Println("elapsed:", time.Since(started)) }()
		next(cfg)
	}
})
```

`Setup` and `Teardown` execute immediately at the call site. To schedule work before or after a test, use
[hooks](../hooks) or [fixture cleanup](../fixture).

## Execution order

Each attempt receives runtime callbacks merged from the Runner and Case, followed by callbacks registered by plugins.
Earlier wrappers are outermost; sinks receive values in registration order.

Test wrappers enclose test hooks and fixture cleanup:

```text
case.start event
  TestWrap enter
    BeforeTest hooks
    test action
    AfterTest hooks
    fixture cleanups (LIFO)
  TestWrap exit
case.finish event
```

This keeps attempt-level instrumentation active while hooks and cleanups emit steps or artefacts. `AfterTest` and
fixture cleanup run even if the body or `BeforeTest` panics. Axiom records a recovered body or `BeforeTest` panic as a
case failure. Panics from `AfterTest` or fixture cleanup propagate to the caller. Deferred cleanup also runs on
`SkipNow` and `FailNow`.

Step wrappers surround the step action, inside its hooks:

```text
step.start event
  BeforeStep hooks
    StepWrap enter
      step action
    StepWrap exit
  AfterStep hooks
step.finish event
```

Setup and teardown have their own wrappers and lifecycle events, without step hooks.
See [events](../events) for the event stream and [hooks](../hooks) for lifecycle callbacks.

## Operation values

`Operation` describes one `Step`, `Setup`, or `Teardown` call. It contains a human-readable name and optional typed
values. A library can use these values to describe an operation's purpose independently of its display text.

Define a key in the library that owns its meaning, using a package-qualified name:

```go
var roleKey = axiom.NewOperationKey[string]("example/validation.role")

cfg.Step("Verify response", check, roleKey.Value("check"))
cfg.Setup("Prepare data", prepare, roleKey.Value("prepare"))
cfg.Teardown("Remove data", cleanup, roleKey.Value("cleanup"))
```

Values are optional: `cfg.Step(name, action)` is sufficient for an ordinary named step.
An installed plugin reads values from the operation supplied to its wrapper:

```go
func LoggingPlugin() axiom.Plugin {
	return func(cfg *axiom.Config) {
		cfg.Runtime.EmitStepWrap(func(step axiom.Operation, next axiom.StepAction) axiom.StepAction {
			return func() {
				if role, ok := roleKey.TryGet(step); ok && role == "check" {
					cfg.Log(axiom.NewInfoLog(step.Name()))
				}
				next()
			}
		})
	}
}
```

Keys with the same name and Go type identify the same value. Use `TryGet` for optional values and `Get` when a value
is required:

| Key method              | Free function                        | Result                      |
|-------------------------|--------------------------------------|-----------------------------|
| `key.Value(value)`      | `WithOperationValue(key, value)`     | Option for one operation    |
| `key.TryGet(operation)` | `GetOperationValue(operation, key)`  | Value and presence flag     |
| `key.Get(operation)`    | `MustOperationValue(operation, key)` | Value, or panic when absent |

## Scope and ownership

Operation values belong to one call. Nested operations receive their own values without inheritance. Options are
applied in order before lifecycle events and hooks; the last value for a key wins. All wrappers for that call receive
the resulting operation.

An operation's name and value entries are immutable, so wrappers can retain the operation after execution. Payloads
are stored without deep copying: shared maps, slices, and pointers remain the caller's responsibility. Concurrent calls
have independent operation storage; shared payloads and callbacks still need appropriate synchronization.

Use operation values for per-call information and `Config.Local` or `Config.Context` for attempt-level state.
A plugin can capture its installation `cfg` when it needs that state. Operation values are delivered to wrappers;
hooks, logs, and events do not automatically receive them.
