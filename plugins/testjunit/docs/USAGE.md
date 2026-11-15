# 📘 JUnit Usage

## 📑 Table of Contents

- [Automatic output](#automatic-output)
- [Options](#options)
- [Explicit reporter](#explicit-reporter)
- [CI configuration](#ci-configuration)
- [Migration to v0.4.0](#migration-to-v040)

## Automatic output

```bash
go get github.com/Nikita-Filonov/axiom/plugins/testjunit@v0.4.0
```

Register one plugin on the package runner:

```go
var runner = axiom.NewRunner(
    axiom.WithRunnerPlugins(
        testjunit.Plugin(
            testjunit.WithSuiteName("employee/http/auth"),
            testjunit.WithOutputFile("junit-results/http-auth.xml"),
        ),
    ),
)
```

Each completed attempt updates the XML file. A late status correction updates it again.
No JUnit-specific `TestMain` or final `WriteFile` call is required.
Creating `Plugin(...)` prepares settings; output registration happens when Axiom installs it on a Config.

## Options

| Option                 | Default                             | Behavior                                                                                        |
|------------------------|-------------------------------------|-------------------------------------------------------------------------------------------------|
| `WithReporter(r)`      | New reporter per `Plugin(...)` call | Share a collector or export it manually. `nil` panics.                                          |
| `WithSuiteName(name)`  | `axiom`                             | Set suite name and testcase classname. Empty names panic.                                       |
| `WithOutputFile(path)` | Disabled                            | Enable incremental XML. Empty/blank paths panic; relative paths resolve at plugin construction. |
| `WithErrorHandler(fn)` | `log.Print`                         | Handle automatic output errors. `nil` is rejected at plugin construction.                       |

```go
testjunit.Plugin(
    testjunit.WithOutputFile("junit-results/auth.xml"),
    testjunit.WithErrorHandler(func(err error) {
        logger.Printf("JUnit output: %v", err)
    }),
)
```

Handlers may run concurrently and run without internal locks. They should not fail or skip tests.
Automatic output failures do not change the test exit code; a later publication retries writing the full snapshot.

## Explicit reporter

```go
reporter := testjunit.NewReporter()
plugin := testjunit.Plugin(
    testjunit.WithReporter(reporter),
    testjunit.WithSuiteName("employee/http/auth"),
)
```

Install `plugin` on the runner. After tests finish, export explicitly:

```go
if err := reporter.WriteFile("junit-results/auth.xml"); err != nil {
    return err
}
```

`reporter.Write(io.Writer)` exports to a caller-owned writer. Both methods return errors directly;
`WithErrorHandler` applies only to automatic output. Add `WithOutputFile` when both modes are needed.

To make export failure fail CI, retain explicit final export in `TestMain`:

```go
func TestMain(m *testing.M) {
    code := m.Run()
    if err := reporter.WriteFile("junit-results/auth.xml"); err != nil {
        log.Print(err)
        if code == 0 {
            code = 1
        }
    }
    os.Exit(code)
}
```

Keep any existing package resource lifecycle as well. Removing JUnit boilerplate does not replace that lifecycle.

## CI configuration

Use a fresh output directory per launch and a distinct filename per Go test package:

```text
junit-results/
├── http-auth.xml
├── http-users.xml
└── grpc-catalogservice.xml
```

An application helper can read its environment; the plugin itself does not read env:

```go
func JUnitPlugin(suiteName, filename string) axiom.Plugin {
    options := []testjunit.Option{testjunit.WithSuiteName(suiteName)}
    if directory := os.Getenv("JUNIT_RESULTS_DIR"); directory != "" {
        options = append(options, testjunit.WithOutputFile(filepath.Join(directory, filename)))
    }
    return testjunit.Plugin(options...)
}
```

GitLab artifact configuration:

```yaml
artifacts:
  when: always
  reports:
    junit: junit-results/*.xml
```

## Migration to v0.4.0

```go
// Before
testjunit.Plugin(reporter, testjunit.WithSuiteName("auth"))

// After: retain explicit reporter access
testjunit.Plugin(testjunit.WithReporter(reporter), testjunit.WithSuiteName("auth"))

// After: automatic output only
testjunit.Plugin(
    testjunit.WithSuiteName("auth"),
    testjunit.WithOutputFile("junit-results/auth.xml"),
)
```

Remove manual final export only when stderr logging is the desired output-error policy.
Attempt outcomes, retry representation, and XML fields are unchanged.
