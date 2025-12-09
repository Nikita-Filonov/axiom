# 📘 Case

A `Case` represents a single test definition in Axiom. It specifies the test’s identity, metadata, fixtures, skip
behavior, retry policy, parameters, plugins, and execution context.

A `Case` is purely declarative: it describes the test but does not execute it. During execution, a `Runner` merges its
own configuration with the Case and produces a runtime `Config` object. This model enables composable configuration,
isolation, predictable overrides, and a consistent test lifecycle.

---

## Example

The following example demonstrates all Case-level options in a single, self-contained test.

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

// UserFixture — simple case-local fixture example.
func UserFixture(cfg *axiom.Config) (any, func(), error) {
	return "demo-user", nil, nil
}

func TestCaseExample(t *testing.T) {

	// -------------------------------------------------------------------------
	// Case definition
	// -------------------------------------------------------------------------

	c := axiom.NewCase(

		// Identification
		axiom.WithCaseID("AUTH-001"),
		axiom.WithCaseName("user can log in"),

		// Metadata
		axiom.WithCaseMeta(
			axiom.WithMetaTag("smoke"),
			axiom.WithMetaFeature("authentication"),
			axiom.WithMetaStory("valid login flow"),
			axiom.WithMetaSeverity(axiom.SeverityCritical),
			axiom.WithMetaLabel("component", "auth-service"),
		),

		// Skip rules (merged with Runner skip)
		// axiom.WithCaseSkip(axiom.WithSkipReason("temporarily disabled")),

		// Retry behavior (overrides Runner retry policy)
		axiom.WithCaseRetry(
			axiom.WithRetryTimes(3),
			axiom.WithRetryDelay(25),
		),

		// Arbitrary parameters passed into the test body
		axiom.WithCaseParams(struct {
			Username string
			Password string
		}{
			Username: "john",
			Password: "secret",
		}),

		// Context for this test only
		axiom.WithCaseContext(
			axiom.WithContextData("env", "dev"),
			axiom.WithContextData("request_id", "abc-123"),
		),

		// Case-specific plugins
		// axiom.WithCasePlugins(myPlugin),

		// Parallel or sequential execution
		axiom.WithCaseParallel(),
		// axiom.WithCaseSequential(),

		// Case-local fixtures
		axiom.WithCaseFixture("user", UserFixture),
	)

	// -------------------------------------------------------------------------
	// Running the Case
	// -------------------------------------------------------------------------

	runner := axiom.NewRunner()

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		// Parameters
		params := axiom.GetParams[struct {
			Username string
			Password string
		}](cfg)
		fmt.Println("Params:", params.Username, params.Password)

		// Fixtures
		user := axiom.GetFixture[string](cfg, "user")
		fmt.Println("Using fixture:", user)

		// Steps
		cfg.Step("login", func() {
			fmt.Println("Performing login...")
		})

		cfg.Step("validate session", func() {
			fmt.Println("Validating session...")
		})
	})
}

```
