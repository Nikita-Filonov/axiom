# 📘 Resource

---

## 📑 Table of Contents

- [Key characteristics](#key-characteristics)
- [Resource lifecycle](#resource-lifecycle)
- [Resource API](#resource-api)
- [Join semantics](#join-semantics)
- [Concurrency model](#concurrency-model)
- [Registering resources](#registering-resources)
- [Example](#example)
- [Typed keys](#typed-keys)
- [Resources vs Fixtures](#resources-vs-fixtures)
- [When to use a Resource](#when-to-use-a-resource)
- [When not to use a Resource](#when-not-to-use-a-resource)
- [Summary](#summary)

---

## Overview

A `Resource` is a long-lived, lazily evaluated dependency bound to the **Runner lifecycle**, not to an individual test
case. A resource is created on first access, cached for the lifetime of the runner, and cleaned up during runner teardown.

Resources are designed for **infrastructure-level dependencies** such as clients, connections, servers, or shared
external systems.

A resource can also own an empty [Cache](../cache), while ordinary fixtures populate it using the current test's
Config. The cache coordinates value creation without changing resource or fixture lifecycle semantics.

Unlike fixtures, resources:

- are **shared across all test cases**
- **persist across retries**
- are cleaned up **only during runner teardown**
- are not reset between test attempts

---

## Key characteristics

A `Resource` has the following guarantees:

- **Lazy evaluation** — a resource is not created unless explicitly requested
- **Single constructor execution** — on a cache miss, the resource constructor runs at most once per resource name
- **Single active instance** — all callers observe the same cached resource instance
- **Runner-level caching** — the cached resource is reused across:
    - multiple test cases
    - retries of the same test case
- **Deterministic teardown** — cleanup is executed once during a single runner lifecycle
- **Safe concurrency** — concurrent access is coordinated so the constructor, cache write, and cleanup registration
  happen once

---

## Resource lifecycle

```
Runner start
   ↓
First GetResource call
   ↓
Resource is created and cached
   ↓
Used by any number of test cases
   ↓
Used across retries
   ↓
Runner finishes
   ↓
User AfterAll hooks run
   ↓
Resource cleanup is executed once
```

When resources depend on other resources, cleanup runs in reverse setup order:

```text
db setup
client setup
session setup
session cleanup
client cleanup
db cleanup
```

Resource cleanups are stored on a dedicated stack separate from user `AfterAll` hooks. User `AfterAll` hooks run first
and can still observe live resources; Axiom then drains the resource cleanup stack in LIFO order. This is the same
pattern fixtures use at the test scope, but at the runner scope.

If a resource constructor returns an error, its cleanup is not registered and the value is not cached. The error is
cached for the lifetime of the runner — see [Concurrency model](#concurrency-model).

---

## Resource API

```go
type Resource func (r *Runner) (value any, cleanup func (), err error)
```

- `value` — the resource instance
- `cleanup` — optional teardown logic, executed once during runner teardown, after user `AfterAll` hooks
- `err` — resource initialization error

Resources are accessed via:

```go
axiom.GetResource[T](runner, name)
axiom.MustResource[T](runner, name)
```

---

## Join semantics

`Resources.Join(other)` merges both resource definition and resource state:

- `Registry` is merged by key
- `Cache` is merged by key
- cleanup callbacks are copied
- if the same key exists in both, values from `other` override base values

This intentionally lets a joined runner reuse resources initialized before the join. It has its own
`BeforeAll`/`AfterAll` lifecycle, while inherited cache values still point to the original instances.

### Practical implications

- A warm join copies both the resource pointer and registered cleanup callback. Each runner executes its own cleanup
  stack at teardown. If both runners finish, the same callback may run once in each lifecycle.
- The source and joined runners refer to the same resource value. Each runner's teardown follows its own lifecycle,
  so callers can choose cleanup behavior that matches the intended sharing pattern.
- A failed constructor's cached error is not copied into a new runner because failed values have no cache entry. The
  joined runner can attempt construction again.

---

## Concurrency model

`GetResource` is safe to call concurrently from multiple goroutines.

Under concurrent access:

- the resource constructor is executed **at most once** for a cache miss
- all successful callers observe the **same cached instance**
- if initialization fails, all callers observe the **same cached error**
- cleanup is registered **only once** for a resource constructed within that runner
- cleanup is executed **only once** during that runner's teardown

The cleanup contract is:

> **Within one runner lifecycle, a constructed resource registers one cleanup that runs at teardown.**

Resource values themselves must still be safe for the way tests use them. If parallel tests share a resource and mutate
it, the resource must provide its own synchronization.

Initialization errors are cached for the runner lifetime. A failed resource constructor is not retried automatically by
subsequent `GetResource` calls.

Resource cleanup runs after user `AfterAll` hooks, so `AfterAll` hooks can still observe live resources. Cleanup is
still guaranteed if an `AfterAll` hook panics.

---

## Registering resources

Resources are registered at the **Runner level**.

```go
runner := axiom.NewRunner(
    axiom.WithRunnerResource("client", ClientResource),
)
```

There are no case-local resources by design.

---

## Example

The following example demonstrates that a resource is bound to the **runner lifecycle** and can be accessed from
**any place where the runner is available**, not only from inside a test case.

The resource is created lazily on first access, reused across test cases, and cleaned up once during runner teardown.

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

// -----------------------------------------------------------------------------
// Resources
// -----------------------------------------------------------------------------

// ClientResource — a shared infrastructure dependency.
func ClientResource(r *axiom.Runner) (any, func(), error) {
	fmt.Println("creating client")

	client := "shared-client"

	cleanup := func() {
		fmt.Println("closing client")
	}

	return client, cleanup, nil
}

// -----------------------------------------------------------------------------
// Example usage
// -----------------------------------------------------------------------------

func TestResourceLifecycle(t *testing.T) {

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("client", ClientResource),
	)

	// -------------------------------------------------------------------------
	// Resource access BEFORE any test cases
	// -------------------------------------------------------------------------

	// Resource can be accessed eagerly if needed.
	// It will be created here and reused later.
	client := axiom.MustResource[string](runner, "client")
	fmt.Println("pre-warmed:", client)

	// -------------------------------------------------------------------------
	// Test cases using the same resource
	// -------------------------------------------------------------------------

	cases := []axiom.Case{
		axiom.NewCase(axiom.WithCaseName("case A")),
		axiom.NewCase(axiom.WithCaseName("case B")),
	}

	for _, c := range cases {
		runner.RunCase(t, c, func(cfg *axiom.Config) {

			// Accessing the same resource inside test execution.
			client := axiom.MustResource[string](cfg.Runner, "client")
			fmt.Println("using in test:", client)
		})
	}
}

```

### Output

```
creating client
pre-warmed: shared-client
using in test: shared-client
using in test: shared-client
closing client
```

---

## Typed keys

`ResourceKey[T]` and `ResourceDef[T]` are the runner-scoped mirror of
[`FixtureKey` / `FixtureDef`](../fixture#typed-keys): a thin, additive layer over the string resource registry that
carries the registry name and the value type in one typed handle. They follow the same **two levels** (a bare key vs. a
self-describing `DefineResource`) and change nothing about the [resource lifecycle](#resource-lifecycle) — reads resolve
against the runner and stay runner-cached.

### API

```go
type TypedResource[T any] func(r *Runner) (T, func(), error)

type ResourceKey[T any]

func NewResourceKey[T any](name string) ResourceKey[T]
func (k ResourceKey[T]) Name() string
func (k ResourceKey[T]) Get(runner *Runner) T             // = MustResource[T](runner, k.Name())
func (k ResourceKey[T]) TryGet(runner *Runner) (T, error) // = GetResource[T](runner, k.Name())

type ResourceDef[T any]

func DefineResource[T any](name string, build TypedResource[T]) ResourceDef[T]
func (d ResourceDef[T]) Key() ResourceKey[T]
func (d ResourceDef[T]) Name() string
func (d ResourceDef[T]) Get(runner *Runner) T
func (d ResourceDef[T]) TryGet(runner *Runner) (T, error)
```

- `Get` returns the value directly and panics if the resource is missing or fails to build; `TryGet` returns the error
  instead. Both inherit the full lifecycle above: single construction, runner-level caching, deterministic teardown.
- A zero-value key panics with `resource: key must be created with NewResourceKey`.

### Registration

```go
func WithRunnerResourceKey[T any](key ResourceKey[T], build TypedResource[T]) RunnerOption
func WithRunnerResources(defs ...ResourceRegistrar) RunnerOption
```

`WithRunnerResourceKey` registers a level-1 key with a constructor; `WithRunnerResources` registers a batch of
`DefineResource` definitions. A `nil` constructor panics with `resource: nil constructor`, and `ResourceRegistrar` is
closed to `ResourceDef` values by an intentionally unexported method.

### Example

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

type Pool struct{ name string }

// PoolResource is a shared, runner-scoped resource behind a typed key.
var PoolResource = axiom.DefineResource(
	"pool",
	func(r *axiom.Runner) (*Pool, func(), error) {
		return &Pool{name: "shared-pool"}, func() { fmt.Println("closing pool") }, nil
	},
)

func TestResourceKeyExample(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerResources(PoolResource),
	)

	// Constructed once, reused everywhere the runner is available.
	pool := PoolResource.Get(runner)
	fmt.Println("pre-warmed:", pool.name)

	c := axiom.NewCase(axiom.WithCaseName("resource key"))

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		if pool, err := PoolResource.TryGet(cfg.Runner); err == nil {
			fmt.Println("using in test:", pool.name)
		}
	})
}
```

For the full typed-keys rationale — the boilerplate they remove, the two-level model, interoperability with the string
registry, and naming — see [Typed keys](../fixture#typed-keys) in the fixture documentation.

---

## Resources vs Fixtures

| Aspect            | Fixture                 | Resource                         |
|-------------------|-------------------------|----------------------------------|
| Scope             | Test attempt            | Runner                           |
| Cache lifetime    | Per test                | Across all tests                 |
| Retry behavior    | Fresh on each retry     | Reused across retries            |
| Cleanup timing    | After `AfterTest` hooks | After `AfterAll` hooks           |
| Intended usage    | Test data, setup        | Infrastructure, clients, servers |
| Concurrency focus | Single test             | Cross-test, concurrent access    |

---

## When to use a `Resource`

Use a `Resource` when:

- initialization is expensive
- the dependency is safe to share
- teardown is global and destructive
- retry isolation is not required

Examples:

- gRPC / HTTP clients
- database connection pools
- external service stubs
- embedded servers
- shared test infrastructure

---

## When **not** to use a `Resource`

Do **not** use a resource when:

* each test requires a clean instance
* teardown must run after each test
* the dependency is tightly coupled to test input
* retries must start from a clean state

In these cases, use a `Fixture`.

---

## Summary

`Resource` is a deliberate, low-level primitive with a strict and simple contract:

> **One runner → one active resource → one cleanup.**

It trades aggressive cleanup for determinism, safety, and a clear lifecycle boundary — making it suitable for
infrastructure-level dependencies in large test suites.
