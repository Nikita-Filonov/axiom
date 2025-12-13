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

// -----------------------------------------------------------------------------
// Test using fixtures
// -----------------------------------------------------------------------------

func TestFixtureExample(t *testing.T) {

	runner := axiom.NewRunner(
		axiom.WithRunnerFixture("db", DBFixture), // global fixture
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
