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
- [Resources vs Fixtures](#resources-vs-fixtures)
- [When to use a Resource](#when-to-use-a-resource)
- [When not to use a Resource](#when-not-to-use-a-resource)
- [Summary](#summary)

---

## Overview

A `Resource` is a long-lived, lazily evaluated dependency bound to the **Runner lifecycle**, not to an individual test
case. A resource is created on first access, cached for the lifetime of the runner, and cleaned up **exactly once**
after all test cases have finished.

Resources are designed for **infrastructure-level dependencies** such as clients, connections, servers, or shared
external systems.

Unlike fixtures, resources:

- are **shared across all test cases**
- **persist across retries**
- are cleaned up **only in `AfterAll`**
- are not reset between test attempts

---

## Key characteristics

A `Resource` has the following guarantees:

- **Lazy evaluation** — a resource is not created unless explicitly requested
- **Single active instance** — at most one resource instance is stored in the runner cache
- **Runner-level caching** — the cached resource is reused across:
    - multiple test cases
    - retries of the same test case
- **Deterministic teardown** — cleanup is executed exactly once, via `AfterAll`
- **Safe concurrency** — concurrent access is supported without deadlocks

At the same time, a resource **does not guarantee**:

- that the underlying constructor is executed only once under concurrent access
- automatic cleanup of temporary instances created during race conditions

This is an intentional design choice.

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
Cleanup is executed once (AfterAll)
```

---

## Resource API

```go
type Resource func (r *Runner) (value any, cleanup func (), err error)
```

- `value` — the resource instance
- `cleanup` — optional teardown logic, executed once in `AfterAll`
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
- if the same key exists in both, values from `other` override base values

This means a joined runner may inherit already initialized resource instances from source runners.

### Practical implications

- Joining is no longer config-only for resources
- warm cache entries can be reused immediately after join
- this behavior is useful for pre-warmed infrastructure

---

## Concurrency model

`GetResource` is safe to call concurrently from multiple goroutines.

Under concurrent access:

- multiple goroutines may attempt to create the resource simultaneously
- **only one instance is stored and used**
- cleanup is registered **only for the stored instance**
- cleanup is executed **only once**

For this reason:

> **Resource cleanup functions must be idempotent and safe to run exactly once.**

They must **not** rely on being called for every constructor execution.

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

The resource is created lazily on first access, reused across test cases, and cleaned up once after all tests finish.

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

## Resources vs Fixtures

| Aspect            | Fixture             | Resource                         |
|-------------------|---------------------|----------------------------------|
| Scope             | Test attempt        | Runner                           |
| Cache lifetime    | Per test            | Across all tests                 |
| Retry behavior    | Fresh on each retry | Reused across retries            |
| Cleanup timing    | AfterTest           | AfterAll                         |
| Intended usage    | Test data, setup    | Infrastructure, clients, servers |
| Concurrency focus | Single test         | Cross-test, concurrent access    |

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
