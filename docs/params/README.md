# 📘 Params

`Params` allow passing arbitrary structured data into a test. Parameters are defined at the `Case` level and become
available inside the test body through `GetParams[T]`. Parameters are immutable and specific to a single test execution.

Axiom performs a type check at runtime: if the stored parameters do not match the requested type `T`, the test panics
with a descriptive error. This ensures predictable, type-safe access to user-defined input-

This model enables:

- declarative, strongly typed test inputs
- separation of test definition from test execution
- static configuration (`Runner`) + dynamic variation (Case-level params)
- clear error reporting when types mismatch

## Example

```go
package example_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

type LoginParams struct {
	Username string
	Password string
}

func TestParamsExample(t *testing.T) {

	// -------------------------------------------------------------------------
	// Case with parameters
	// -------------------------------------------------------------------------

	c := axiom.NewCase(
		axiom.WithCaseName("login with parameters"),
		axiom.WithCaseParams(LoginParams{
			Username: "john",
			Password: "secret",
		}),
	)

	runner := axiom.NewRunner()

	runner.RunCase(t, c, func(cfg *axiom.Config) {

		// Strongly typed access
		params := axiom.GetParams[LoginParams](cfg)

		fmt.Println("username:", params.Username)
		fmt.Println("password:", params.Password)

		cfg.Step("validate", func() {
			fmt.Println("running validation...")
		})
	})
}

```
