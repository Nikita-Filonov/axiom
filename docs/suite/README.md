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
* sharing one `Runner` configuration across registered suite tests
* runner-scoped resources shared by all cases in the suite
* case-attempt fixtures for each individual case execution
* consistent metadata, context, retry policy, plugins, and runtime wrappers
* true runner-level start/finish lifecycle around the whole group

---

## Minimal Example

```go
package example_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

var UsersRunner = axiom.NewRunner()

type UsersSuite struct { axiom.Suite }

func TestUsersSuite(t *testing.T) {
	suite := axiom.NewSuite(t, new(UsersSuite), axiom.WithSuiteConfigRunner(UsersRunner))
	suite.Test("user can log in", (*UsersSuite).UserCanLogin)
	suite.Test("admin can block user", (*UsersSuite).AdminCanBlockUser)
	suite.Run()
}

func (s *UsersSuite) UserCanLogin() {
	s.RunCase(axiom.NewCase(axiom.WithCaseName("user can log in")), func(cfg *axiom.Config) {
		cfg.Step("login", func() {
			// test body
		})
	})
}

func (s *UsersSuite) AdminCanBlockUser() {
	s.RunCase(axiom.NewCase(axiom.WithCaseName("admin can block user")), func(cfg *axiom.Config) {
		cfg.Step("block user", func() {
			// test body
		})
	})
}
```

The only Go test discovered by `go test` is the top-level `TestUsersSuite`. Suite tests are registered explicitly with
`suite.Test(...)`; Axiom does not discover receiver methods by name.

Suite methods do not need a `Test` prefix. They become executable suite tests only when they are registered:

```go
suite.Test("user can log in", (*UsersSuite).UserCanLogin)
```

`NewSuite` accepts any non-nil pointer implementing `axiom.TestingSuite`. Embedding `axiom.Suite` is the standard way to
provide that contract:

```go
type BaseSuite struct {
	axiom.Suite
}

type UsersSuite struct {
	BaseSuite
}
```

`SuiteConfig` controls suite-level configuration. If no runner is provided, Axiom creates a default runner:

```go
suite := axiom.NewSuite(t, new(UsersSuite))
```

Use `WithSuiteConfigRunner` when the suite should run through a shared runner:

```go
suite := axiom.NewSuite(t, new(UsersSuite), axiom.WithSuiteConfigRunner(UsersRunner))
```

Use `WithSuiteTestRunner` when one registered suite test should run through a different runner:

```go
suite.Test(
	"user can log in",
	(*UsersSuite).UserCanLogin,
	axiom.WithSuiteTestRunner(LoginRunner),
)
```

The test runner replaces the suite runner for that registered suite test. If it should include the suite runner behavior,
compose it explicitly:

```go
var LoginRunner = UsersRunner.Join(
	axiom.NewRunner(
		axiom.WithRunnerMeta(axiom.WithMetaStory("login")),
	),
)
```

## Parallel Suite Tests

Parallel suite tests are explicit by design.

`NewSuite(t, new(UsersSuite))` uses one suite instance and intentionally stays sequential. A shared suite instance has
mutable runtime fields such as `SubT` and `Runner`, so Axiom does not try to make it parallel with hidden cloning,
locking, or reflection tricks.

Use `NewSuiteFactory` when registered suite tests should run in parallel. The factory creates a fresh suite instance for
every registered test, so `SubT` and `Runner` belong to that test only:

```go
suite := axiom.NewSuiteFactory(
	t,
	func() *UsersSuite {
		return new(UsersSuite)
	},
	axiom.WithSuiteConfigRunner(UsersRunner),
	axiom.WithSuiteConfigParallel(),
)

suite.Test("user can log in", (*UsersSuite).UserCanLogin)
suite.Test("admin can block user", (*UsersSuite).AdminCanBlockUser)
suite.Run()
```

Use `WithSuiteConfigParallel` to mark all registered suite tests as parallel:

```go
suite := axiom.NewSuiteFactory(
	t,
	func() *UsersSuite { return new(UsersSuite) },
	axiom.WithSuiteConfigParallel(),
)
```

Use `WithSuiteTestParallel` to mark only one registered suite test as parallel:

```go
suite.Test(
	"user can log in",
	(*UsersSuite).UserCanLogin,
	axiom.WithSuiteTestParallel(),
)
```

Parallel suite tests require `NewSuiteFactory`. If you pass `WithSuiteConfigParallel` or `WithSuiteTestParallel` to a
shared-instance suite, Axiom panics early instead of running with unsafe shared state.

The model is:

```text
NewSuite
  one suite instance
  registered suite tests are sequential

NewSuiteFactory
  fresh suite instance per registered suite test
  registered suite tests may run in parallel
```

`Config`, `Local`, fixtures, toolsets, retry attempts, and case hooks still belong to a concrete case attempt. Runner
resources are shared through the runner and should be safe for the way you use them.

### Hooks And Parallel Suite Tests

Parallel suite tests do not introduce a second hook model. Hooks still belong to `Runner` and `Case`.

`BeforeAll` and `AfterAll` are runner-level hooks. They run once per `Runner`, guarded by the runner lifecycle, even when
multiple suite tests run in parallel. Runner resource cleanup runs first, then `AfterAll` is executed from Go's
`testing.T.Cleanup`, after the top-level suite test finishes and all parallel subtests have completed.

`BeforeTest` and `AfterTest` are case-attempt hooks. Fixture cleanup runs before `AfterTest`. If several suite tests or
cases run in parallel, these hooks may run concurrently. Keep shared state in those hooks immutable, runner-scoped and
concurrency-safe, or protected explicitly.

When a registered suite test uses `WithSuiteTestRunner`, that runner has its own lifecycle:

```go
suite.Test(
	"user can log in",
	(*UsersSuite).UserCanLogin,
	axiom.WithSuiteTestRunner(LoginRunner),
)
```

If `LoginRunner` is a separate runner, its `BeforeAll` and `AfterAll` are separate from `UsersRunner`. If the test should
inherit the suite runner behavior, compose the runner explicitly with `UsersRunner.Join(...)`.

Rules:

* test names passed to `suite.Test` must be non-empty
* test names must be unique within a suite
* test actions must be non-nil
* register all tests before calling `suite.Run`
* call `suite.Run` once per bound suite

---

## Complete Example

The following example demonstrates a complete suite use case with a shared runner, runner-scoped resource, fixture,
metadata, context, hooks, runtime wrappers, plugin, case-level overrides, and multiple registered suite tests.

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
	// BeforeAll/AfterAll wrap the whole suite because NewSuite provides
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
	suite := axiom.NewSuite(t, new(UsersSuite), axiom.WithSuiteConfigRunner(UsersRunner))
	suite.Test("user can log in", (*UsersSuite).UserCanLogin)
	suite.Test("admin can block user", (*UsersSuite).AdminCanBlockUser)
	suite.Run()
}

// -----------------------------------------------------------------------------
// Suite methods
// -----------------------------------------------------------------------------

func (s *UsersSuite) UserCanLogin() {
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

func (s *UsersSuite) AdminCanBlockUser() {
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

func (s *UsersSuite) helper() {}
```

In this example, `UsersRunner` remains the central execution configuration. The suite only defines a boundary around a
group of related registered tests.

Axiom runs the methods explicitly registered with `Test`. Inside each suite method, `s.RunCase(...)` executes a
normal Axiom `Case` through the suite runner.

The resulting test structure is:

```text
TestUsersSuite
  admin can block user
    admin can block user
  user can log in
    user can log in
```

The execution model is:

```text
NewSuite
  Runner ApplyStart
    bound suite test
      s.RunCase
        Case execution
    bound suite test
      s.RunCase
        Case execution
  Runner ApplyFinish
```

`Suite` is not a second runner and not a separate framework layer.

It is an optional grouping boundary around normal Axiom cases:

```text
Runner = shared execution configuration
Suite  = boundary for related registered tests
Case   = individual test definition
Config = runtime state for one case attempt
```
