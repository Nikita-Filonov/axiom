package axiom_test

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const caseExecutionHelperEnv = "AXIOM_CASE_EXECUTION_HELPER"

func runCaseExecutionHelper(t *testing.T, testName string, env ...string) (string, error) {
	t.Helper()

	cmd := exec.Command(
		os.Args[0],
		"-test.run=^"+testName+"$",
		"-test.parallel=4",
		"-test.v",
	)
	cmd.Env = append(os.Environ(), caseExecutionHelperEnv+"=1")
	cmd.Env = append(cmd.Env, env...)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestCaseExecution_WaitBeforeAttempt_AppliesRetryDelay(t *testing.T) {
	const delay = 5 * time.Millisecond
	runner := axiom.NewRunner(axiom.WithRunnerRetry(axiom.WithRetryDelay(delay)))
	execution := axiom.NewCaseExecution(runner, t, axiom.NewCase(), nil)

	started := time.Now()
	execution.WaitBeforeAttempt(2)

	assert.GreaterOrEqual(t, time.Since(started), delay)
}

func TestCaseExecution_ParallelRetry_RunsFreshLifecycleForEveryFailureKind(t *testing.T) {
	for _, failure := range []string{"error", "panic", "fail-now"} {
		t.Run(failure, func(t *testing.T) {
			output, err := runCaseExecutionHelper(
				t,
				"TestCaseExecution_ParallelRetry_LifecycleHelperProcess",
				"AXIOM_RETRY_FAILURE="+failure,
			)

			require.Error(t, err, "the first failed attempt must keep the helper process failed")
			assert.Contains(t, output, "=== PAUSE TestCaseExecution_ParallelRetry_LifecycleHelperProcess/parallel_retry_case")
			assert.Contains(t, output,
				"parallel-retry lifecycle failure="+failure+
					" parallel=true attempts=2 plugins=3 wraps=2 before=2 after=2 fixtures=2 cleanups=2 resources=1 resource-cleanups=1 delay=true",
			)
		})
	}
}

func TestCaseExecution_ParallelRetry_StopsAfterConfiguredAttempts(t *testing.T) {
	output, err := runCaseExecutionHelper(
		t,
		"TestCaseExecution_ParallelRetry_LifecycleHelperProcess",
		"AXIOM_RETRY_FAILURE=error",
		"AXIOM_RETRY_FAIL_ALL=1",
	)

	require.Error(t, err, "every attempt intentionally fails")
	assert.Contains(t, output,
		"parallel-retry lifecycle failure=error parallel=true attempts=3 plugins=4 wraps=3 before=3 after=3 fixtures=3 cleanups=3 resources=1 resource-cleanups=1 delay=true",
	)
}

func TestCaseExecution_ParallelRetry_LifecycleHelperProcess(t *testing.T) {
	if os.Getenv(caseExecutionHelperEnv) != "1" {
		t.Skip("helper process")
	}

	const retryDelay = 20 * time.Millisecond

	failure := os.Getenv("AXIOM_RETRY_FAILURE")
	failAll := os.Getenv("AXIOM_RETRY_FAIL_ALL") == "1"
	disableParallel := os.Getenv("AXIOM_DISABLE_PARALLEL") == "1"

	var attempts int
	var pluginInstalls int
	var wrappedActions int
	var beforeTests int
	var afterTests int
	var fixtureSetups int
	var fixtureCleanups int
	var resourceSetups int
	var resourceCleanups int
	var firstAttemptStarted time.Time
	var delayObserved bool
	var effectiveParallel bool

	runner := axiom.NewRunner(
		axiom.WithRunnerRetry(
			axiom.WithRetryTimes(3),
			axiom.WithRetryDelay(retryDelay),
		),
		axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			pluginInstalls++
			cfg.Runtime.EmitTestWrap(func(next axiom.TestAction) axiom.TestAction {
				return func(cfg *axiom.Config) {
					wrappedActions++
					next(cfg)
				}
			})
		}),
		axiom.WithRunnerHooks(
			axiom.WithBeforeTest(func(cfg *axiom.Config) { beforeTests++ }),
			axiom.WithAfterTest(func(cfg *axiom.Config) { afterTests++ }),
		),
		axiom.WithRunnerFixture("value", func(cfg *axiom.Config) (any, func(), error) {
			fixtureSetups++
			return fixtureSetups, func() { fixtureCleanups++ }, nil
		}),
		axiom.WithRunnerResource("shared", func(runner *axiom.Runner) (any, func(), error) {
			resourceSetups++
			return "resource", func() { resourceCleanups++ }, nil
		}),
	)

	caseOptions := []axiom.CaseOption{axiom.WithCaseName("parallel retry case")}
	if disableParallel {
		caseOptions = append(caseOptions,
			axiom.WithCaseParallel(axiom.WithParallelDisabled()),
		)
	}

	t.Cleanup(func() {
		t.Logf(
			"parallel-retry lifecycle failure=%s parallel=%t attempts=%d plugins=%d wraps=%d before=%d after=%d fixtures=%d cleanups=%d resources=%d resource-cleanups=%d delay=%t",
			failure,
			effectiveParallel,
			attempts,
			pluginInstalls,
			wrappedActions,
			beforeTests,
			afterTests,
			fixtureSetups,
			fixtureCleanups,
			resourceSetups,
			resourceCleanups,
			delayObserved,
		)
	})

	runner.RunCase(t, axiom.NewCase(caseOptions...), func(cfg *axiom.Config) {
		attempts++
		effectiveParallel = cfg.Parallel.Enabled

		fixtureValue := axiom.GetFixture[int](cfg, "value")
		if fixtureValue != attempts {
			cfg.T().Errorf("fixture value leaked between attempts: got %d, attempt %d", fixtureValue, attempts)
		}
		if resource := axiom.MustResource[string](cfg.Runner, "shared"); resource != "resource" {
			cfg.T().Errorf("unexpected shared resource: %q", resource)
		}

		if attempts == 1 {
			firstAttemptStarted = time.Now()
		} else {
			expectedDelay := retryDelay * time.Duration(attempts-1)
			delayObserved = time.Since(firstAttemptStarted) >= expectedDelay
		}

		if failAll {
			cfg.T().Errorf("fail attempt %d", attempts)
			return
		}

		switch attempts {
		case 1:
			switch failure {
			case "error":
				cfg.T().Error("fail the first attempt")
			case "panic":
				panic("fail the first attempt")
			case "fail-now":
				cfg.T().FailNow()
			default:
				cfg.T().Fatalf("unknown failure mode %q", failure)
			}
		default:
			if attempts > 2 {
				cfg.T().Errorf("retry did not stop after the passing attempt: %d", attempts)
			}
		}
	})
}

func TestCaseExecution_CaseDisablesRunnerParallelForRetries(t *testing.T) {
	output, err := runCaseExecutionHelper(
		t,
		"TestCaseExecution_ParallelRetry_LifecycleHelperProcess",
		"AXIOM_RETRY_FAILURE=error",
		"AXIOM_DISABLE_PARALLEL=1",
	)

	require.Error(t, err, "the first failed attempt must keep the helper process failed")
	assert.NotContains(t, output, "=== PAUSE")
	assert.Contains(t, output,
		"parallel-retry lifecycle failure=error parallel=false attempts=2 plugins=3 wraps=2 before=2 after=2 fixtures=2 cleanups=2 resources=1 resource-cleanups=1 delay=true",
	)
}

func TestCaseExecution_ParallelRetry_SkipStopsBeforeAttempts(t *testing.T) {
	output, err := runCaseExecutionHelper(
		t,
		"TestCaseExecution_ParallelRetry_SkipHelperProcess",
	)

	require.NoError(t, err)
	assert.NotContains(t, output, "=== PAUSE")
	assert.Contains(t, output, "parallel-retry skip attempts=0 plugins=1")
}

func TestCaseExecution_ParallelRetry_SkipHelperProcess(t *testing.T) {
	if os.Getenv(caseExecutionHelperEnv) != "1" {
		t.Skip("helper process")
	}

	var pluginInstalls atomic.Int64
	var attempts atomic.Int64

	runner := axiom.NewRunner(
		axiom.WithRunnerRetry(axiom.WithRetryTimes(3)),
		axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			pluginInstalls.Add(1)
		}),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseName("skipped"),
		axiom.WithCaseSkip(axiom.SkipBecause("not applicable")),
	)

	t.Cleanup(func() {
		t.Logf(
			"parallel-retry skip attempts=%d plugins=%d",
			attempts.Load(),
			pluginInstalls.Load(),
		)
	})

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		attempts.Add(1)
	})
}

func TestCaseExecution_SkipIsScopedToSelectedCase(t *testing.T) {
	runner := axiom.NewRunner()
	skippedActionRan := false
	nextActionRan := false

	runner.RunCase(
		t,
		axiom.NewCase(
			axiom.WithCaseName("skipped"),
			axiom.WithCaseSkip(axiom.SkipBecause("not applicable")),
		),
		func(cfg *axiom.Config) {
			skippedActionRan = true
		},
	)
	runner.RunCase(
		t,
		axiom.NewCase(axiom.WithCaseName("next")),
		func(cfg *axiom.Config) {
			nextActionRan = true
		},
	)

	assert.False(t, skippedActionRan)
	assert.True(t, nextActionRan)
}

func TestCaseExecution_ParallelRetry_NamePluginDoesNotAccumulate(t *testing.T) {
	output, err := runCaseExecutionHelper(
		t,
		"TestCaseExecution_ParallelRetry_NamePluginHelperProcess",
	)

	require.Error(t, err, "the first failed attempt must keep the helper process failed")
	assert.Contains(t, output, "=== PAUSE TestCaseExecution_ParallelRetry_NamePluginHelperProcess/[Feature]_name")
	assert.Contains(t, output,
		"name-plugin source=name attempts=2 applications=3 names=[[Feature] name [Feature] name]",
	)
	assert.NotContains(t, output, "[Feature] [Feature] name")
}

func TestCaseExecution_ParallelRetry_NamePluginHelperProcess(t *testing.T) {
	if os.Getenv(caseExecutionHelperEnv) != "1" {
		t.Skip("helper process")
	}

	runner := axiom.NewRunner(
		axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
		axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
	)

	var attempts int
	var applications int
	testCase := axiom.NewCase(
		axiom.WithCaseName("name"),
		axiom.WithCaseMeta(axiom.WithMetaFeature("Feature")),
		axiom.WithCasePlugins(func(cfg *axiom.Config) {
			applications++
			cfg.Case.Name = fmt.Sprintf("[%s] %s", cfg.Meta.Feature, cfg.Case.Name)
		}),
	)

	var names []string
	t.Cleanup(func() {
		t.Logf(
			"name-plugin source=%s attempts=%d applications=%d names=%v",
			testCase.Name,
			attempts,
			applications,
			names,
		)
	})

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		attempts++
		names = append(names, cfg.Case.Name)

		if cfg.Case.Name != "[Feature] name" {
			cfg.T().Errorf("name plugin accumulated: %q", cfg.Case.Name)
		}
		if attempts == 1 {
			cfg.T().Error("fail the first attempt")
		}
	})
}

func TestCaseExecution_CaseRetryOneUsesRegularParallelExecution(t *testing.T) {
	var attempts atomic.Int64
	var pluginInstalls atomic.Int64
	var actionTestName string

	runner := axiom.NewRunner(
		axiom.WithRunnerRetry(axiom.WithRetryTimes(3)),
		axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			pluginInstalls.Add(1)
		}),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseName("name"),
		axiom.WithCaseRetry(axiom.WithRetryTimes(1)),
	)

	t.Run("scope", func(t *testing.T) {
		runner.RunCase(t, testCase, func(cfg *axiom.Config) {
			attempts.Add(1)
			actionTestName = cfg.T().Name()
		})
	})

	assert.Equal(t, int64(1), attempts.Load())
	assert.Equal(t, int64(2), pluginInstalls.Load(), "one base and one attempt config")
	assert.Equal(t, t.Name()+"/scope/name", actionTestName, "Retry.Times=1 must not add a retry wrapper")
}

func TestCaseExecution_CaseParallelEnablesRetryWrapper(t *testing.T) {
	var attempts atomic.Int64
	var actionTestName string
	var effectiveParallel bool

	runner := axiom.NewRunner(
		axiom.WithRunnerRetry(axiom.WithRetryTimes(3)),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseName("name"),
		axiom.WithCaseParallel(axiom.WithParallelEnabled()),
	)

	t.Run("scope", func(t *testing.T) {
		runner.RunCase(t, testCase, func(cfg *axiom.Config) {
			attempts.Add(1)
			actionTestName = cfg.T().Name()
			effectiveParallel = cfg.Parallel.Enabled
		})
	})

	assert.Equal(t, int64(1), attempts.Load(), "a passing first attempt must stop retries")
	assert.True(t, effectiveParallel)
	assert.Equal(t, t.Name()+"/scope/name/name", actionTestName, "parallel retries must run inside the Case wrapper")
}

func TestCaseExecution_RunAttemptsAppliesPoliciesInOrder(t *testing.T) {
	var order []string

	execution := axiom.NewCaseExecution(
		axiom.NewRunner(),
		t,
		axiom.NewCase(axiom.WithCaseName("name")),
		func(cfg *axiom.Config) {
			order = append(order, "action")
		},
	)

	execution.RunAttempts(
		t,
		func(cfg *axiom.Config) { order = append(order, "first") },
		func(cfg *axiom.Config) { order = append(order, "second") },
	)

	assert.Equal(t, []string{"first", "second", "action"}, order)
}

type parallelRetrySuiteState struct {
	attemptsMu sync.Mutex
	attempts   map[string]int
	ready      chan string
	release    chan struct{}
	overlapped atomic.Bool

	actions          atomic.Int64
	pluginInstalls   atomic.Int64
	wrappedActions   atomic.Int64
	beforeTests      atomic.Int64
	afterTests       atomic.Int64
	fixtureSetups    atomic.Int64
	fixtureCleanups  atomic.Int64
	resourceSetups   atomic.Int64
	resourceCleanups atomic.Int64
}

func newParallelRetrySuiteState() *parallelRetrySuiteState {
	return &parallelRetrySuiteState{
		attempts: map[string]int{},
		ready:    make(chan string, 2),
		release:  make(chan struct{}),
	}
}

func (s *parallelRetrySuiteState) nextAttempt(name string) int {
	s.attemptsMu.Lock()
	defer s.attemptsMu.Unlock()

	s.attempts[name]++
	return s.attempts[name]
}

func (s *parallelRetrySuiteState) attemptCount(name string) int {
	s.attemptsMu.Lock()
	defer s.attemptsMu.Unlock()

	return s.attempts[name]
}

type parallelRetryIntegrationSuite struct {
	axiom.Suite
	state *parallelRetrySuiteState
}

func (s *parallelRetryIntegrationSuite) TestParallelRetries() {
	for _, name := range []string{"first", "second"} {
		testCase := axiom.NewCase(axiom.WithCaseName(name))
		s.RunCase(testCase, func(cfg *axiom.Config) {
			attempt := s.state.nextAttempt(cfg.Case.Name)
			s.state.actions.Add(1)
			_ = axiom.GetFixture[int64](cfg, "value")
			_ = axiom.MustResource[int64](cfg.Runner, "shared")

			if attempt == 1 {
				s.state.ready <- cfg.Case.Name
				<-s.state.release
				cfg.T().Error("fail the first attempt")
			}
		})
	}
}

func TestCaseExecution_SuiteParallelRetry_CasesOverlapAndRetryIndependently(t *testing.T) {
	output, err := runCaseExecutionHelper(
		t,
		"TestCaseExecution_SuiteParallelRetry_HelperProcess",
	)

	require.Error(t, err, "each Case intentionally fails its first attempt")
	assert.Contains(t, output, "=== PAUSE TestCaseExecution_SuiteParallelRetry_HelperProcess/parallel_retry_cases/first")
	assert.Contains(t, output, "=== PAUSE TestCaseExecution_SuiteParallelRetry_HelperProcess/parallel_retry_cases/second")
	assert.Contains(t, output,
		"suite-parallel-retry first=2 second=2 overlapped=true actions=4 plugins=6 wraps=4 before=4 after=4 fixtures=4 cleanups=4 resources=1 resource-cleanups=1",
	)
}

func TestCaseExecution_SuiteParallelRetry_HelperProcess(t *testing.T) {
	if os.Getenv(caseExecutionHelperEnv) != "1" {
		t.Skip("helper process")
	}

	state := newParallelRetrySuiteState()
	runner := axiom.NewRunner(
		axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
		axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			state.pluginInstalls.Add(1)
			cfg.Runtime.EmitTestWrap(func(next axiom.TestAction) axiom.TestAction {
				return func(cfg *axiom.Config) {
					state.wrappedActions.Add(1)
					next(cfg)
				}
			})
		}),
		axiom.WithRunnerHooks(
			axiom.WithBeforeTest(func(cfg *axiom.Config) { state.beforeTests.Add(1) }),
			axiom.WithAfterTest(func(cfg *axiom.Config) { state.afterTests.Add(1) }),
		),
		axiom.WithRunnerFixture("value", func(cfg *axiom.Config) (any, func(), error) {
			value := state.fixtureSetups.Add(1)
			return value, func() { state.fixtureCleanups.Add(1) }, nil
		}),
		axiom.WithRunnerResource("shared", func(runner *axiom.Runner) (any, func(), error) {
			value := state.resourceSetups.Add(1)
			return value, func() { state.resourceCleanups.Add(1) }, nil
		}),
	)

	go func() {
		timer := time.NewTimer(time.Second)
		defer timer.Stop()

		select {
		case first := <-state.ready:
			select {
			case second := <-state.ready:
				state.overlapped.Store(first != second)
			case <-timer.C:
			}
		case <-timer.C:
		}

		close(state.release)
	}()

	t.Cleanup(func() {
		t.Logf(
			"suite-parallel-retry first=%d second=%d overlapped=%t actions=%d plugins=%d wraps=%d before=%d after=%d fixtures=%d cleanups=%d resources=%d resource-cleanups=%d",
			state.attemptCount("first"),
			state.attemptCount("second"),
			state.overlapped.Load(),
			state.actions.Load(),
			state.pluginInstalls.Load(),
			state.wrappedActions.Load(),
			state.beforeTests.Load(),
			state.afterTests.Load(),
			state.fixtureSetups.Load(),
			state.fixtureCleanups.Load(),
			state.resourceSetups.Load(),
			state.resourceCleanups.Load(),
		)
	})

	testSuite := axiom.NewSuiteFactory(
		t,
		func() *parallelRetryIntegrationSuite {
			return &parallelRetryIntegrationSuite{state: state}
		},
		axiom.WithSuiteConfigRunner(runner),
	)
	testSuite.Test(
		"parallel retry cases",
		(*parallelRetryIntegrationSuite).TestParallelRetries,
	)
	testSuite.Run()
}

func TestCaseExecution_KeepsTemplateIsolatedFromBasePlugins(t *testing.T) {
	pluginInstalls := 0
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			pluginInstalls++
			cfg.Case.Name = "plugin:" + cfg.Case.Name
			cfg.Case.Context.SetData("base-plugin", pluginInstalls)
			cfg.Context.SetData("plugin-install", pluginInstalls)
		}),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseName("case"),
		axiom.WithCaseContext(axiom.WithContextData("source", "original")),
	)

	execution := axiom.NewCaseExecution(runner, t, testCase, func(cfg *axiom.Config) {})
	first := execution.BaseConfig().Execution.NextAttempt()
	firstAttempt := execution.NewConfig(first)
	secondAttempt := execution.NewConfig(first.NextAttempt())

	if execution.CaseTemplate().Name != "case" {
		t.Fatalf("case template was mutated: %q", execution.CaseTemplate().Name)
	}
	if _, exists := execution.CaseTemplate().Context.Data["base-plugin"]; exists {
		t.Fatal("base plugin mutation leaked into the case template")
	}
	if execution.BaseConfig().Case.Name != "plugin:case" {
		t.Fatalf("unexpected base case name: %q", execution.BaseConfig().Case.Name)
	}
	if firstAttempt.Case.Name != "plugin:case" {
		t.Fatalf("unexpected first attempt name: %q", firstAttempt.Case.Name)
	}
	if secondAttempt.Case.Name != "plugin:case" {
		t.Fatalf("unexpected second attempt name: %q", secondAttempt.Case.Name)
	}
	if testCase.Name != "case" {
		t.Fatalf("source case was mutated: %q", testCase.Name)
	}
	if pluginInstalls != 3 {
		t.Fatalf("expected one base and two attempt plugin installs, got %d", pluginInstalls)
	}
}

func TestCaseExecution_CreatesFreshAttemptConfigs(t *testing.T) {
	runner := axiom.NewRunner()
	localKey := axiom.NewLocalKey[string]("attempt")
	testCase := axiom.NewCase(
		axiom.WithCaseName("case"),
		axiom.WithCaseContext(axiom.WithContextData("value", "template")),
		axiom.WithCaseFixture("fixture", func(cfg *axiom.Config) (any, func(), error) {
			return "value", nil, nil
		}),
	)
	execution := axiom.NewCaseExecution(runner, t, testCase, func(cfg *axiom.Config) {})

	first := execution.BaseConfig().Execution.NextAttempt()
	firstAttempt := execution.NewConfig(first)
	firstAttempt.Context.SetData("value", "changed")
	firstAttempt.Fixtures.Cache()["fixture"] = "cached"
	axiom.SetLocal(firstAttempt, localKey, "first")

	secondAttempt := execution.NewConfig(first.NextAttempt())

	if firstAttempt == secondAttempt {
		t.Fatal("attempt configs must have different identities")
	}
	if got := axiom.MustContextValue[string](&secondAttempt.Context, "value"); got != "template" {
		t.Fatalf("attempt context leaked: %q", got)
	}
	if len(secondAttempt.Fixtures.Cache()) != 0 {
		t.Fatalf("attempt fixture cache leaked: %#v", secondAttempt.Fixtures.Cache())
	}
	if value, exists := axiom.GetLocal(secondAttempt, localKey); exists {
		t.Fatalf("attempt local value leaked: %q", value)
	}
}
