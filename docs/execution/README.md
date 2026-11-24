# 📘 Execution

Axiom keeps the public `Runner` / `Case` / `Config` API on top of Go's ordinary `testing.T`.
The private `internal/execution` engine owns native subtests, retries, parallel scheduling, skip gates, and terminal
observation.

This page describes the contract between those layers and the extension points available to plugins.

## 📑 Table of Contents

- [The model at a glance](#the-model-at-a-glance)
- [Why there are three decisions](#why-there-are-three-decisions)
- [Run, Skip, and Exclude](#run-skip-and-exclude)
- [What executes](#what-executes)
- [Events and terminal attempts](#events-and-terminal-attempts)
- [Retries and parallel tests](#retries-and-parallel-tests)
- [Plugin patterns](#plugin-patterns)
- [Engine boundary](#engine-boundary)
- [Migration checklist](#migration-checklist)

## The model at a glance

One `RunCase` invocation has one identity and may prepare several Configs:

```text
RunCase
├── planning Config     Attempt.Number = 0
│   └── resolves name, retry, and parallel scheduling
│
└── attempt Config      Attempt.Number = 1, 2, ...
    ├── plugins resolve Decision
    ├── execution engine runs or omits callbacks
    └── terminal Attempt is delivered to attempt sinks
```

`RunID` is shared by the planning Config and every retry. Each actual attempt has a fresh Config, Local state,
fixture cache, plugin installation, and one-based `Number`.

The prepared value and terminal value are deliberately separate:

```go
// Available to plugins, wrappers, hooks, and the body.
cfg.Attempt // RunID + Number + Decision; Outcome is nil

// Delivered later to AttemptSink.
attempt // Same identity and Decision; terminal Outcome may be populated
```

`cfg.Attempt` is not rewritten at completion. This matters for wrappers such as `testtimeout`, whose body goroutine
may continue reading its Config after the native test has already finished.

## Why there are three decisions

Suppose a run selects only cases from the `smoke` suite:

| Case                                                  | Decision          | Report result                      |
|-------------------------------------------------------|-------------------|------------------------------------|
| case assigned to another suite                        | `DecisionExclude` | absent                             |
| selected case explicitly disabled with `WithCaseSkip` | `DecisionSkip`    | skipped                            |
| selected runnable case                                | `DecisionRun`     | passed, failed, or runtime-skipped |

The distinction is semantic, not textual. `"dependency unavailable"` and `"suite does not match"` are messages for
humans; integrations branch on `Decision.Kind`.

## Run, Skip, and Exclude

A `Decision` is an instruction resolved before submission to the engine. It is not the test result.

| Decision          | TestWrap | Hooks and body | Terminal `Outcome`         | Typical use                                  |
|-------------------|----------|----------------|----------------------------|----------------------------------------------|
| `DecisionRun`     | Yes      | Yes            | passed, failed, or skipped | ordinary selected test                       |
| `DecisionSkip`    | Yes      | No             | normally skipped           | selected test with unavailable functionality |
| `DecisionExclude` | No       | No             | nil                        | tag, environment, or inventory filter        |

### Declarative skip

Use `Skip` when the case was selected and should appear in reports as skipped:

```go
testCase := axiom.NewCase(
    axiom.WithCaseName("temporarily unavailable"),
    axiom.WithCaseSkip(
        axiom.SkipBecause("dependency unavailable"),
    ),
)
```

Runner-level skip is inherited. A Case can replace it or explicitly disable it with `WithSkipDisabled()`.

### Filter exclusion

A filtering plugin excludes a case during installation:

```go
func SuiteFilter(suite string) axiom.Plugin {
    return func(cfg *axiom.Config) {
        if cfg.Meta.Labels["suite"] != suite {
            cfg.Attempt.Exclude("suite does not match")
        }
    }
}
```

The first exclusion wins, including an empty reason. Exclusion takes precedence over merged Skip, so
`WithSkipDisabled()` cannot undo a filter decision.

Reasons are diagnostic text. Consumers distinguish decisions through `Decision.Kind`, never by comparing reason
strings.

### Runtime skip

A call from an executing hook or body is different:

```go
runner.RunCase(t, testCase, func(cfg *axiom.Config) {
    if serviceUnavailable() {
        cfg.T().Skip("service unavailable")
    }
})
```

Execution has already entered `DecisionRun`. Deferred `AfterTest`, fixture teardown, wrapper defers, and native cleanup
still unwind. The terminal outcome is skipped unless the native test also failed; failure takes precedence.

## What executes

### Ordinary run

```text
case.start
TestWrap enter
  BeforeTest
  body
  AfterTest
  fixture cleanup (LIFO)
TestWrap exit
case.finish
native child tests and testing.T.Cleanup finish
terminal Attempt → AttemptSink
```

`case.finish` closes Axiom's executable callback scope. It is not proof that child tests or native cleanup have
finished, and it does not contain the final status.

A normal callback return also does not imply success: the callback may have called `t.Error` or a child may fail later.

### Declarative skip

```text
case.skip
TestWrap enter
  continuation calls t.Skip(reason)
TestWrap deferred work
native cleanup registered by wrappers
terminal Attempt { DecisionSkip, Outcome: skipped }
```

`BeforeTest`, the body, `AfterTest`, and fixture teardown never start. Reporting wrappers may create their result and
register their own cleanup. Execution-only wrappers should forward directly to `next` without starting workers,
timers, fixture setup, or leak detection.

### Exclusion

```text
native Go subtest is created
└── t.Skip(reason) keeps it visible in go test output

no TestWrap
no hooks or body
case.excluded
terminal Attempt { DecisionExclude, Outcome: nil }
```

`case.excluded` is emitted after the native subtest ends. Its T is already finished, so observers must not register
cleanup on it.

A native `go test -run` deselection also produces no outcome. It retains the submitted Decision because the Go filter,
not Axiom policy, omitted the callback.

### Panic and cleanup rules

- Panics in `BeforeTest`, the body, or a TestWrap are converted into a native test failure.
- Panics in `AfterTest` or fixture teardown propagate.
- Once execution enters the body gate, deferred `AfterTest` and fixture teardown run on panic, `FailNow`, and runtime
  `SkipNow`.
- Final status is observed only after native children and cleanup can contribute failures.
- A sibling cleanup panic may start parent cleanup while another parallel attempt still has an active body,
  child subtest, or native cleanup. That unfinished attempt receives no terminal result.

## Events and terminal attempts

Raw events and attempt sinks answer different questions:

| API                      | Answers                                      | Final status?       |
|--------------------------|----------------------------------------------|---------------------|
| `WithRuntimeEventSink`   | What lifecycle fact happened?                | No                  |
| `WithRuntimeAttemptSink` | How did this attempt finish or get selected? | Yes, when delivered |

The main case events are:

| Event           | Meaning                                              |
|-----------------|------------------------------------------------------|
| `case.start`    | selected Run attempt entered execution               |
| `case.finish`   | executable callbacks unwound; native work may remain |
| `case.panic`    | engine recovered a callback panic                    |
| `case.skip`     | selected declarative skip, emitted before TestWrap   |
| `case.excluded` | filter exclusion completed without a test outcome    |

### Observe final attempts

```go
func AttemptReporter(report func(axiom.Attempt)) axiom.Plugin {
    return func(cfg *axiom.Config) {
        cfg.Runtime.EmitAttemptSink(func(attempt axiom.Attempt) {
            switch attempt.Decision.Kind {
            case axiom.DecisionExclude:
                // Filtered: intentionally absent from test-result totals.
                return
            }

            if attempt.Outcome == nil {
                // Native deselection has no Go outcome.
                return
            }

            report(attempt)
        })
    }
}
```

A delivered terminal Attempt contains:

```go
type Attempt struct {
    RunID    string
    Number   int
    Outcome  *AttemptOutcome
    Decision Decision
}
```

`AttemptOutcome` contains the native status, Go test name, start time, and cleanup time. `CleanupAt` excludes any later
wait for a safe parent delivery boundary.

### Delivery rules

- `Complete(Result)` and public attempt sinks receive at most one terminal value per attempt.
- Normal completion delivers one value; fatal process termination may deliver none.
- A sequential attempt normally delivers after its own `t.Run` completes.
- A parallel attempt may deliver later from a safe parent boundary, after `RunCase` has returned.
- There is no global ordering between parallel attempts.
- Sinks run in registration order for one result and must be concurrency-safe.
- Sinks are observational: do not call test actions, `FailNow`, `SkipNow`, or register cleanup.
- Treat the Attempt and its Outcome as read-only.
- A timeout wrapper may leave its body goroutine running after terminal delivery. That goroutine keeps the immutable
  prepared `cfg.Attempt` and never observes the terminal Outcome through Config.

Incremental reporters should not postpone all persistence until the terminal sink. A process crash can prevent late
delivery. JUnit and teststats therefore keep provisional cleanup-time records and reconcile them from the terminal
Attempt when it arrives.

## Retries and parallel tests

A retry is started only when all of these are true:

```text
Decision == Run
Result.Kind == Completed
Outcome.Status == Failed
attempt number < MaxAttempts
```

Skips, exclusions, native deselections, passed attempts, and pending parallel attempts do not retry. A failed attempt
still makes the overall Go test fail even if a later retry passes.

Preparation is lazy: attempt 2 is not built until attempt 1 has completed and needs a retry.

### Parallel retry layout

Parallel retries use one native parallel container and sequential attempts inside it:

```text
parallel case container
├── attempt 1
├── attempt 2
└── attempt 3
```

Different Cases may still overlap. A first-attempt Skip or Exclude bypasses the parallel container and does not call
`t.Parallel`.

Use Axiom's Parallel options together with Retry. Calling `cfg.T().Parallel()` manually can suspend an attempt before
the engine reaches its retry decision. The engine later observes that attempt but does not wait while holding its
parent open and does not start another retry.

A native `-run` filter may omit the entire parallel container. In that case no attempt is submitted and no attempt
sink fires.

## Plugin patterns

### Filtering plugin

Use Exclude, not Skip, for selection:

```go
func RequiredTag(tag string) axiom.Plugin {
    return func(cfg *axiom.Config) {
        if !slices.Contains(cfg.Meta.Tags, tag) {
            cfg.Attempt.Exclude("required tag is absent")
        }
    }
}
```

This keeps filtered cases out of Allure, JUnit, and teststats without parsing reason text.

### Reporting wrapper

A reporting wrapper may surround Run and Skip:

```go
func LogReportingScope() axiom.Plugin {
    return func(cfg *axiom.Config) {
        cfg.Runtime.EmitTestWrap(func(next axiom.TestAction) axiom.TestAction {
            return func(current *axiom.Config) {
                current.T().Log("report: start")
                defer current.T().Log("report: finish")

                next(current)
            }
        })
    }
}
```

For `DecisionSkip`, `next` calls native `t.Skip`. Defers and wrapper-owned native cleanup still execute.

### Execution-only wrapper

Instrumentation that starts work must bypass non-Run decisions:

```go
func WorkerScope() axiom.Plugin {
    return func(cfg *axiom.Config) {
        cfg.Runtime.EmitTestWrap(func(next axiom.TestAction) axiom.TestAction {
            return func(current *axiom.Config) {
                if current.Attempt.Decision.Kind != axiom.DecisionRun {
                    next(current)
                    return
                }

                current.T().Log("worker: start")
                defer current.T().Log("worker: stop")

                next(current)
            }
        })
    }
}
```

`testtimeout` and `testleaks` follow this pattern. Excluded attempts never enter either wrapper.

### Plugin installation rules

- Plugins run once on the planning Config and again on every prepared attempt Config.
- Planning has `Attempt.Number == 0` and emits no attempt lifecycle events.
- Every Run, Skip, and Exclude decision uses a fresh one-based attempt Config.
- Runner plugins run before Case plugins.
- `Attempt.Exclude` is the only supported decision mutation during installation.
- After installation, `cfg.Attempt` is the immutable prepared snapshot.
- A plugin that directly replaces policies such as `cfg.Retry` must preserve their documented invariants; Axiom does
  not silently normalize plugin mutations afterward.

## Engine boundary

`internal/execution` behaves like a standalone library. It imports only the standard library and knows nothing about
Runner, Case, Config, fixtures, or plugins.

The only root-package entry is:

```go
execution.Run(parent, plan)
```

`Plan` owns scheduling parameters and a lazy `Prepare(PrepareContext) Spec` callback. A `Spec` is the flat contract of
one attempt:

```go
type Spec struct {
    Name     string
    Decision Decision
    Parallel bool

    Bind func(*testing.T)

    Around func(func())

    Before  func()
    Body    func()
    After   func()
    Release func()

    Observe  func(Event)
    Complete func(Result)
}
```

`Observe` carries non-terminal engine facts. `Complete` carries at most one terminal Result after confirmed native
completion or selection.

| File                               | Responsibility                                    |
|------------------------------------|---------------------------------------------------|
| `execution.go`                     | adapt Axiom Configs, events, and terminal results |
| `internal/execution/plan.go`       | scheduling and lazy preparation contract          |
| `internal/execution/spec.go`       | callbacks for one immutable attempt               |
| `internal/execution/decision.go`   | Run, Skip, and Exclude instructions               |
| `internal/execution/run.go`        | parallel container and retry loop                 |
| `internal/execution/attempt.go`    | native subtest, callback gate, and unwinding      |
| `internal/execution/completion.go` | confirmed completion and retry eligibility        |
| `internal/execution/event.go`      | non-terminal lifecycle facts                      |
| `internal/execution/result.go`     | terminal selection/result contract                |
| `internal/execution/outcome.go`    | native status and timing                          |

The engine does not replace `testing.T`, intercept its methods, recover arbitrary goroutine panics, or cancel
wrapper-owned goroutines.

## Migration checklist

For plugins written against the previous lifecycle:

- Replace filtering assignments to `cfg.Skip` with `cfg.Attempt.Exclude(reason)`.
- Keep explicit feature disabling on `Skip`.
- Remove comparisons against skip reason strings.
- Use `Attempt.Decision.Kind` or `case.excluded` to identify filtering.
- Expect TestWrap to run for declarative Skip.
- Bypass execution-only instrumentation when `Decision.Kind != DecisionRun`.
- Remove separate `case.skip` result-creation workarounds that now duplicate reporting wrappers.
- Read final status from the Attempt passed to an attempt sink, not from `cfg.Attempt.Outcome`.
- Do not register cleanup from `case.excluded` or an attempt sink.
- Do not assume Skip uses the planning Config or attempt number zero.
- Keep provisional persistence when a process crash must not erase earlier completed work.

Core and affected plugin modules should be released together because these are intentional lifecycle changes.

## Related documentation

- [Skip](../skip) — inheritance, override semantics, and runtime skip.
- [Retry](../retry) and [Parallel](../parallel) — public scheduling options.
- [Runtime](../runtime) — wrappers and sinks.
- [Events](../events) — raw lifecycle facts.
- [Plugins](../plugins) — plugin installation and state ownership.

