# 📘 Toolset

`Toolset` is a typed convenience layer over `Local`.

It combines:

- a typed local key
- a builder function
- a `Bind` hook
- a typed `Use` adapter for the test body

`Toolset` is useful when the same group of runtime helpers should be prepared before the test body and consumed as a
typed bundle.

---

## Why `Toolset` exists

Without a toolset, every test body may end up repeating setup code:

```go
runner.RunCase(t, c, func(cfg *axiom.Config) {
	assertions := NewAssertions(cfg.T())
	client := getServiceClient(cfg.Runner)
	trace := getTraceClient(cfg.Runner)

	_ = assertions
	_ = client
	_ = trace
})
```

Or every package starts creating its own case context wrapper:

```go
type CardsCaseCtx struct {
	*axiom.Config
	Assertions *Assertions
	Client     CardsClient
	Trace      *TraceClient
}
```

`Toolset` keeps the core `Config` non-generic while still giving the test body a typed view:

```go
Tools.Use(func(cfg *axiom.ConfigWithTools[*ServiceTools]) {
	cfg.Tools.Assertions.NoError(err)
})
```

---

## API

```go
type ConfigWithTools[T any] struct {
	*Config
	Tools T
}

type Toolset[T any]

func NewToolset[T any](name string, build func(*Config) T) Toolset[T]

func (t Toolset[T]) Bind(cfg *Config)
func (t Toolset[T]) Use(action func(*ConfigWithTools[T])) TestAction
func (t Toolset[T]) Action(action func(*Config, T)) TestAction
func (t Toolset[T]) Get(cfg *Config) (T, bool)
func (t Toolset[T]) Must(cfg *Config) T
```

- `Bind` stores the built value in `cfg.Local`.
- `Use` reads that value and passes a typed `ConfigWithTools[T]` to the action.
- `Action` is an alternative adapter. It reads the same value and passes the original `*Config` and typed tools as 
  separate arguments.

---

## Example

```go
package example_test

import (
	"context"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/require"
)

type ServiceClient interface {
	Get(ctx context.Context) (string, error)
}

type Assertions struct {
	t *testing.T
}

func NewAssertions(t *testing.T) *Assertions {
	return &Assertions{t: t}
}

func (a *Assertions) NoError(err error) {
	require.NoError(a.t, err)
}

func NewServiceClient() ServiceClient {
	return fakeClient{}
}

type fakeClient struct{}

func (fakeClient) Get(context.Context) (string, error) {
	return "ok", nil
}

type ServiceTools struct {
	Assertions *Assertions
	Client     ServiceClient
}

var Tools = axiom.NewToolset("example.service.tools", func(cfg *axiom.Config) *ServiceTools {
	return &ServiceTools{
		Assertions: NewAssertions(cfg.T()),
		Client:     NewServiceClient(),
	}
})

func TestToolsetExample(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeTest(Tools.Bind),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("toolset example"),
	)

	runner.RunCase(t, c, Tools.Use(func(cfg *axiom.ConfigWithTools[*ServiceTools]) {
		resp, err := cfg.Tools.Client.Get(cfg.Context.RPC)

		cfg.Tools.Assertions.NoError(err)
		require.Equal(cfg.T(), "ok", resp)
	}))
}
```

The builder runs when `Tools.Bind` runs. When used as `WithBeforeTest(Tools.Bind)`, it runs inside the case subtest and
can safely use `cfg.T()` / `cfg.SubT`.

---

## Naming

Toolset names become local key names. Together with the toolset type, they define the local slot used for lookup and
error messages.

Prefer namespaced names:

```go
var Tools = axiom.NewToolset("cardsservice.tools", buildCardsTools)
var Tools = axiom.NewToolset("cashbackservice.tools", buildCashbackTools)
```

Avoid overly generic reusable names such as `"tools"` in shared packages.

---

## Generic Service Toolsets

When many packages follow the same pattern, keep the bundle generic and only provide the service-specific client
resolver.

```go
type ServiceTools[C any] struct {
	Assertions *Assertions
	Client     C
	Trace      *TraceClient
}

func NewServiceToolset[C any](
	name string,
	client func(*axiom.Runner) C,
) axiom.Toolset[*ServiceTools[C]] {
	return axiom.NewToolset(name, func(cfg *axiom.Config) *ServiceTools[C] {
		return &ServiceTools[C]{
			Assertions: NewAssertions(cfg.T()),
			Client:     client(cfg.Runner),
			Trace:      GetTraceClient(cfg.Runner),
		}
	})
}
```

Then a service package only declares:

```go
var Tools = NewServiceToolset(
	"cardsservice.tools",
	getCardsServiceClientResource,
)
```

---

## Lifecycle

`Toolset` values are stored in `Config.Local`.

That means they are:

- fresh for every retry attempt
- scoped to one `Config`
- not shared across cases
- not runner-level cache

Use `Resource` for shared infrastructure. Use `Fixture` for lazy setup with cleanup. Use `Toolset` for already prepared
runtime helpers that the test body should consume ergonomically.

---

## Concurrency

`Toolset` uses `Local`, so it follows the same concurrency model:

- bind before the test body
- read in the test body
- avoid concurrent mutation unless the stored value owns its synchronization

`Toolset` does not make the tools inside it thread-safe.

---

## Missing Bind

If `Use` or `Action` runs before `Bind`, the test fails with a missing local value.

The intended setup is:

```go
runner := axiom.NewRunner(
	axiom.WithRunnerHooks(
		axiom.WithBeforeTest(Tools.Bind),
	),
)
```

This keeps tool preparation explicit and visible at runner/case configuration level.

---

## `Use` vs `Action`

`Use` is useful when a single wrapper value reads better:

```go
Tools.Use(func(cfg *axiom.ConfigWithTools[*ServiceTools]) {
	cfg.Tools.Assertions.NoError(err)
})
```

`Action` is useful when the generic wrapper type is too noisy in the test body:

```go
Tools.Action(func(cfg *axiom.Config, tools *ServiceTools) {
	tools.Assertions.NoError(err)
})
```
