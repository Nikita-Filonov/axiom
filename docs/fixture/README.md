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
	db := fmt.Sprintf("db-%s", cfg.ID)
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
