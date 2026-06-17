package axiom_test

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runBoundSuite[T axiom.TestingSuite](
	t *testing.T,
	suite T,
	bind func(*axiom.BoundSuite[T]),
	options ...axiom.SuiteConfigOption,
) {
	boundSuite := axiom.NewSuite(t, suite, options...)
	if bind != nil {
		bind(boundSuite)
	}
	boundSuite.Run()
}

func runBoundSuiteFactory[T axiom.TestingSuite](
	t *testing.T,
	factory func() T,
	bind func(*axiom.BoundSuite[T]),
	options ...axiom.SuiteConfigOption,
) {
	boundSuite := axiom.NewSuiteFactory(t, factory, options...)
	if bind != nil {
		bind(boundSuite)
	}
	boundSuite.Run()
}

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
		runBoundSuite(t, &lifecycleSuite{order: &order}, func(s *axiom.BoundSuite[*lifecycleSuite]) {
			s.Test("TestAlpha", (*lifecycleSuite).TestAlpha)
			s.Test("TestBeta", (*lifecycleSuite).TestBeta)
		}, axiom.WithSuiteConfigRunner(runner))
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
		runBoundSuite(t, &hookCountingSuite{}, func(s *axiom.BoundSuite[*hookCountingSuite]) {
			s.Test("TestOne", (*hookCountingSuite).TestOne)
			s.Test("TestTwo", (*hookCountingSuite).TestTwo)
		}, axiom.WithSuiteConfigRunner(runner))
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
		runBoundSuite(t, &runnerUseCaseSuite{seen: &seen}, func(s *axiom.BoundSuite[*runnerUseCaseSuite]) {
			s.Test("TestRunnerConfigurationIsApplied", (*runnerUseCaseSuite).TestRunnerConfigurationIsApplied)
		}, axiom.WithSuiteConfigRunner(runner))
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
		runBoundSuite(t, &resourceSuite{seen: &seen}, func(s *axiom.BoundSuite[*resourceSuite]) {
			s.Test("TestFirst", (*resourceSuite).TestFirst)
			s.Test("TestSecond", (*resourceSuite).TestSecond)
		}, axiom.WithSuiteConfigRunner(runner))
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
		runBoundSuite(t, &pointerEmbeddedSuite{Suite: new(axiom.Suite), called: &called}, func(s *axiom.BoundSuite[*pointerEmbeddedSuite]) {
			s.Test("TestPointerEmbeddedSuite", (*pointerEmbeddedSuite).TestPointerEmbeddedSuite)
		})
	})

	assert.True(t, called)
}

type nestedSuiteLayer struct {
	axiom.Suite
}

type nestedBaseSuite struct {
	nestedSuiteLayer
}

type nestedEmbeddedSuite struct {
	nestedBaseSuite
	called *bool
}

func (s *nestedEmbeddedSuite) TestNestedEmbeddedSuite() {
	require.NotNil(s.SubT, s.Suite)

	s.RunCase(axiom.NewCase(axiom.WithCaseName("nested embedded")), func(cfg *axiom.Config) {
		*s.called = true
		assert.Same(cfg.SubT, s.SubT, cfg.RootT)
	})
}

func TestSuite_AllowsNestedEmbeddedSuite(t *testing.T) {
	called := false

	t.Run("suite", func(t *testing.T) {
		runBoundSuite(t, &nestedEmbeddedSuite{called: &called}, func(s *axiom.BoundSuite[*nestedEmbeddedSuite]) {
			s.Test("TestNestedEmbeddedSuite", (*nestedEmbeddedSuite).TestNestedEmbeddedSuite)
		})
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
		runBoundSuite(t, &defaultRunnerSuite{seenRunner: &seenRunner}, func(s *axiom.BoundSuite[*defaultRunnerSuite]) {
			s.Test("TestDefaultRunner", (*defaultRunnerSuite).TestDefaultRunner)
		})
	})

	assert.NotNil(t, seenRunner)
}

func TestSuite_UsesDefaultRunnerWhenOptionSetsNilRunner(t *testing.T) {
	var seenRunner *axiom.Runner

	t.Run("suite", func(t *testing.T) {
		runBoundSuite(t, &defaultRunnerSuite{seenRunner: &seenRunner}, func(s *axiom.BoundSuite[*defaultRunnerSuite]) {
			s.Test("TestDefaultRunner", (*defaultRunnerSuite).TestDefaultRunner)
		}, axiom.WithSuiteConfigRunner(nil))
	})

	assert.NotNil(t, seenRunner)
}

func TestSuite_TestOptionUsesSuiteRunnerWhenRunnerIsNil(t *testing.T) {
	var seenRunner *axiom.Runner
	runner := axiom.NewRunner()

	t.Run("suite", func(t *testing.T) {
		runBoundSuite(t, &defaultRunnerSuite{seenRunner: &seenRunner}, func(s *axiom.BoundSuite[*defaultRunnerSuite]) {
			s.Test("TestDefaultRunner", (*defaultRunnerSuite).TestDefaultRunner, axiom.WithSuiteTestRunner(nil))
		}, axiom.WithSuiteConfigRunner(runner))
	})

	assert.Same(t, runner, seenRunner)
}

func TestSuite_NewSuitePanicsWhenSuiteConfigParallelIsUsed(t *testing.T) {
	assert.PanicsWithValue(t, "suite: parallel suite tests require NewSuiteFactory", func() {
		axiom.NewSuite(t, &emptySuite{}, axiom.WithSuiteConfigParallel())
	})
}

func TestSuite_TestPanicsWhenSuiteTestParallelIsUsedWithoutFactory(t *testing.T) {
	boundSuite := axiom.NewSuite(t, &emptySuite{})

	assert.PanicsWithValue(t, "suite: parallel suite tests require NewSuiteFactory", func() {
		boundSuite.Test("parallel", func(s *emptySuite) {}, axiom.WithSuiteTestParallel())
	})
}

func TestSuite_NewSuiteFactoryPanicsWhenTestingTIsNil(t *testing.T) {
	assert.PanicsWithValue(t, "suite: nil *testing.T", func() {
		axiom.NewSuiteFactory[*emptySuite](nil, func() *emptySuite { return &emptySuite{} })
	})
}

func TestSuite_NewSuiteFactoryPanicsWhenFactoryIsNil(t *testing.T) {
	assert.PanicsWithValue(t, "suite: nil suite factory", func() {
		axiom.NewSuiteFactory[*emptySuite](t, nil)
	})
}

func TestSuite_BuildSuitePanicsWhenBoundSuiteIsNil(t *testing.T) {
	var boundSuite *axiom.BoundSuite[*emptySuite]

	assert.PanicsWithValue(t, "suite: nil BoundSuite", func() {
		_ = boundSuite.BuildSuite()
	})
}

func TestSuite_BuildSuiteReturnsSharedSuiteInstance(t *testing.T) {
	runner := axiom.NewRunner()
	suite := &emptySuite{}
	boundSuite := axiom.NewSuite(t, suite, axiom.WithSuiteConfigRunner(runner))

	built := boundSuite.BuildSuite()

	assert.Same(t, suite, built)
	assert.Same(t, t, built.RootT)
	assert.Nil(t, built.SubT)
	assert.Same(t, runner, built.Runner)
}

func TestSuiteFactory_BuildSuiteCreatesConfiguredSuiteInstance(t *testing.T) {
	runner := axiom.NewRunner()
	var created int
	boundSuite := axiom.NewSuiteFactory(t, func() *emptySuite {
		created++
		return &emptySuite{}
	}, axiom.WithSuiteConfigRunner(runner))

	first := boundSuite.BuildSuite()
	second := boundSuite.BuildSuite()

	assert.Equal(t, 2, created)
	assert.NotSame(t, first, second)
	assert.Same(t, t, first.RootT)
	assert.Nil(t, first.SubT)
	assert.Same(t, runner, first.Runner)
	assert.Same(t, t, second.RootT)
	assert.Nil(t, second.SubT)
	assert.Same(t, runner, second.Runner)
}

func TestSuiteFactory_BuildSuitePanicsWhenFactoryReturnsNil(t *testing.T) {
	boundSuite := axiom.NewSuiteFactory(t, func() *emptySuite {
		return nil
	})

	assert.PanicsWithValue(t, "suite: suite must be a non-nil pointer implementing axiom.TestingSuite", func() {
		_ = boundSuite.BuildSuite()
	})
}

func TestSuiteFactory_BuildSuitePanicsWhenFactoryReturnsNonStructPointer(t *testing.T) {
	boundSuite := axiom.NewSuiteFactory(t, func() *scalarTestingSuite {
		suite := scalarTestingSuite(1)
		return &suite
	})

	assert.PanicsWithValue(t, "suite: suite must be a pointer to a struct implementing axiom.TestingSuite", func() {
		_ = boundSuite.BuildSuite()
	})
}

type factoryInstanceSuite struct {
	axiom.Suite
	id   int64
	seen *[]string
}

func (s *factoryInstanceSuite) TestFirst() {
	*s.seen = append(*s.seen, fmt.Sprintf("first:%d", s.id))
}

func (s *factoryInstanceSuite) TestSecond() {
	*s.seen = append(*s.seen, fmt.Sprintf("second:%d", s.id))
}

func TestSuiteFactory_UsesFreshSuiteInstanceForEachRegisteredTest(t *testing.T) {
	var nextID atomic.Int64
	var seen []string
	runner := axiom.NewRunner()

	t.Run("suite", func(t *testing.T) {
		runBoundSuiteFactory(t, func() *factoryInstanceSuite {
			return &factoryInstanceSuite{
				id:   nextID.Add(1),
				seen: &seen,
			}
		}, func(s *axiom.BoundSuite[*factoryInstanceSuite]) {
			s.Test("TestFirst", (*factoryInstanceSuite).TestFirst)
			s.Test("TestSecond", (*factoryInstanceSuite).TestSecond)
		}, axiom.WithSuiteConfigRunner(runner))
	})

	assert.Equal(t, []string{"first:1", "second:2"}, seen)
}

type parallelFactorySuite struct {
	axiom.Suite
	id      int64
	ready   chan<- int64
	release <-chan struct{}
	records chan<- int64
}

func (s *parallelFactorySuite) TestParallel() {
	s.ready <- s.id
	<-s.release

	s.RunCase(axiom.NewCase(axiom.WithCaseName(fmt.Sprintf("case-%d", s.id))), func(cfg *axiom.Config) {
		assert.NotNil(s.T(), s.SubT)
		assert.Same(s.T(), s.SubT, cfg.RootT)
		assert.Same(s.T(), s.Runner, cfg.Runner)

		s.records <- s.id
	})
}

func TestSuiteFactory_RunsParallelSuiteTestsWithSeparateInstances(t *testing.T) {
	var nextID atomic.Int64
	ready := make(chan int64, 2)
	release := make(chan struct{})
	records := make(chan int64, 2)
	overlapped := make(chan bool, 1)
	runner := axiom.NewRunner()

	go func() {
		<-ready
		select {
		case <-ready:
			overlapped <- true
		case <-time.After(500 * time.Millisecond):
			overlapped <- false
		}
		close(release)
	}()

	t.Run("suite", func(t *testing.T) {
		runBoundSuiteFactory(t, func() *parallelFactorySuite {
			return &parallelFactorySuite{
				id:      nextID.Add(1),
				ready:   ready,
				release: release,
				records: records,
			}
		}, func(s *axiom.BoundSuite[*parallelFactorySuite]) {
			s.Test("first", (*parallelFactorySuite).TestParallel)
			s.Test("second", (*parallelFactorySuite).TestParallel)
		}, axiom.WithSuiteConfigRunner(runner), axiom.WithSuiteConfigParallel())
	})

	assert.True(t, <-overlapped)
	assert.ElementsMatch(t, []int64{1, 2}, []int64{<-records, <-records})
}

type suiteTestRunnerSuite struct {
	axiom.Suite
	seen          *[]string
	defaultRunner *axiom.Runner
	customRunner  *axiom.Runner
}

func (s *suiteTestRunnerSuite) TestDefaultRunner() {
	s.RunCase(axiom.NewCase(axiom.WithCaseName("default runner case")), func(cfg *axiom.Config) {
		client := axiom.MustResource[string](cfg.Runner, "client")

		*s.seen = append(*s.seen, "case:default:"+cfg.Meta.Feature+":"+client)
		assert.Same(cfg.SubT, s.defaultRunner, cfg.Runner)
	})
}

func (s *suiteTestRunnerSuite) TestCustomRunner() {
	s.RunCase(axiom.NewCase(axiom.WithCaseName("custom runner case")), func(cfg *axiom.Config) {
		client := axiom.MustResource[string](cfg.Runner, "client")

		*s.seen = append(*s.seen, "case:custom:"+cfg.Meta.Feature+":"+client)
		assert.Same(cfg.SubT, s.customRunner, cfg.Runner)
	})
}

func TestSuite_TestOptionRunnerOverridesSuiteRunner(t *testing.T) {
	var seen []string
	var defaultCreated int
	var defaultCleaned int
	var customCreated int
	var customCleaned int

	defaultRunner := axiom.NewRunner(
		axiom.WithRunnerMeta(axiom.WithMetaFeature("default")),
		axiom.WithRunnerHooks(
			axiom.WithBeforeAll(func(r *axiom.Runner) { seen = append(seen, "default:before-all") }),
			axiom.WithAfterAll(func(r *axiom.Runner) { seen = append(seen, "default:after-all") }),
			axiom.WithBeforeTest(func(cfg *axiom.Config) { seen = append(seen, "default:before-test:"+cfg.Meta.Feature) }),
			axiom.WithAfterTest(func(cfg *axiom.Config) { seen = append(seen, "default:after-test:"+cfg.Meta.Feature) }),
		),
		axiom.WithRunnerResource("client", func(r *axiom.Runner) (any, func(), error) {
			defaultCreated++
			return "default-client", func() { defaultCleaned++ }, nil
		}),
	)
	customRunner := axiom.NewRunner(
		axiom.WithRunnerMeta(axiom.WithMetaFeature("custom")),
		axiom.WithRunnerHooks(
			axiom.WithBeforeAll(func(r *axiom.Runner) { seen = append(seen, "custom:before-all") }),
			axiom.WithAfterAll(func(r *axiom.Runner) { seen = append(seen, "custom:after-all") }),
			axiom.WithBeforeTest(func(cfg *axiom.Config) { seen = append(seen, "custom:before-test:"+cfg.Meta.Feature) }),
			axiom.WithAfterTest(func(cfg *axiom.Config) { seen = append(seen, "custom:after-test:"+cfg.Meta.Feature) }),
		),
		axiom.WithRunnerResource("client", func(r *axiom.Runner) (any, func(), error) {
			customCreated++
			return "custom-client", func() { customCleaned++ }, nil
		}),
	)
	suite := &suiteTestRunnerSuite{
		seen:          &seen,
		defaultRunner: defaultRunner,
		customRunner:  customRunner,
	}

	t.Run("suite", func(t *testing.T) {
		runBoundSuite(t, suite, func(s *axiom.BoundSuite[*suiteTestRunnerSuite]) {
			s.Test("TestDefaultRunner", (*suiteTestRunnerSuite).TestDefaultRunner)
			s.Test("TestCustomRunner", (*suiteTestRunnerSuite).TestCustomRunner, axiom.WithSuiteTestRunner(customRunner))
		}, axiom.WithSuiteConfigRunner(defaultRunner))

		assert.Same(t, defaultRunner, suite.Runner)
		assert.Nil(t, suite.SubT)
		assert.Equal(t, 0, defaultCleaned)
		assert.Equal(t, 0, customCleaned)
	})

	assert.Equal(t, 1, defaultCreated)
	assert.Equal(t, 1, defaultCleaned)
	assert.Equal(t, 1, customCreated)
	assert.Equal(t, 1, customCleaned)
	assert.Equal(t, []string{
		"default:before-all",
		"default:before-test:default",
		"case:default:default:default-client",
		"default:after-test:default",
		"custom:before-all",
		"custom:before-test:custom",
		"case:custom:custom:custom-client",
		"custom:after-test:custom",
		"custom:after-all",
		"default:after-all",
	}, seen)
}

type suiteTestRunnerRestoreSuite struct {
	axiom.Suite
	runnerDuringMethod **axiom.Runner
}

func (s *suiteTestRunnerRestoreSuite) TestRecoveringPanic() {
	defer func() {
		_ = recover()
		*s.runnerDuringMethod = s.Runner
	}()

	panic("boom")
}

func TestSuite_RestoresSuiteRunnerAfterSuiteTestMethod(t *testing.T) {
	var runnerDuringMethod *axiom.Runner
	defaultRunner := axiom.NewRunner()
	customRunner := axiom.NewRunner()
	suite := &suiteTestRunnerRestoreSuite{
		runnerDuringMethod: &runnerDuringMethod,
	}

	t.Run("suite", func(t *testing.T) {
		runBoundSuite(t, suite, func(s *axiom.BoundSuite[*suiteTestRunnerRestoreSuite]) {
			s.Test("TestRecoveringPanic", (*suiteTestRunnerRestoreSuite).TestRecoveringPanic, axiom.WithSuiteTestRunner(customRunner))
		}, axiom.WithSuiteConfigRunner(defaultRunner))
	})

	assert.Same(t, customRunner, runnerDuringMethod)
	assert.Same(t, defaultRunner, suite.Runner)
	assert.Nil(t, suite.SubT)
}

type repeatedSuiteTestRunnerSuite struct {
	axiom.Suite
	events *[]string
}

func (s *repeatedSuiteTestRunnerSuite) TestFirst() {
	*s.events = append(*s.events, "first")
}

func (s *repeatedSuiteTestRunnerSuite) TestSecond() {
	*s.events = append(*s.events, "second")
}

func TestSuite_ReusesSameSuiteTestRunnerLifecycle(t *testing.T) {
	var events []string
	runner := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeAll(func(r *axiom.Runner) {
				events = append(events, "before-all")
			}),
			axiom.WithAfterAll(func(r *axiom.Runner) {
				events = append(events, "after-all")
			}),
		),
	)

	t.Run("suite", func(t *testing.T) {
		runBoundSuite(t, &repeatedSuiteTestRunnerSuite{events: &events}, func(s *axiom.BoundSuite[*repeatedSuiteTestRunnerSuite]) {
			s.Test("TestFirst", (*repeatedSuiteTestRunnerSuite).TestFirst, axiom.WithSuiteTestRunner(runner))
			s.Test("TestSecond", (*repeatedSuiteTestRunnerSuite).TestSecond, axiom.WithSuiteTestRunner(runner))
		})

		assert.Equal(t, []string{"before-all", "first", "second"}, events)
	})

	assert.Equal(t, []string{"before-all", "first", "second", "after-all"}, events)
}

func TestSuite_NewSuiteBindsRootTAndRunner(t *testing.T) {
	runner := axiom.NewRunner()
	suite := &emptySuite{}

	axiom.NewSuite(t, suite, axiom.WithSuiteConfigRunner(runner))

	assert.Same(t, t, suite.RootT)
	assert.Nil(t, suite.SubT)
	assert.Same(t, runner, suite.Runner)
}

type rootAndSubTSuite struct {
	axiom.Suite
	rootName *string
	subName  *string
	suiteT   *string
	caseRoot *string
	caseSub  *string
	caseT    *string
}

func (s *rootAndSubTSuite) TestTBinding() {
	*s.rootName = s.RootT.Name()
	*s.subName = s.SubT.Name()
	*s.suiteT = s.T().Name()

	s.RunCase(axiom.NewCase(axiom.WithCaseName("case t binding")), func(cfg *axiom.Config) {
		*s.caseRoot = cfg.RootT.Name()
		*s.caseSub = cfg.SubT.Name()
		*s.caseT = cfg.T().Name()
	})
}

func TestSuite_BindsRootAndSubTestingT(t *testing.T) {
	var rootName string
	var subName string
	var suiteT string
	var caseRoot string
	var caseSub string
	var caseT string

	t.Run("suite", func(t *testing.T) {
		runBoundSuite(t, &rootAndSubTSuite{
			rootName: &rootName,
			subName:  &subName,
			suiteT:   &suiteT,
			caseRoot: &caseRoot,
			caseSub:  &caseSub,
			caseT:    &caseT,
		}, func(s *axiom.BoundSuite[*rootAndSubTSuite]) {
			s.Test("TestTBinding", (*rootAndSubTSuite).TestTBinding)
		})
	})

	assert.True(t, strings.HasSuffix(rootName, "/suite"), rootName)
	assert.True(t, strings.HasSuffix(subName, "/suite/TestTBinding"), subName)
	assert.Equal(t, subName, suiteT)
	assert.Equal(t, subName, caseRoot)
	assert.True(t, strings.HasSuffix(caseSub, "/suite/TestTBinding/case_t_binding"), caseSub)
	assert.Equal(t, caseSub, caseT)
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
		runBoundSuite(t, &emptySuite{}, nil, axiom.WithSuiteConfigRunner(runner))
		assert.Equal(t, []string{"before"}, order)
	})

	assert.Equal(t, []string{"before", "after"}, order)
}

func TestSuite_TestPanicsWhenBoundSuiteIsNil(t *testing.T) {
	var boundSuite *axiom.BoundSuite[*emptySuite]

	assert.PanicsWithValue(t, "suite: nil BoundSuite", func() {
		boundSuite.Test("empty", func(s *emptySuite) {})
	})
}

func TestSuite_TestPanicsWhenNameIsEmpty(t *testing.T) {
	boundSuite := axiom.NewSuite(t, &emptySuite{})

	assert.PanicsWithValue(t, "suite: test name must not be empty", func() {
		boundSuite.Test("", func(s *emptySuite) {})
	})
}

func TestSuite_TestPanicsWhenActionIsNil(t *testing.T) {
	boundSuite := axiom.NewSuite(t, &emptySuite{})

	assert.PanicsWithValue(t, "suite: nil test action", func() {
		boundSuite.Test("empty", nil)
	})
}

func TestSuite_TestPanicsWhenNameIsDuplicated(t *testing.T) {
	boundSuite := axiom.NewSuite(t, &emptySuite{})
	boundSuite.Test("empty", func(s *emptySuite) {})

	assert.PanicsWithValue(t, "suite: duplicate test name: empty", func() {
		boundSuite.Test("empty", func(s *emptySuite) {})
	})
}

func TestSuite_TestPanicsAfterRun(t *testing.T) {
	boundSuite := axiom.NewSuite(t, &emptySuite{})
	boundSuite.Run()

	assert.PanicsWithValue(t, "suite: cannot register test after Run", func() {
		boundSuite.Test("empty", func(s *emptySuite) {})
	})
}

func TestSuite_RunPanicsWhenBoundSuiteIsNil(t *testing.T) {
	var boundSuite *axiom.BoundSuite[*emptySuite]

	assert.PanicsWithValue(t, "suite: nil BoundSuite", func() {
		boundSuite.Run()
	})
}

func TestSuite_RunPanicsWhenSuiteAlreadyRan(t *testing.T) {
	boundSuite := axiom.NewSuite(t, &emptySuite{})
	boundSuite.Run()

	assert.PanicsWithValue(t, "suite: suite already ran", func() {
		boundSuite.Run()
	})
}

func TestSuite_NewSuitePanicsWhenTestingTIsNil(t *testing.T) {
	assert.PanicsWithValue(t, "suite: nil *testing.T", func() {
		axiom.NewSuite(nil, &emptySuite{})
	})
}

func TestSuite_NewSuitePanicsWhenSuitePointerIsNil(t *testing.T) {
	var nilSuite *emptySuite

	assert.PanicsWithValue(t, "suite: suite must be a non-nil pointer implementing axiom.TestingSuite", func() {
		axiom.NewSuite(t, nilSuite)
	})
}

func TestSuite_NewSuitePanicsWhenSuiteInterfaceIsNil(t *testing.T) {
	assert.PanicsWithValue(t, "suite: suite must be a non-nil pointer implementing axiom.TestingSuite", func() {
		axiom.NewSuite[axiom.TestingSuite](t, nil)
	})
}

type valueTestingSuite struct{}

func (s valueTestingSuite) SetRootT(_ *testing.T) {}

func (s valueTestingSuite) SetSubT(_ *testing.T) {}

func (s valueTestingSuite) SetRunner(_ *axiom.Runner) {}

func (s valueTestingSuite) RunCase(_ axiom.Case, _ axiom.TestAction) {}

func TestSuite_NewSuitePanicsWhenSuiteIsNotPointer(t *testing.T) {
	assert.PanicsWithValue(t, "suite: suite must be a non-nil pointer implementing axiom.TestingSuite", func() {
		axiom.NewSuite(t, valueTestingSuite{})
	})
}

type scalarTestingSuite int

func (s *scalarTestingSuite) SetRootT(_ *testing.T) {}

func (s *scalarTestingSuite) SetSubT(_ *testing.T) {}

func (s *scalarTestingSuite) SetRunner(_ *axiom.Runner) {}

func (s *scalarTestingSuite) RunCase(_ axiom.Case, _ axiom.TestAction) {}

func TestSuite_NewSuitePanicsWhenSuitePointerDoesNotPointToStruct(t *testing.T) {
	var suite scalarTestingSuite

	assert.PanicsWithValue(t, "suite: suite must be a pointer to a struct implementing axiom.TestingSuite", func() {
		axiom.NewSuite(t, &suite)
	})
}

func TestSuite_SettersPanicWhenSuiteIsNil(t *testing.T) {
	var suite *axiom.Suite

	assert.PanicsWithValue(t, "suite: nil Suite", func() {
		suite.SetRootT(t)
	})
	assert.PanicsWithValue(t, "suite: nil Suite", func() {
		suite.SetSubT(t)
	})
	assert.PanicsWithValue(t, "suite: nil Suite", func() {
		suite.SetRunner(axiom.NewRunner())
	})
}

func TestSuite_T(t *testing.T) {
	subT := &testing.T{}

	suite := &axiom.Suite{
		RootT: t,
		SubT:  subT,
	}

	assert.Same(t, subT, suite.T())

	suite.SubT = nil
	assert.Same(t, t, suite.T())

	suite.RootT = nil
	assert.Nil(t, suite.T())
}

func TestSuite_SetRootT(t *testing.T) {
	suite := &axiom.Suite{}

	suite.SetRootT(t)

	assert.Same(t, t, suite.RootT)
}

func TestSuite_SetSubT(t *testing.T) {
	suite := &axiom.Suite{}

	suite.SetSubT(t)
	assert.Same(t, t, suite.SubT)

	suite.SetSubT(nil)
	assert.Nil(t, suite.SubT)
}

func TestSuite_SetRunner(t *testing.T) {
	runner := axiom.NewRunner()
	suite := &axiom.Suite{}

	suite.SetRunner(runner)
	assert.Same(t, runner, suite.Runner)

	suite.SetRunner(nil)
	assert.Nil(t, suite.Runner)
}

func TestSuite_RunCaseUsesConfiguredRunnerAndSubT(t *testing.T) {
	runner := axiom.NewRunner()
	suite := &axiom.Suite{}
	suite.SetSubT(t)
	suite.SetRunner(runner)

	called := false
	suite.RunCase(axiom.NewCase(axiom.WithCaseName("direct suite run case")), func(cfg *axiom.Config) {
		called = true
		assert.Same(cfg.SubT, t, cfg.RootT)
		assert.Same(cfg.SubT, runner, cfg.Runner)
	})

	assert.True(t, called)
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
		runBoundSuite(t, &runCaseWithoutSubTSuite{}, func(s *axiom.BoundSuite[*runCaseWithoutSubTSuite]) {
			s.Test("TestRunCaseWithoutSubT", (*runCaseWithoutSubTSuite).TestRunCaseWithoutSubT)
		})
	})
}

func TestSuite_TestReceivesOriginalSuiteInstance(t *testing.T) {
	suite := &emptySuite{}
	var seen *emptySuite

	t.Run("suite", func(t *testing.T) {
		runBoundSuite(t, suite, func(s *axiom.BoundSuite[*emptySuite]) {
			s.Test("identity", func(suite *emptySuite) {
				seen = suite
			})
		})
	})

	assert.Same(t, suite, seen)
	assert.Nil(t, suite.SubT)
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
		runBoundSuite(t, &multipleCasesSuite{order: &order}, func(s *axiom.BoundSuite[*multipleCasesSuite]) {
			s.Test("TestSeveralCases", (*multipleCasesSuite).TestSeveralCases)
		}, axiom.WithSuiteConfigRunner(runner))
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
		runBoundSuite(t, &fixtureIsolationSuite{values: &values}, func(s *axiom.BoundSuite[*fixtureIsolationSuite]) {
			s.Test("TestFirst", (*fixtureIsolationSuite).TestFirst)
			s.Test("TestSecond", (*fixtureIsolationSuite).TestSecond)
		}, axiom.WithSuiteConfigRunner(runner))
	})

	assert.Equal(t, 2, created)
	assert.Equal(t, 2, cleaned)
	assert.Equal(t, []string{"fixture-1", "fixture-2"}, values)
}

func TestSuite_RunCasePanicsWhenSubTIsMissing(t *testing.T) {
	s := &axiom.Suite{}
	s.SetRunner(axiom.NewRunner())

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
	suite := &subTResetSuite{names: &names}

	t.Run("suite", func(t *testing.T) {
		runBoundSuite(t, suite, func(s *axiom.BoundSuite[*subTResetSuite]) {
			s.Test("TestFirst", (*subTResetSuite).TestFirst)
			s.Test("TestSecond", (*subTResetSuite).TestSecond)
		})
	})

	require.Len(t, names, 2)
	assert.True(t, strings.HasSuffix(names[0], "/suite/TestFirst"), names[0])
	assert.True(t, strings.HasSuffix(names[1], "/suite/TestSecond"), names[1])
	assert.Nil(t, suite.SubT)
}
