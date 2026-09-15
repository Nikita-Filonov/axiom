# 📘 Fixture

A `Fixture` is a lazily evaluated resource used during a test execution. A fixture is created on first access, cached
for the remainder of the test attempt, and cleaned up automatically after the test finishes. Fixtures may depend on
other fixtures and can be defined at both Runner and Case level.

A fixture does **not** run unless the test accesses it with `GetFixture`. Each retry receives a fresh fixture lifecycle.

This model enables:

- deterministic setup/teardown
- lazy evaluation
- isolated retries
- reusable shared resources
- clean dependency injection via `GetFixture[T]`

---

## 📑 Table of Contents

- [Lifecycle guarantees](#lifecycle-guarantees)
- [Preloading fixtures with UseFixtures](#preloading-fixtures-with-usefixtures)
- [Example](#example)
- [Typed keys](#typed-keys)
- [Parameterised fixtures](#parameterised-fixtures)

---

## Lifecycle guarantees

Fixtures have a per-attempt lifecycle:

- a fixture is created on the first `GetFixture[T](cfg, name)` call
- the created value is cached for the current `Config`
- repeated access returns the cached value and does not register cleanup twice
- each retry attempt receives a fresh `Config`, fixture cache, and cleanup lifecycle
- fixture cleanups run automatically after the test body and `AfterTest` hooks finish
- cleanup remains inside the active runtime test wraps, so it may emit runtime steps, logs, assertions, and artefacts

When fixtures depend on other fixtures, cleanup runs in reverse setup order:

```text
db setup
user setup
session setup
session cleanup
user cleanup
db cleanup
```

Fixture cleanups are stored separately from user `AfterTest` hooks. User `AfterTest` hooks run first, then Axiom drains
fixture cleanups. That means `AfterTest` hooks can still observe live fixtures, while cleanup is still guaranteed if an
`AfterTest` hook panics.

The complete attempt boundary is:

```text
TestWrap enter
  BeforeTest
  test action
  AfterTest
  fixture cleanup (LIFO)
TestWrap exit
```

This matters for plugins that own attempt-scoped state. For example, a reporting plugin keeps its test context open
until fixture cleanup has emitted final attachments.

If a fixture setup returns an error, its cleanup is not registered and the value is not cached. If setup succeeds and
returns a cleanup, Axiom registers that cleanup even when the caller requested the wrong type, preventing leaked setup
work.

### Reporting cleanup as teardown

The cleanup returned by a fixture is scheduled automatically. `cfg.Teardown` does not schedule work; it executes the
provided function immediately and lets teardown runtime wrappers observe it. The two APIs can be composed:

```go
func ReportFixture(cfg *axiom.Config) (any, func(), error) {
	report := NewReport()

	return report, func() {
		cfg.Teardown("finalize flow report", func() {
			report.Finalize()
			cfg.Artefact(axiom.NewTextArtefact("flow-result.txt", report.String()))
		})
	}, nil
}
```

The cleanup still runs automatically at the end of the attempt. A runtime reporter may display the operation as a
teardown step and attach the emitted artefact to that step.

---

## Preloading fixtures with `UseFixtures`

In some cases, a test requires certain fixtures to be available before any test logic or steps are executed. For
example, data fixtures that must be loaded upfront, or side-effect-only fixtures whose return value is not used
directly.

For this purpose, Axiom provides `UseFixtures`, which can be attached as a test hook. `UseFixtures` eagerly evaluates
the specified fixtures at the beginning of the test, while preserving all standard fixture guarantees:

- lazy execution (only once per test attempt)
- caching
- automatic cleanup
- retry isolation

---

## Example

The following example demonstrates fixture definition, dependency resolution, caching, cleanup, and the `GetFixture`
API.

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

// -----------------------------------------------------------------------------
// Fixtures
// -----------------------------------------------------------------------------

// DBFixture — created once per test attempt, cleaned up automatically.
func DBFixture(cfg *axiom.Config) (any, func(), error) {
	db := fmt.Sprintf("db-%s", cfg.Case.ID)
	cleanup := func() { fmt.Println("closing:", db) }
	return db, cleanup, nil
}

// UserFixture — depends on the DB fixture via GetFixture.
func UserFixture(cfg *axiom.Config) (any, func(), error) {
	db := axiom.GetFixture[string](cfg, "db")
	user := fmt.Sprintf("user-from-%s", db)
	return user, nil, nil
}

// Data fixtures — side-effect-only fixtures that are typically preloaded.

func MongoDataFixture(cfg *axiom.Config) (any, func(), error) {
	fmt.Println("loading mongo data")
	return struct{}{}, func() {
		fmt.Println("cleanup mongo data")
	}, nil
}

func PostgresDataFixture(cfg *axiom.Config) (any, func(), error) {
	fmt.Println("loading postgres data")
	return struct{}{}, func() {
		fmt.Println("cleanup postgres data")
	}, nil
}

// -----------------------------------------------------------------------------
// Test using fixtures
// -----------------------------------------------------------------------------

func TestFixtureExample(t *testing.T) {

	runner := axiom.NewRunner(
		axiom.WithRunnerFixture("db", DBFixture), // global fixture

		// Data fixtures registered at Runner level.
		axiom.WithRunnerFixture("mongo-data", MongoDataFixture),
		axiom.WithRunnerFixture("postgres-data", PostgresDataFixture),

		// Preload fixtures before test execution.
		// This triggers fixture evaluation early while preserving caching and cleanup semantics.
		axiom.WithRunnerHooks(
			axiom.WithBeforeTest(
				axiom.UseFixtures("mongo-data", "postgres-data"),
			),
		),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("fixture example"),
		axiom.WithCaseFixture("user", UserFixture), // case-local fixture
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		// First access: fixture is created
		db := axiom.GetFixture[string](cfg, "db")
		fmt.Println("using:", db)

		// Cached access: no setup, no cleanup re-registration
		again := axiom.GetFixture[string](cfg, "db")
		fmt.Println("cached:", again)

		// Fixture with dependency
		user := axiom.GetFixture[string](cfg, "user")
		fmt.Println("user:", user)

		cfg.Step("validate", func() {
			fmt.Println("validating...")
		})
	})
}

```

---

## Typed keys

`FixtureKey[T]` and `FixtureDef[T]` are a thin, **additive** layer over the string-based fixture registry. They collapse
the usual `const` name plus `any` constructor plus typed getter wrapper into a single typed handle that carries the
registry **name** and the value **type** together. Nothing about the [lifecycle](#lifecycle-guarantees) changes — keys
write into the same registry, so fixtures still follow the per-attempt lifecycle described above.

Without a key, a package that exposes a fixture repeats the same three-part boilerplate:

```go
const ClientFixtureKey = "service-client"

func SetClientFixture(cfg *axiom.Config) (any, func(), error) { return newClient(cfg), nil, nil }

func GetClientFixture(cfg *axiom.Config) ServiceClient {
	return axiom.GetFixture[ServiceClient](cfg, ClientFixtureKey)
}
```

A typed key removes the `any` and the getter wrapper, and makes a name/type mismatch impossible by construction.

### Two levels

Both levels are opt-in.

**Level 1 — typed key.** A `FixtureKey[T]` is a typed handle for a name. Use it when the constructor lives elsewhere, is
registered separately, or is chosen dynamically.

```go
var ClientFixture = axiom.NewFixtureKey[ServiceClient]("service-client")

axiom.WithRunnerFixtureKey(ClientFixture, buildClient) // register somewhere
client := ClientFixture.Get(cfg)                       // read anywhere with a *Config
```

**Level 2 — self-describing definition.** A `FixtureDef[T]` bundles the key with its constructor, so a fixture is
declared once and both registered and read through the same value.

```go
var ClientFixture = axiom.DefineFixture("service-client", buildClient)

axiom.WithRunnerFixtures(ClientFixture) // batch registration
client := ClientFixture.Get(cfg)
```

`DefineFixture` is `NewFixtureKey` plus its constructor; `def.Key()` returns the underlying level-1 key.

### API

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
  `GetFixture[T]`, so it inherits the full lifecycle above: lazy setup, per-attempt caching, and LIFO cleanup.
- A zero-value key panics with `fixture: key must be created with NewFixtureKey`; an empty name panics with
  `fixture: key name must not be empty`.

### Registration

```go
func WithRunnerFixtureKey[T any](key FixtureKey[T], build TypedFixture[T]) RunnerOption
func WithRunnerFixtures(defs ...RunnerFixtureRegistrar) RunnerOption

func WithCaseFixtureKey[T any](key FixtureKey[T], build TypedFixture[T]) CaseOption
func WithCaseFixtures(defs ...CaseFixtureRegistrar) CaseOption
```

`WithRunnerFixtureKey` / `WithCaseFixtureKey` register a level-1 key with an explicit constructor at runner or case
scope; `WithRunnerFixtures` / `WithCaseFixtures` register a batch of level-2 definitions. A `nil` constructor panics with
`fixture: nil constructor`. `RunnerFixtureRegistrar` and `CaseFixtureRegistrar` are interfaces with an intentionally
unexported method, so only `FixtureDef` values (and parameterised per-case variants) can be passed.

### Interoperability

Typed keys are purely additive: they write into and read from the **same** registry as `WithRunnerFixture`, so the two
styles are interchangeable and a package can migrate one fixture at a time without breaking existing string-based access.

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

### Example — self-describing definition

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

### Configuring a fixture

There is no dedicated "configurable fixture" primitive, and there does not need to be. Beyond reading a
[`Resource`](../resource) inside the constructor or varying behavior per case through [`Params`](../params), the
idiomatic way to produce configured variants is an ordinary factory that returns a `FixtureDef`:

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

Each variant is a distinct registry entry with its own name and typed accessor, and no framework state is involved.

### Naming

Key names are registry names. Prefer namespaced, stable names, exactly as for string registrations:

```go
var ClientFixture = axiom.DefineFixture("cardsservice.client", buildCardsClient)
var DatabaseFixture = axiom.DefineFixture("cardsservice.database", buildCardsDatabase)
```

Avoid generic names such as `"client"` or `"db"` in shared packages, since a name collision silently overrides the
earlier registration.

> The same two-level pattern applies to runner-scoped [resources](../resource#typed-keys) (`ResourceKey` /
> `DefineResource`) and to per-test [context values](../context#typed-keys) (`ContextKey`).

---

## Parameterised fixtures

A `ParamFixture[P, T]` is a single typed fixture with a fixed name whose constructor is **parameterised** by `P`. Each
case selects one variant with `For(params)`; every variant shares the same key, so a toolset reads it back with a single
`Get`. This removes the "carry the variant in `Params`, then `switch` in the body" pattern: the param→constructor binding
lives at declaration and the selection is declarative in the case.

Like [typed keys](#typed-keys), a param fixture is purely additive — a variant is registered through the ordinary
case/runner fixture registries, so the lifecycle (lazy setup, per-attempt caching, LIFO cleanup, retry isolation) is
unchanged.

```go
type ParamFixture[P, T any]

func DefineParamFixture[P, T any](name string, build func(*Config, P) (T, func(), error)) ParamFixture[P, T]

func (f ParamFixture[P, T]) Name() string
func (f ParamFixture[P, T]) Key() FixtureKey[T]                // shared key, for the plain key API
func (f ParamFixture[P, T]) For(params P) CaseFixtureRegistrar // select a variant for one case
func (f ParamFixture[P, T]) Default(params P) RunnerOption     // runner-level fallback, overridden by For
func (f ParamFixture[P, T]) Get(cfg *Config) T                 // typed GetFixture[T]
```

A selected variant is attached to a case through `WithCaseFixtures`, the case-level counterpart of
`WithRunnerFixtures`:

```go
func WithCaseFixtures(fixtures ...CaseFixtureRegistrar) CaseOption
```

`WithCaseFixtures` accepts both a self-describing `FixtureDef` (a fixed variant) and `pf.For(params)` (a
parameterised variant). Because case fixtures are merged over runner fixtures, `pf.Default(params)` provides a
fallback that any individual case can override with its own `For`.

### Example

```go
// One key, many builders — the status is bound at selection time.
var ByStatus = axiom.DefineParamFixture(
	"atm-by-status",
	func(cfg *axiom.Config, status atmdata.Status) (StatefulATM, func(), error) {
		prepared, err := GRPCFactoryFixture.Get(cfg).CreateWithStatus(status)
		return prepared, nil, err
	},
)

// Each case selects its own variant, declaratively, next to the other options.
enabled := axiom.NewCase(
	axiom.WithCaseID("107401"),
	axiom.WithCaseName("atm status is enabled"),
	axiom.WithCaseFixtures(ByStatus.For(atmdata.StatusEnabled)),
)

blocked := axiom.NewCase(
	axiom.WithCaseID("107414"),
	axiom.WithCaseName("atm status is blocked"),
	axiom.WithCaseFixtures(ByStatus.For(atmdata.StatusBlocked)),
)

// The toolset reads the selected variant with a single accessor:
//   func (t *tools) PreparedATM() StatefulATM { return ByStatus.Get(t.cfg) }
```

### Param fixture vs factory

Both parameterise a fixture; they differ in how many registry entries exist and when the variant is chosen.

| | `ParamFixture` | Factory returning `FixtureDef` |
|---|---|---|
| Registry entries | one shared name | one distinct name per variant |
| Variant chosen | per case, via `For` | at declaration |
| Best when | each case uses exactly one variant | variants are known up front and may be used together |

Use a param fixture to replace a `switch` on a case parameter; use a [factory](#configuring-a-fixture) when you need
several named variants live at once (`PrimaryDB`, `ReplicaDB`).
