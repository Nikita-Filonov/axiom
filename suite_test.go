package axiom_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lifecycleSuite struct {
	axiom.Suite
	order *[]string
}

func (s *lifecycleSuite) TestAlpha() {
	c := axiom.NewCase(axiom.WithCaseName("alpha case"))

	s.RunCase(c, func(cfg *axiom.Config) {
		*s.order = append(*s.order, "test:alpha")
		assert.Equal(cfg.SubT, "alpha case", cfg.Case.Name)
		assert.Same(cfg.SubT, s.SubT, cfg.RootT)
	})
}

func (s *lifecycleSuite) TestBeta() {
	c := axiom.NewCase(axiom.WithCaseName("beta case"))

	s.RunCase(c, func(cfg *axiom.Config) {
		*s.order = append(*s.order, "test:beta")
		assert.Equal(cfg.SubT, "beta case", cfg.Case.Name)
		assert.Same(cfg.SubT, s.SubT, cfg.RootT)
	})
}

func (s *lifecycleSuite) Helper() {}

func (s *lifecycleSuite) TestWithArgs(_ int) {}

func (s *lifecycleSuite) TestWithReturn() bool { return false }

func TestSuite_BeforeAllAfterAllWrapAllSuiteTests(t *testing.T) {
	var order []string

	runner := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeAll(func(r *axiom.Runner) {
				order = append(order, "before-all")
			}),
			axiom.WithAfterAll(func(r *axiom.Runner) {
				order = append(order, "after-all")
			}),
		),
	)

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &lifecycleSuite{order: &order}, axiom.WithSuiteRunner(runner))
	})

	assert.Equal(t, []string{"before-all", "test:alpha", "test:beta", "after-all"}, order)
}

type hookCountingSuite struct {
	axiom.Suite
}

func (s *hookCountingSuite) TestOne() {
	s.RunCase(axiom.NewCase(axiom.WithCaseName("one")), func(cfg *axiom.Config) {
		cfg.Step("step", func() {})
	})
}

func (s *hookCountingSuite) TestTwo() {
	s.RunCase(axiom.NewCase(axiom.WithCaseName("two")), func(cfg *axiom.Config) {
		cfg.Step("step", func() {})
	})
}

func TestSuite_TestAndStepHooksRunForEachCase(t *testing.T) {
	var beforeTestCount int
	var afterTestCount int
	var beforeStepCount int
	var afterStepCount int

	runner := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeTest(func(cfg *axiom.Config) { beforeTestCount++ }),
			axiom.WithAfterTest(func(cfg *axiom.Config) { afterTestCount++ }),
			axiom.WithBeforeStep(func(cfg *axiom.Config, name string) { beforeStepCount++ }),
			axiom.WithAfterStep(func(cfg *axiom.Config, name string) { afterStepCount++ }),
		),
	)

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &hookCountingSuite{}, axiom.WithSuiteRunner(runner))
	})

	assert.Equal(t, 2, beforeTestCount)
	assert.Equal(t, 2, afterTestCount)
	assert.Equal(t, 2, beforeStepCount)
	assert.Equal(t, 2, afterStepCount)
}

type runnerUseCaseSuite struct {
	axiom.Suite
	seen *[]string
}

func (s *runnerUseCaseSuite) TestRunnerConfigurationIsApplied() {
	c := axiom.NewCase(
		axiom.WithCaseName("runner configuration is applied"),
		axiom.WithCaseMeta(
			axiom.WithMetaStory("valid login"),
			axiom.WithMetaTag("case-tag"),
		),
		axiom.WithCaseContext(
			axiom.WithContextData("role", "admin"),
		),
		axiom.WithCaseFixture("token", func(cfg *axiom.Config) (any, func(), error) {
			user := axiom.GetFixture[string](cfg, "user")
			client := axiom.MustResource[string](cfg.Runner, "client")

			return fmt.Sprintf("token:%s:%s", user, client), nil, nil
		}),
	)

	s.RunCase(c, func(cfg *axiom.Config) {
		env := axiom.MustContextValue[string](&cfg.Context, "env")
		role := axiom.MustContextValue[string](&cfg.Context, "role")
		pluginValue := axiom.MustContextValue[string](&cfg.Context, "plugin")

		user := axiom.GetFixture[string](cfg, "user")
		token := axiom.GetFixture[string](cfg, "token")
		client := axiom.MustResource[string](cfg.Runner, "client")

		*s.seen = append(*s.seen,
			"body",
			"env:"+env,
			"role:"+role,
			"plugin:"+pluginValue,
			"meta:"+cfg.Meta.Feature+":"+cfg.Meta.Story,
			"user:"+user,
			"client:"+client,
			"token:"+token,
		)

		cfg.Step("validate", func() {
			*s.seen = append(*s.seen, "step:validate")
		})
	})
}

func TestSuite_UsesFullRunnerConfiguration(t *testing.T) {
	var seen []string
	var resourceCreated int
	var resourceCleaned int
	var fixtureCleaned int

	runner := axiom.NewRunner(
		axiom.WithRunnerMeta(
			axiom.WithMetaEpic("platform"),
			axiom.WithMetaFeature("users"),
			axiom.WithMetaTag("runner-tag"),
		),
		axiom.WithRunnerContext(
			axiom.WithContextData("env", "staging"),
		),
		axiom.WithRunnerResource("client", func(r *axiom.Runner) (any, func(), error) {
			resourceCreated++
			env := axiom.MustContextValue[string](&r.Context, "env")

			return "client:" + env, func() { resourceCleaned++ }, nil
		}),
		axiom.WithRunnerFixture("user", func(cfg *axiom.Config) (any, func(), error) {
			env := axiom.MustContextValue[string](&cfg.Context, "env")

			return "user:" + env, func() { fixtureCleaned++ }, nil
		}),
		axiom.WithRunnerHooks(
			axiom.WithBeforeAll(func(r *axiom.Runner) { seen = append(seen, "before-all") }),
			axiom.WithAfterAll(func(r *axiom.Runner) { seen = append(seen, "after-all") }),
			axiom.WithBeforeTest(func(cfg *axiom.Config) { seen = append(seen, "before-test") }),
			axiom.WithAfterTest(func(cfg *axiom.Config) { seen = append(seen, "after-test") }),
			axiom.WithBeforeStep(func(cfg *axiom.Config, name string) { seen = append(seen, "before-step:"+name) }),
			axiom.WithAfterStep(func(cfg *axiom.Config, name string) { seen = append(seen, "after-step:"+name) }),
		),
		axiom.WithRunnerRuntime(
			axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
				return func(cfg *axiom.Config) {
					seen = append(seen, "wrap-test-before")
					next(cfg)
					seen = append(seen, "wrap-test-after")
				}
			}),
			axiom.WithRuntimeStepWrap(func(name string, next axiom.StepAction) axiom.StepAction {
				return func() {
					seen = append(seen, "wrap-step-before:"+name)
					next()
					seen = append(seen, "wrap-step-after:"+name)
				}
			}),
		),
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			cfg.Context.SetData("plugin", "applied")
		}),
	)

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &runnerUseCaseSuite{seen: &seen}, axiom.WithSuiteRunner(runner))
		assert.Equal(t, 0, resourceCleaned)
	})

	assert.Equal(t, 1, resourceCreated)
	assert.Equal(t, 1, resourceCleaned)
	assert.Equal(t, 1, fixtureCleaned)
	assert.Equal(t, []string{
		"before-all",
		"before-test",
		"wrap-test-before",
		"body",
		"env:staging",
		"role:admin",
		"plugin:applied",
		"meta:users:valid login",
		"user:user:staging",
		"client:client:staging",
		"token:token:user:staging:client:staging",
		"before-step:validate",
		"wrap-step-before:validate",
		"step:validate",
		"wrap-step-after:validate",
		"after-step:validate",
		"wrap-test-after",
		"after-test",
		"after-all",
	}, seen)
}

type resourceSuite struct {
	axiom.Suite
	seen *[]string
}

func (s *resourceSuite) TestFirst() {
	s.RunCase(axiom.NewCase(axiom.WithCaseName("first")), func(cfg *axiom.Config) {
		*s.seen = append(*s.seen, axiom.MustResource[string](cfg.Runner, "shared"))
	})
}

func (s *resourceSuite) TestSecond() {
	s.RunCase(axiom.NewCase(axiom.WithCaseName("second")), func(cfg *axiom.Config) {
		*s.seen = append(*s.seen, axiom.MustResource[string](cfg.Runner, "shared"))
	})
}

func TestSuite_ResourcesAreSharedAndCleanedUpAfterSuite(t *testing.T) {
	var created int
	var cleaned int
	var seen []string

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("shared", func(r *axiom.Runner) (any, func(), error) {
			created++
			return "resource", func() { cleaned++ }, nil
		}),
	)

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &resourceSuite{seen: &seen}, axiom.WithSuiteRunner(runner))
		assert.Equal(t, 0, cleaned)
	})

	assert.Equal(t, 1, created)
	assert.Equal(t, []string{"resource", "resource"}, seen)
	assert.Equal(t, 1, cleaned)
}

type pointerEmbeddedSuite struct {
	*axiom.Suite
	called *bool
}

func (s *pointerEmbeddedSuite) TestPointerEmbeddedSuite() {
	require.NotNil(s.SubT, s.Suite)

	s.RunCase(axiom.NewCase(axiom.WithCaseName("pointer embedded")), func(cfg *axiom.Config) {
		*s.called = true
		assert.Same(cfg.SubT, s.SubT, cfg.RootT)
	})
}

func TestSuite_AllowsPointerEmbeddedSuite(t *testing.T) {
	called := false

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &pointerEmbeddedSuite{called: &called})
	})

	assert.True(t, called)
}

type defaultRunnerSuite struct {
	axiom.Suite
	seenRunner **axiom.Runner
}

func (s *defaultRunnerSuite) TestDefaultRunner() {
	s.RunCase(axiom.NewCase(axiom.WithCaseName("default runner")), func(cfg *axiom.Config) {
		*s.seenRunner = cfg.Runner
		assert.NotNil(cfg.SubT, cfg.Runner)
	})
}

func TestSuite_UsesDefaultRunnerWhenOptionIsMissing(t *testing.T) {
	var seenRunner *axiom.Runner

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &defaultRunnerSuite{seenRunner: &seenRunner})
	})

	assert.NotNil(t, seenRunner)
}

func TestSuite_UsesDefaultRunnerWhenOptionSetsNilRunner(t *testing.T) {
	var seenRunner *axiom.Runner

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &defaultRunnerSuite{seenRunner: &seenRunner}, axiom.WithSuiteRunner(nil))
	})

	assert.NotNil(t, seenRunner)
}

type rootAndSubTSuite struct {
	axiom.Suite
	rootName *string
	subName  *string
	caseRoot *string
	caseSub  *string
}

func (s *rootAndSubTSuite) TestTBinding() {
	*s.rootName = s.RootT.Name()
	*s.subName = s.SubT.Name()

	s.RunCase(axiom.NewCase(axiom.WithCaseName("case t binding")), func(cfg *axiom.Config) {
		*s.caseRoot = cfg.RootT.Name()
		*s.caseSub = cfg.SubT.Name()
	})
}

func TestSuite_BindsRootAndSubTestingT(t *testing.T) {
	var rootName string
	var subName string
	var caseRoot string
	var caseSub string

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &rootAndSubTSuite{
			rootName: &rootName,
			subName:  &subName,
			caseRoot: &caseRoot,
			caseSub:  &caseSub,
		})
	})

	assert.True(t, strings.HasSuffix(rootName, "/suite"), rootName)
	assert.True(t, strings.HasSuffix(subName, "/suite/TestTBinding"), subName)
	assert.Equal(t, subName, caseRoot)
	assert.True(t, strings.HasSuffix(caseSub, "/suite/TestTBinding/case_t_binding"), caseSub)
}

type emptySuite struct {
	axiom.Suite
}

func TestSuite_RunsBeforeAllAfterAllEvenWithoutTestMethods(t *testing.T) {
	var order []string

	runner := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeAll(func(r *axiom.Runner) { order = append(order, "before") }),
			axiom.WithAfterAll(func(r *axiom.Runner) { order = append(order, "after") }),
		),
	)

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &emptySuite{}, axiom.WithSuiteRunner(runner))
		assert.Equal(t, []string{"before"}, order)
	})

	assert.Equal(t, []string{"before", "after"}, order)
}

type namedSuiteField struct {
	Base axiom.Suite
}

func TestSuite_RequiresEmbeddedSuite(t *testing.T) {
	assert.PanicsWithValue(t, "suite: suite must embed axiom.Suite", func() {
		axiom.RunSuite(t, &namedSuiteField{})
	})
}

type namedPointerSuiteField struct {
	Base *axiom.Suite
}

func TestSuite_RequiresEmbeddedPointerSuite(t *testing.T) {
	assert.PanicsWithValue(t, "suite: suite must embed axiom.Suite", func() {
		axiom.RunSuite(t, &namedPointerSuiteField{})
	})
}

func TestSuite_RunPanicsWhenTestingTIsNil(t *testing.T) {
	assert.PanicsWithValue(t, "suite: nil *testing.T", func() {
		axiom.RunSuite(nil, &emptySuite{})
	})
}

func TestSuite_RunPanicsWhenSuiteIsNil(t *testing.T) {
	assert.PanicsWithValue(t, "suite: suite must be a non-nil pointer to a struct", func() {
		axiom.RunSuite(t, nil)
	})
}

func TestSuite_RunPanicsWhenSuiteIsNotPointer(t *testing.T) {
	assert.PanicsWithValue(t, "suite: suite must be a non-nil pointer to a struct", func() {
		axiom.RunSuite(t, emptySuite{})
	})
}

func TestSuite_RunPanicsWhenSuitePointerIsNil(t *testing.T) {
	var nilSuite *emptySuite

	assert.PanicsWithValue(t, "suite: suite must be a non-nil pointer to a struct", func() {
		axiom.RunSuite(t, nilSuite)
	})
}

func TestSuite_RunPanicsWhenSuitePointerDoesNotPointToStruct(t *testing.T) {
	v := 1

	assert.PanicsWithValue(t, "suite: suite must be a pointer to a struct", func() {
		axiom.RunSuite(t, &v)
	})
}

func TestSuite_RunPanicsWhenStructDoesNotEmbedSuite(t *testing.T) {
	assert.PanicsWithValue(t, "suite: suite must embed axiom.Suite", func() {
		axiom.RunSuite(t, &struct{}{})
	})
}

func TestSuite_RunCasePanicsForInvalidRuntimeState(t *testing.T) {
	var nilSuite *axiom.Suite
	assert.PanicsWithValue(t, "suite: nil Suite", func() {
		nilSuite.RunCase(axiom.NewCase(), func(cfg *axiom.Config) {})
	})

	s := &axiom.Suite{SubT: t}
	assert.PanicsWithValue(t, "suite: runner is not configured", func() {
		s.RunCase(axiom.NewCase(), func(cfg *axiom.Config) {})
	})
}

type runCaseWithoutSubTSuite struct {
	axiom.Suite
}

func (s *runCaseWithoutSubTSuite) TestRunCaseWithoutSubT() {
	s.SubT = nil

	assert.PanicsWithValue(s.RootT, "suite: nil *testing.T", func() {
		s.RunCase(axiom.NewCase(), func(cfg *axiom.Config) {})
	})
}

func TestSuite_RunCasePanicsWhenCalledOutsideSuiteTestMethod(t *testing.T) {
	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &runCaseWithoutSubTSuite{})
	})
}

type discoverySuite struct {
	axiom.Suite
	called *[]string
}

func (s *discoverySuite) TestValid() {
	*s.called = append(*s.called, "valid")
}

func (s *discoverySuite) Helper() {
	*s.called = append(*s.called, "helper")
}

func (s *discoverySuite) TestWithReturn() bool {
	*s.called = append(*s.called, "with-return")
	return true
}

func (s *discoverySuite) TestWithArgs(_ int) {
	*s.called = append(*s.called, "with-args")
}

func TestSuite_DiscoveryRunsOnlyExportedZeroArgTestMethods(t *testing.T) {
	var called []string

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &discoverySuite{called: &called})
	})

	assert.Equal(t, []string{"valid"}, called)
}

type multipleCasesSuite struct {
	axiom.Suite
	order *[]string
}

func (s *multipleCasesSuite) TestSeveralCases() {
	first := axiom.NewCase(axiom.WithCaseName("first case"))
	second := axiom.NewCase(axiom.WithCaseName("second case"))

	s.RunCase(first, func(cfg *axiom.Config) {
		*s.order = append(*s.order, "case:first")
		assert.Same(cfg.SubT, s.SubT, cfg.RootT)
	})

	s.RunCase(second, func(cfg *axiom.Config) {
		*s.order = append(*s.order, "case:second")
		assert.Same(cfg.SubT, s.SubT, cfg.RootT)
	})
}

func TestSuite_AllowsMultipleCasesInsideSingleSuiteMethod(t *testing.T) {
	var order []string

	runner := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeTest(func(cfg *axiom.Config) {
				order = append(order, "before:"+cfg.Case.Name)
			}),
			axiom.WithAfterTest(func(cfg *axiom.Config) {
				order = append(order, "after:"+cfg.Case.Name)
			}),
		),
	)

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &multipleCasesSuite{order: &order}, axiom.WithSuiteRunner(runner))
	})

	assert.Equal(t, []string{
		"before:first case",
		"case:first",
		"after:first case",
		"before:second case",
		"case:second",
		"after:second case",
	}, order)
}

type fixtureIsolationSuite struct {
	axiom.Suite
	values *[]string
}

func (s *fixtureIsolationSuite) TestFirst() {
	s.RunCase(axiom.NewCase(axiom.WithCaseName("first")), func(cfg *axiom.Config) {
		*s.values = append(*s.values, axiom.GetFixture[string](cfg, "value"))
	})
}

func (s *fixtureIsolationSuite) TestSecond() {
	s.RunCase(axiom.NewCase(axiom.WithCaseName("second")), func(cfg *axiom.Config) {
		*s.values = append(*s.values, axiom.GetFixture[string](cfg, "value"))
	})
}

func TestSuite_FixturesAreCreatedPerCase(t *testing.T) {
	var created int
	var cleaned int
	var values []string

	runner := axiom.NewRunner(
		axiom.WithRunnerFixture("value", func(cfg *axiom.Config) (any, func(), error) {
			created++
			value := fmt.Sprintf("fixture-%d", created)

			return value, func() { cleaned++ }, nil
		}),
	)

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &fixtureIsolationSuite{values: &values}, axiom.WithSuiteRunner(runner))
	})

	assert.Equal(t, 2, created)
	assert.Equal(t, 2, cleaned)
	assert.Equal(t, []string{"fixture-1", "fixture-2"}, values)
}

func TestSuite_RunCasePanicsWhenSubTIsMissing(t *testing.T) {
	s := &axiom.Suite{}
	axiom.WithSuiteRunner(axiom.NewRunner())(s)

	assert.PanicsWithValue(t, "suite: nil *testing.T", func() {
		s.RunCase(axiom.NewCase(), func(cfg *axiom.Config) {})
	})
}

type subTResetSuite struct {
	axiom.Suite
	names *[]string
}

func (s *subTResetSuite) TestFirst() {
	*s.names = append(*s.names, s.SubT.Name())
}

func (s *subTResetSuite) TestSecond() {
	*s.names = append(*s.names, s.SubT.Name())
}

func TestSuite_RebindsSubTForEachSuiteMethod(t *testing.T) {
	var names []string

	t.Run("suite", func(t *testing.T) {
		axiom.RunSuite(t, &subTResetSuite{names: &names})
	})

	require.Len(t, names, 2)
	assert.True(t, strings.HasSuffix(names[0], "/suite/TestFirst"), names[0])
	assert.True(t, strings.HasSuffix(names[1], "/suite/TestSecond"), names[1])
}
