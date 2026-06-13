# 📘 Suite

`Suite` is an optional execution boundary for grouping related Axiom cases under a single `Runner`.

A suite does not replace `Runner`, `Case`, or Go's native `testing` package. It only defines where a related group of
tests starts and finishes. The `Runner` still owns execution configuration, resources, fixtures, metadata, context,
runtime wrappers, plugins, retry policy, hooks, and reporting behavior.

A regular Axiom test can still be written as a plain Go test:

```go
func TestUserCanLogin(t *testing.T) {
	c := axiom.NewCase(axiom.WithCaseName("user can log in"))

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Step("login", func() {
			// test body
		})
	})
}
```

Use `Suite` when several related cases should be executed as one logical group with one shared runner boundary.

This model enables:

* grouping related cases without replacing native Go tests
* sharing one `Runner` configuration across suite methods
* runner-scoped resources shared by all cases in the suite
* case-attempt fixtures for each individual case execution
* consistent metadata, context, retry policy, plugins, and runtime wrappers
* true runner-level start/finish lifecycle around the whole group

---

## Example

The following example demonstrates a complete suite use case with a shared runner, runner-scoped resource, fixture,
metadata, context, hooks, runtime wrappers, plugin, case-level overrides, and multiple suite test methods.

```go
package example_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

// -----------------------------------------------------------------------------
// Example domain objects
// -----------------------------------------------------------------------------

type APIClient struct {
	BaseURL string
}

func (c *APIClient) CreateUser(email string) string {
	fmt.Println("create user:", email)
	return "user-1"
}

func (c *APIClient) Login(email string, password string) bool {
	fmt.Println("login:", email, password)
	return true
}

func (c *APIClient) BlockUser(userID string) {
	fmt.Println("block user:", userID)
}

type User struct {
	ID       string
	Email    string
	Password string
}

type LoginParams struct {
	RememberMe bool
}

// -----------------------------------------------------------------------------
// Runner-scoped resource
// -----------------------------------------------------------------------------

func APIClientResource(r *axiom.Runner) (any, func(), error) {
	client := &APIClient{
		BaseURL: "https://users-api.example.test",
	}

	cleanup := func() {
		fmt.Println("close api client")
	}

	return client, cleanup, nil
}

// -----------------------------------------------------------------------------
// Case-attempt fixture
// -----------------------------------------------------------------------------

func UserFixture(cfg *axiom.Config) (any, func(), error) {
	user := User{
		Email:    fmt.Sprintf("%s@example.test", cfg.Case.ID),
		Password: "password",
	}

	cleanup := func() {
		fmt.Println("cleanup user fixture:", user.Email)
	}

	return user, cleanup, nil
}

// -----------------------------------------------------------------------------
// Example plugin
// -----------------------------------------------------------------------------

func TimingPlugin() axiom.Plugin {
	return func(cfg *axiom.Config) {
		cfg.Runtime.TestWraps = append(cfg.Runtime.TestWraps, func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				start := time.Now()
				next(c)
				fmt.Println("case duration:", time.Since(start))
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Example hooks
// -----------------------------------------------------------------------------

func beforeAll(r *axiom.Runner) {
	fmt.Println("start users suite")
}

func afterAll(r *axiom.Runner) {
	fmt.Println("finish users suite")
}

func beforeTest(c *axiom.Config) {
	fmt.Println("before case:", c.Case.Name)
}

func afterTest(c *axiom.Config) {
	fmt.Println("after case:", c.Case.Name)
}

func beforeStep(c *axiom.Config, name string) {
	fmt.Println("before step:", name)
}

func afterStep(c *axiom.Config, name string) {
	fmt.Println("after step:", name)
}

// -----------------------------------------------------------------------------
// Shared runner configuration
// -----------------------------------------------------------------------------

var UsersRunner = axiom.NewRunner(

	// Metadata applied to all cases executed through this runner
	axiom.WithRunnerMeta(
		axiom.WithMetaEpic("platform"),
		axiom.WithMetaFeature("users"),
		axiom.WithMetaLayer("e2e"),
		axiom.WithMetaTag("suite"),
	),

	// Context shared by all cases unless a Case overrides or extends it
	axiom.WithRunnerContext(
		axiom.WithContextData("env", "staging"),
		axiom.WithContextData("service", "users-api"),
	),

	// Retry policy used by default for all cases in the suite
	axiom.WithRunnerRetry(
		axiom.WithRetryTimes(2),
		axiom.WithRetryDelay(50*time.Millisecond),
	),

	// Resource is created lazily, cached in the Runner, reused by suite cases,
	// and cleaned up when the suite runner finishes.
	axiom.WithRunnerResource("api-client", APIClientResource),

	// Fixture is created for a concrete case attempt and cleaned up after it.
	axiom.WithRunnerFixture("user", UserFixture),

	// Hooks are part of runner configuration.
	// BeforeAll/AfterAll wrap the whole suite because RunSuite provides
	// an explicit execution boundary.
	axiom.WithRunnerHooks(
		axiom.WithBeforeAll(beforeAll),
		axiom.WithAfterAll(afterAll),
		axiom.WithBeforeTest(beforeTest),
		axiom.WithAfterTest(afterTest),
		axiom.WithBeforeStep(beforeStep),
		axiom.WithAfterStep(afterStep),
	),

	// Plugins are applied to every Config built by this Runner.
	axiom.WithRunnerPlugins(
		TimingPlugin(),
	),

	// Runtime wrappers are shared by all cases in the suite.
	axiom.WithRunnerRuntime(
		axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				fmt.Println("[runner] before test body")
				next(c)
				fmt.Println("[runner] after test body")
			}
		}),

		axiom.WithRuntimeStepWrap(func(name string, next axiom.StepAction) axiom.StepAction {
			return func() {
				fmt.Println("[runner] step:", name)
				next()
			}
		}),
	),
)

// -----------------------------------------------------------------------------
// Suite definition
// -----------------------------------------------------------------------------

type UsersSuite struct {
	axiom.Suite
}

// -----------------------------------------------------------------------------
// Suite entrypoint
// -----------------------------------------------------------------------------

func TestUsersSuite(t *testing.T) {
	axiom.RunSuite(t, new(UsersSuite), axiom.WithSuiteRunner(UsersRunner))
}

// -----------------------------------------------------------------------------
// Suite test methods
// -----------------------------------------------------------------------------

func (s *UsersSuite) TestUserCanLogin() {
	c := axiom.NewCase(
		axiom.WithCaseName("user can log in"),

		// Case metadata extends Runner metadata.
		axiom.WithCaseMeta(
			axiom.WithMetaStory("login"),
			axiom.WithMetaTag("smoke"),
		),

		// Case context can extend or override Runner context.
		axiom.WithCaseContext(
			axiom.WithContextData("role", "customer"),
		),

		// Params describe input specific to this Case.
		axiom.WithCaseParams(LoginParams{
			RememberMe: true,
		}),
	)

	s.RunCase(c, func(cfg *axiom.Config) {
		user := axiom.GetFixture[User](cfg, "user")
		client := axiom.MustResource[*APIClient](cfg.Runner, "api-client")
		params := axiom.GetParams[LoginParams](cfg)

		cfg.Step("create user", func() {
			user.ID = client.CreateUser(user.Email)
		})

		cfg.Step("login as user", func() {
			ok := client.Login(user.Email, user.Password)

			fmt.Println("remember me:", params.RememberMe)
			fmt.Println("login result:", ok)
		})
	})
}

func (s *UsersSuite) TestAdminCanBlockUser() {
	c := axiom.NewCase(
		axiom.WithCaseName("admin can block user"),

		// This case belongs to the same suite and uses the same Runner,
		// but it still has its own Case definition and runtime Config.
		axiom.WithCaseMeta(
			axiom.WithMetaStory("blocking"),
			axiom.WithMetaTag("admin"),
		),

		axiom.WithCaseContext(
			axiom.WithContextData("role", "admin"),
		),
	)

	s.RunCase(c, func(cfg *axiom.Config) {
		user := axiom.GetFixture[User](cfg, "user")
		client := axiom.MustResource[*APIClient](cfg.Runner, "api-client")

		cfg.Step("create user", func() {
			user.ID = client.CreateUser(user.Email)
		})

		cfg.Step("block user", func() {
			client.BlockUser(user.ID)
		})
	})
}

// This method is ignored by RunSuite because it does not start with Test.
func (s *UsersSuite) helper() {}

// This method is ignored by RunSuite because it requires an argument.
func (s *UsersSuite) TestWithArgs(_ int) {}
```

In this example, `UsersRunner` remains the central execution configuration. The suite only defines a boundary around a
group of related test methods.

Axiom discovers exported zero-argument methods whose names start with `Test` and runs each method as a Go subtest.
Inside each suite method, `s.RunCase(...)` executes a normal Axiom `Case` through the suite runner.

The resulting test structure is:

```text
TestUsersSuite
  TestAdminCanBlockUser
    admin can block user
  TestUserCanLogin
    user can log in
```

The execution model is:

```text
RunSuite
  Runner ApplyStart
    suite test method
      s.RunCase
        Case execution
    suite test method
      s.RunCase
        Case execution
  Runner ApplyFinish
```

`Suite` is not a second runner and not a separate framework layer.

It is an optional grouping boundary around normal Axiom cases:

```text
Runner = shared execution configuration
Suite  = boundary for related test methods
Case   = individual test definition
Config = runtime state for one case attempt
```
