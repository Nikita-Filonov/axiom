# 📝 Logger Plugin (`testlogger`)

Forwards logs from `cfg.Log(...)` to Go's standard `log/slog` package.

## 📑 Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Custom logger](#custom-logger)
- [Log levels and context](#log-levels-and-context)

## Installation

```shell
go get github.com/Nikita-Filonov/axiom/plugins/testlogger
```

## Usage

`Plugin()` writes text logs to stdout, including Debug messages:

```go
runner := axiom.NewRunner(
    axiom.WithRunnerPlugins(testlogger.Plugin()),
)

runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("create user")), func(cfg *axiom.Config) {
    cfg.Log(axiom.NewInfoLog("starting test"))
    cfg.Log(axiom.NewDebugLog("preparing request"))
})
```

## Custom logger

Use `WithLogger` to supply a configured `*slog.Logger`. Its handler controls the output destination, format, filtering,
and attributes. For example, write JSON to stderr at Info level and above:

```go
logger := slog.New(
    slog.NewJSONHandler(
        os.Stderr,
        &slog.HandlerOptions{Level: slog.LevelInfo},
    ),
).With("service", "payments")

runner := axiom.NewRunner(
    axiom.WithRunnerPlugins(
        testlogger.Plugin(testlogger.WithLogger(logger)),
    ),
)
```

You can also pass your application's logger or `slog.Default()`. Options apply in order; the last logger wins.
A nil option or a nil configured logger causes a panic.

The logger is shared across test attempts. Its handler must support concurrent calls. The caller owns the output,
including closing files or flushing a custom handler.

## Log levels and context

| Axiom level      | slog level |
|------------------|------------|
| Debug            | Debug      |
| Info             | Info       |
| Warning          | Warn       |
| Error, Fatal     | Error      |
| Empty or unknown | Info       |

Logging a Fatal message does not stop or fail the test.

Each record uses the current `cfg.Context.Raw`. Custom handlers can read its values, including during cleanup after
cancellation. Test and step attributes are not added automatically.
