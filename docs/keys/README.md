# 📘 Typed Keys

Typed keys are a thin, **additive** layer over the string-based fixture and resource registries. They replace the
hand-written `const` name plus typed getter wrappers with a single typed handle, while staying fully interchangeable
with [`WithRunnerFixture`](../fixture) and [`WithRunnerResource`](../resource).

A typed key carries two things at once: the **registry name** and the **value type** `T`. Registration accepts a
constructor that returns `T` directly (no `any`), and reads return `T` without a call-site type assertion.

This layer changes **nothing** about the underlying lifecycle. Keys write into the same registries, so fixtures still
follow the per-attempt lifecycle and resources still follow the runner lifecycle.

---

## 📑 Table of Contents

- [Why typed keys exist](#why-typed-keys-exist)
- [Two levels](#two-levels)
- [Fixture API](#fixture-api)
- [Resource API](#resource-api)
- [Registration](#registration)
- [Interoperability](#interoperability)
- [Example — fixture key](#example--fixture-key)
- [Example — self-describing definitions](#example--self-describing-definitions)
- [Example — resource key](#example--resource-key)
- [Configuring a fixture](#configuring-a-fixture)
- [Relationship to other primitives](#relationship-to-other-primitives)
- [Naming](#naming)
- [Summary](#summary)

---

## Why typed keys exist

Without typed keys, a package that exposes a fixture or resource usually repeats the same three-part boilerplate: a
string constant, a constructor that returns `any`, and a typed getter wrapper.

```go
const ClientFixtureKey = "service-client"

func SetClientFixture(cfg *axiom.Config) (any, func(), error) {
	return newClient(cfg), nil, nil
}

func GetClientFixture(cfg *axiom.Config) ServiceClient {
	return axiom.GetFixture[ServiceClient](cfg, ClientFixtureKey)
}
```

The `any` in the constructor and the getter wrapper exist only to bridge the string registry and the concrete type. A
typed key collapses the triple into a single declaration where name, type, and constructor live together:

```go
var ClientFixture = axiom.DefineFixture(
	"service-client",
	func(cfg *axiom.Config) (ServiceClient, func(), error) {
		return newClient(cfg), nil, nil
	},
)

// register: axiom.WithRunnerFixtures(ClientFixture)
// read:     ClientFixture.Get(cfg)
```

The `any` is gone, the getter wrapper is gone, and a name/type mismatch is impossible by construction.

---

## Two levels

Typed keys come in two levels, and both are opt-in.

**Level 1 — typed key.** A `FixtureKey[T]` / `ResourceKey[T]` is a typed handle for a name. Use it when the constructor
lives elsewhere, is registered separately, or is chosen dynamically.

```go
var ClientFixture = axiom.NewFixtureKey[ServiceClient]("service-client")

// register somewhere:
axiom.WithRunnerFixtureKey(ClientFixture, buildClient)

// read anywhere with a *Config:
client := ClientFixture.Get(cfg)
```

**Level 2 — self-describing definition.** A `FixtureDef[T]` / `ResourceDef[T]` bundles the key with its constructor, so
a fixture is declared once and both registered and read through the same value.

```go
var ClientFixture = axiom.DefineFixture("service-client", buildClient)

axiom.WithRunnerFixtures(ClientFixture) // batch registration
client := ClientFixture.Get(cfg)
```

`DefineFixture` is `NewFixtureKey` plus its constructor; `def.Key()` returns the underlying level‑1 key.

---

## Fixture API

```go
type TypedFixture[T any] func(cfg *Config) (T, func(), error)

type FixtureKey[T any]

func NewFixtureKey[T any](name string) FixtureKey[T]

func (k FixtureKey[T]) Name() string
func (k FixtureKey[T]) Get(cfg *Config) T // = GetFixture[T](cfg, k.Name())

type FixtureDef[T any]

func DefineFixture[T any](name string, build TypedFixture[T]) FixtureDef[T]

func (d FixtureDef[T]) Key() FixtureKey[T]
func (d FixtureDef[T]) Name() string
func (d FixtureDef[T]) Get(cfg *Config) T
```

- `Get` resolves the fixture for the current test, constructing and caching it on first use. It is a typed shortcut for
  `GetFixture[T]`, so it inherits the full [fixture lifecycle](../fixture): lazy setup, per‑attempt caching, and LIFO
  cleanup.
- A zero-value key panics with `fixture: key must be created with NewFixtureKey`.
- An empty name panics with `fixture: key name must not be empty`.

---

## Resource API

```go
type TypedResource[T any] func(r *Runner) (T, func(), error)

type ResourceKey[T any]

func NewResourceKey[T any](name string) ResourceKey[T]

func (k ResourceKey[T]) Name() string
func (k ResourceKey[T]) Get(runner *Runner) T              // = MustResource[T](runner, k.Name())
func (k ResourceKey[T]) TryGet(runner *Runner) (T, error)  // = GetResource[T](runner, k.Name())

type ResourceDef[T any]

func DefineResource[T any](name string, build TypedResource[T]) ResourceDef[T]

func (d ResourceDef[T]) Key() ResourceKey[T]
func (d ResourceDef[T]) Name() string
func (d ResourceDef[T]) Get(runner *Runner) T
func (d ResourceDef[T]) TryGet(runner *Runner) (T, error)
```

- `Get` panics if the resource is missing or fails to build; `TryGet` returns the error instead. Both inherit the full
  [resource lifecycle](../resource): single construction, runner-level caching, deterministic teardown.
- A zero-value key panics with `resource: key must be created with NewResourceKey`.

---

## Registration

The registration options live next to their string-based siblings on the runner and follow the same shape.

```go
// Fixtures
func WithRunnerFixtureKey[T any](key FixtureKey[T], build TypedFixture[T]) RunnerOption
func WithRunnerFixtures(defs ...FixtureRegistrar) RunnerOption

// Resources
func WithRunnerResourceKey[T any](key ResourceKey[T], build TypedResource[T]) RunnerOption
func WithRunnerResources(defs ...ResourceRegistrar) RunnerOption
```

- `WithRunnerFixtureKey` / `WithRunnerResourceKey` register a level‑1 key with an explicit constructor.
- `WithRunnerFixtures` / `WithRunnerResources` register a batch of level‑2 definitions declared via `DefineFixture` /
  `DefineResource`.
- A `nil` constructor panics with `fixture: nil constructor` / `resource: nil constructor`.

`FixtureRegistrar` and `ResourceRegistrar` are interfaces with an intentionally unexported method: the set of
registrable definitions is closed to the package, so only `FixtureDef` / `ResourceDef` values can be passed.

---

## Interoperability

Typed keys are purely additive. They write into and read from the **same** registries as `WithRunnerFixture` and
`WithRunnerResource`, so the two styles are interchangeable and can coexist during an incremental migration.

A fixture registered with the plain string API is readable through a typed key that shares the same name:

```go
var Greeting = axiom.NewFixtureKey[string]("greeting")

runner := axiom.NewRunner(
	// string registration
	axiom.WithRunnerFixture("greeting", func(cfg *axiom.Config) (any, func(), error) {
		return "hello", nil, nil
	}),
)

runner.RunCase(t, c, func(cfg *axiom.Config) {
	// typed read of the same slot
	fmt.Println(Greeting.Get(cfg)) // "hello"
})
```

This means a package can migrate one fixture at a time without breaking existing string-based access.

---

## Example — fixture key

The following example registers a typed fixture, reads it through the key, and relies on the standard fixture cleanup.

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

type DB struct {
	dsn string
}

// DBFixture is a typed key: the name and the value type live together.
var DBFixture = axiom.NewFixtureKey[*DB]("db")

func TestFixtureKeyExample(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerFixtureKey(DBFixture, func(cfg *axiom.Config) (*DB, func(), error) {
			db := &DB{dsn: "db-" + cfg.Case.Name}
			cleanup := func() { fmt.Println("closing:", db.dsn) }
			return db, cleanup, nil
		}),
	)

	c := axiom.NewCase(axiom.WithCaseName("fixture key"))

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		db := DBFixture.Get(cfg) // typed, no assertion
		fmt.Println("using:", db.dsn)
	})
}
```

---

## Example — self-describing definitions

`DefineFixture` keeps the name, type, and constructor in one value. `WithRunnerFixtures` registers any number of them.

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

type Client struct{ base string }

func newClient() *Client { return &Client{base: "https://api"} }

// ClientFixture is declared once, then registered and read through the same value.
var ClientFixture = axiom.DefineFixture(
	"service-client",
	func(cfg *axiom.Config) (*Client, func(), error) {
		return newClient(), nil, nil
	},
)

func TestFixtureDefExample(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerFixtures(ClientFixture),
	)

	c := axiom.NewCase(axiom.WithCaseName("fixture def"))

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		client := ClientFixture.Get(cfg)
		fmt.Println("client base:", client.base)
	})
}
```

---

## Example — resource key

Resource keys mirror fixture keys but resolve against the runner, and expose `TryGet` for the error-returning path.

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
		pool := &Pool{name: "shared-pool"}
		return pool, func() { fmt.Println("closing pool") }, nil
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

---

## Configuring a fixture

There is no dedicated "configurable fixture" primitive, and there does not need to be. A fixture is configured by the
same mechanisms Axiom already provides:

- read runtime configuration from a [`Resource`](../resource) inside the constructor;
- vary behavior per case through [`Params`](../params);
- or **parameterize at declaration** with an ordinary factory that returns a `FixtureDef`.

The factory pattern is the idiomatic way to produce configured variants of the same fixture:

```go
type dbOptions struct{ readOnly bool }
type dbOption func(*dbOptions)

func WithReadOnly() dbOption { return func(o *dbOptions) { o.readOnly = true } }

// DBFixture builds a self-describing fixture configured by options.
func DBFixture(name string, opts ...dbOption) axiom.FixtureDef[*DB] {
	options := dbOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return axiom.DefineFixture(name, func(cfg *axiom.Config) (*DB, func(), error) {
		db := openDB(options.readOnly)
		return db, func() { db.Close() }, nil
	})
}

var PrimaryDB = DBFixture("db.primary")
var ReplicaDB = DBFixture("db.replica", WithReadOnly())

// runner: axiom.WithRunnerFixtures(PrimaryDB, ReplicaDB)
```

Each variant is a distinct registry entry with its own name and its own typed accessor, and no framework state is
involved.

---

## Relationship to other primitives

Typed keys do not add a new lifecycle; they are a typed front door to existing ones.

| You want                                   | Use                                                    |
|--------------------------------------------|--------------------------------------------------------|
| Per-test setup with cleanup                | [`Fixture`](../fixture), via `FixtureKey` / `DefineFixture`   |
| Runner-scoped shared infrastructure        | [`Resource`](../resource), via `ResourceKey` / `DefineResource` |
| A prepared bundle of helpers for the body  | [`Toolset`](../toolset)                                |
| Raw per-attempt typed slot on `Config`     | [`Local`](../local)                                    |

`FixtureKey` / `ResourceKey` are to fixtures and resources what `Toolset` is to `Local`: a typed convenience layer that
keeps the core registries string-based while giving call sites a typed view.

---

## Naming

Key names are registry names. Prefer namespaced, stable names, exactly as for string registrations and toolsets:

```go
var ClientFixture = axiom.DefineFixture("cardsservice.client", buildCardsClient)
var DatabaseFixture = axiom.DefineFixture("cardsservice.database", buildCardsDatabase)
```

Avoid generic names such as `"client"` or `"db"` in shared packages, since a name collision silently overrides the
earlier registration.

---

## Summary

Typed keys are a small, additive convenience layer:

> **One declaration carries the name and the type; registration and reads stay type-safe; the lifecycle is unchanged.**

They remove the `const` name plus `any` constructor plus typed getter triple, interoperate with the string registries
for incremental adoption, and leave [`Fixture`](../fixture) and [`Resource`](../resource) semantics exactly as they were.
