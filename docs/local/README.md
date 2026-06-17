# 📘 Local

`Local` is a small typed store attached to `Config`.

It is intended for runtime values that belong to one concrete test execution attempt: assertions, case-local helpers,
prepared clients, generated request data, temporary matchers, or anything a hook prepares for the test body.

Every retry receives a fresh `Config`, so every retry also receives fresh `Local` state.

---

## Why `Local` exists

Some values are not part of test declaration, and they are not long-lived infrastructure. They are created while the
test is running and should only be visible inside the current attempt.

Common examples:

- assertion helpers bound to the current `cfg.T()`
- a typed wrapper around a shared resource
- a generated id used by the current case
- a helper object prepared by `BeforeTest`
- temporary state shared between hooks and the test body

`Local` gives those values an explicit home without global state, `context.WithValue`, or suite-level mutable fields.

---

## API

```go
type LocalKey[T any]

func NewLocalKey[T any](name string) LocalKey[T]

func SetLocal[T any](cfg *Config, key LocalKey[T], value T)
func GetLocal[T any](cfg *Config, key LocalKey[T]) (T, bool)
func MustLocal[T any](cfg *Config, key LocalKey[T]) T
```

`LocalKey` is typed. The same name with a different type is a different key.

The same name with the same type points to the same local slot:

```go
first := axiom.NewLocalKey[string]("request-id")
second := axiom.NewLocalKey[string]("request-id")

// first and second address the same Local value.
```

Key names are part of the public contract. Prefer namespaced names in reusable packages:

```go
var AssertionsKey = axiom.NewLocalKey[*Assertions]("testassertions.assertions")
```

---

## Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/require"
)

type Assertions struct {
	t *testing.T
}

func NewAssertions(t *testing.T) *Assertions {
	return &Assertions{t: t}
}

func (a *Assertions) NoError(err error) {
	require.NoError(a.t, err)
}

var AssertionsKey = axiom.NewLocalKey[*Assertions]("example.assertions")

func TestLocalExample(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeTest(func(cfg *axiom.Config) {
				axiom.SetLocal(cfg, AssertionsKey, NewAssertions(cfg.T()))
			}),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("local example"),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		assertions := axiom.MustLocal(cfg, AssertionsKey)

		var err error
		assertions.NoError(err)
	})
}
```

`BeforeTest` is the right place for values that need `cfg.T()` or `cfg.SubT`, because it runs inside the case subtest.

---

## Lifecycle

```text
Runner.RunCase
  attempt #1
    Config
      Local

  attempt #2
    Config
      Local
```

`Local` is **per Config**, which means **per attempt**.

Values stored in one attempt are not visible in another attempt, even when both attempts belong to the same logical
case.

---

## Concurrency

`Local` is not a concurrent store.

The intended pattern is:

1. Bind values before the test body, usually in `BeforeTest`.
2. Read those values in the test body.
3. Avoid concurrent mutation.

If a test starts goroutines and mutates values stored in `Local`, the test owns the synchronization for those values.

Protecting the `Local` map alone would not make the values inside it safe. For example, a toolset may contain a map,
client, matcher, or assertion helper with its own concurrency rules.

---

## Local vs Other State

| Concept       | Lifetime             | Intended usage                                                 |
|---------------|----------------------|----------------------------------------------------------------|
| `Local`       | One Config / attempt | Runtime helpers and temporary state prepared for the test body |
| `Context`     | Runner + Case merge  | Execution contexts, cancellation boundaries, lightweight data  |
| `Fixture`     | One test attempt     | Lazy setup with cleanup, created on first access               |
| `Resource`    | Runner               | Shared infrastructure reused across tests and retries          |
| `Suite` field | Suite instance       | Suite-level state, not case-attempt-local                      |

---

## When to use `Local`

Use `Local` when:

- a value is prepared by a hook and consumed by the test body
- a helper must be bound to the current case-level `*testing.T`
- the value should be fresh for each retry
- global state or suite fields would make ownership unclear

Do not use `Local` for:

- long-lived clients shared across tests — use `Resource`
- lazy setup with cleanup — use `Fixture`
- request contexts and cancellation — use `Context`
- values that must be safe for concurrent mutation without additional locking

