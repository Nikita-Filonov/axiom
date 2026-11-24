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
	"github.com/Nikita-Filonov/axiom/internal/execution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionAdapterReportsOnlyFinalAttempts(t *testing.T) {
	submitted := axiom.Attempt{
		RunID:  "original",
		Number: 2,
		Decision: axiom.Decision{
			Kind:   axiom.DecisionSkip,
			Reason: "disabled",
		},
	}
	var received []axiom.Attempt
	cfg := &axiom.Config{
		Attempt: submitted,
		Runtime: axiom.NewRuntime(axiom.WithRuntimeAttemptSink(func(attempt axiom.Attempt) {
			received = append(received, attempt)
		})),
	}
	spec := cfg.Spec(func(*axiom.Config) {})
	cfg.Attempt = axiom.Attempt{RunID: "changed", Number: 9}

	for _, kind := range []execution.EventType{
		execution.AttemptStarted, execution.AttemptExecutionFinished,
		execution.AttemptPanicRecovered,
	} {
		spec.Observe(execution.Event{Type: kind, Message: "recovered panic"})
	}
	assert.Empty(t, received, "lifecycle facts are not terminal attempts")

	started := time.Now()
	ended := started.Add(time.Second)
	input := execution.Result{
		Kind: execution.ResultCompleted,
		Outcome: &execution.Outcome{
			Status:    execution.AttemptFailed,
			TestName:  "Test/attempt",
			StartedAt: started,
			CleanupAt: ended,
		},
	}
	spec.Complete(input)
	require.Len(t, received, 1)
	got := received[0]
	assert.Equal(t, submitted.RunID, got.RunID)
	assert.Equal(t, submitted.Number, got.Number)
	assert.Equal(t, submitted.Decision, got.Decision)
	assert.Equal(t, &axiom.AttemptOutcome{
		Status:    axiom.AttemptFailed,
		TestName:  "Test/attempt",
		StartedAt: started,
		CleanupAt: ended,
	}, got.Outcome)
	assert.Equal(t, got, cfg.Attempt, "Config must expose the terminal attempt before sinks run")

	input.Outcome.Status = execution.AttemptPassed
	assert.Equal(t, axiom.AttemptFailed, cfg.Attempt.Outcome.Status)
}

func TestExecutionAdapterPreservesSelectionDecision(t *testing.T) {
	for _, tc := range []struct {
		name     string
		result   execution.ResultKind
		decision axiom.DecisionKind
	}{
		{name: "excluded", result: execution.ResultExcluded, decision: axiom.DecisionExclude},
		{name: "not selected", result: execution.ResultNotSelected, decision: axiom.DecisionRun},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var received []axiom.Attempt
			cfg := &axiom.Config{
				Attempt: axiom.Attempt{Decision: axiom.Decision{Kind: tc.decision}},
				Runtime: axiom.NewRuntime(axiom.WithRuntimeAttemptSink(func(attempt axiom.Attempt) {
					received = append(received, attempt)
				})),
			}
			spec := cfg.Spec(func(*axiom.Config) {})
			spec.Complete(execution.Result{Kind: tc.result})
			require.Len(t, received, 1)
			assert.Equal(t, tc.decision, received[0].Decision.Kind)
			assert.Nil(t, received[0].Outcome)
		})
	}
}

func TestExecution_RetryKeepsInvocationIDAcrossAttempts(t *testing.T) {
	for _, tc := range []struct {
		name     string
		parallel string
	}{
		{name: "sequential", parallel: "0"},
		{name: "parallel", parallel: "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := runCaseExecutionHelper(
				t,
				"TestExecution_RetryIdentityHelperProcess",
				"AXIOM_EXECUTION_PARALLEL="+tc.parallel,
			)

			require.Error(t, err, "the first failed attempt remains visible to Go's testing package")
			assert.Contains(t, output, "execution retry identity verified")
		})
	}
}

func TestExecution_RetryIdentityHelperProcess(t *testing.T) {
	if os.Getenv(caseExecutionHelperEnv) != "1" {
		t.Skip("helper process")
	}

	var pluginConfigs []*axiom.Config
	var actionAttempts []axiom.Attempt
	runnerOptions := []axiom.RunnerOption{
		axiom.WithRunnerRetry(axiom.WithRetryTimes(3)),
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			pluginConfigs = append(pluginConfigs, cfg)
		}),
	}
	if os.Getenv("AXIOM_EXECUTION_PARALLEL") == "1" {
		runnerOptions = append(runnerOptions, axiom.WithRunnerParallel(axiom.WithParallelEnabled()))
	}

	t.Cleanup(func() {
		require.Len(t, pluginConfigs, 3, "planning config and two attempt configs")
		require.Len(t, actionAttempts, 2)
		runID := pluginConfigs[0].Attempt.RunID
		require.NotEmpty(t, runID)
		matches := true
		for number, cfg := range pluginConfigs {
			matches = assert.Equal(t, runID, cfg.Attempt.RunID) && matches
			matches = assert.Equal(t, number, cfg.Attempt.Number) && matches
			matches = assert.Equal(t, axiom.DecisionRun, cfg.Attempt.Decision.Kind) && matches
		}
		for i, attempt := range actionAttempts {
			prepared := pluginConfigs[i+1].Attempt
			matches = assert.Equal(t, prepared.RunID, attempt.RunID) && matches
			matches = assert.Equal(t, prepared.Number, attempt.Number) && matches
			matches = assert.Equal(t, prepared.Decision, attempt.Decision) && matches
			matches = assert.Nil(t, attempt.Outcome) && matches
		}
		if matches {
			t.Log("execution retry identity verified")
		}
	})

	axiom.NewRunner(runnerOptions...).RunCase(t, axiom.NewCase(), func(cfg *axiom.Config) {
		actionAttempts = append(actionAttempts, cfg.Attempt)
		if cfg.Attempt.Number == 1 {
			cfg.T().Error("fail the first attempt")
		}
	})
}

func TestExecution_PluginSkipOnFirstAttemptStopsRetries(t *testing.T) {
	for _, tc := range []struct {
		name     string
		parallel bool
	}{
		{name: "sequential"},
		{name: "parallel retry", parallel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			type observedEvent struct {
				attempt axiom.Attempt
				event   axiom.Event
			}
			var events []observedEvent
			var pluginInstalls, actions int
			runnerOptions := []axiom.RunnerOption{
				axiom.WithRunnerRetry(axiom.WithRetryTimes(3)),
				axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
					pluginInstalls++
					if cfg.Attempt.Number == 1 {
						cfg.Skip = axiom.NewSkip(axiom.SkipBecause("plugin decision"))
					}
					cfg.Runtime.EmitEventSink(func(event axiom.Event) {
						events = append(events, observedEvent{attempt: cfg.Attempt, event: event})
					})
				}),
			}
			if tc.parallel {
				runnerOptions = append(runnerOptions, axiom.WithRunnerParallel(axiom.WithParallelEnabled()))
			}

			t.Run("scope", func(t *testing.T) {
				axiom.NewRunner(runnerOptions...).RunCase(t, axiom.NewCase(), func(*axiom.Config) {
					actions++
				})
			})

			require.Len(t, events, 1, "a skipped attempt has no start or finish event")
			assert.Equal(t, axiom.EventTypeCaseSkip, events[0].event.Type)
			assert.Equal(t, "plugin decision", events[0].event.Message)
			assert.Equal(t, 1, events[0].attempt.Number)
			assert.Equal(t, 2, pluginInstalls, "the skipped attempt must stop retries")
			assert.Zero(t, actions)
		})
	}
}

func TestExecution_PolicySkipEmitsCaseSkip(t *testing.T) {
	type observed struct {
		attempt int
		event   axiom.Event
	}

	for _, tc := range []struct {
		name     string
		parallel bool
		attempt  int
	}{
		{name: "sequential", attempt: 1},
		{name: "parallel retry", parallel: true, attempt: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var events []observed
			runner := axiom.NewRunner(
				axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
				axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
					cfg.Runtime.EmitEventSink(func(e axiom.Event) {
						events = append(events, observed{attempt: cfg.Attempt.Number, event: e})
					})
				}),
			)
			if tc.parallel {
				runner.Parallel = axiom.NewParallel(axiom.WithParallelEnabled())
			}

			runner.RunCase(t, axiom.NewCase(axiom.WithCaseSkip(axiom.SkipBecause("reason"))), func(*axiom.Config) {
				t.Error("skipped action ran")
			})

			require.Len(t, events, 1)
			assert.Equal(t, axiom.EventTypeCaseSkip, events[0].event.Type)
			assert.Equal(t, "reason", events[0].event.Message)
			assert.Equal(t, tc.attempt, events[0].attempt)
		})
	}
}

func TestExecution_CleanupRegisteredOnCaseStartOrSkipRunsLast(t *testing.T) {
	var order []string
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			cfg.Runtime.EmitEventSink(func(e axiom.Event) {
				if e.Type == axiom.EventTypeCaseStart || e.Type == axiom.EventTypeCaseSkip {
					cfg.T().Cleanup(func() { order = append(order, "observer") })
				}
			})
		}),
		axiom.WithRunnerRuntime(axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(cfg *axiom.Config) {
				cfg.T().Cleanup(func() { order = append(order, "wrap") })
				next(cfg)
			}
		})),
		axiom.WithRunnerFixture("fixture", func(*axiom.Config) (any, func(), error) {
			return 1, func() { order = append(order, "fixture") }, nil
		}),
	)

	runner.RunCase(t, axiom.NewCase(), func(cfg *axiom.Config) {
		axiom.GetFixture[int](cfg, "fixture")
		cfg.T().Cleanup(func() { order = append(order, "test") })
	})
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseSkip(axiom.SkipBecause("reason"))), func(*axiom.Config) {})

	assert.Equal(t, []string{"fixture", "test", "wrap", "observer", "wrap", "observer"}, order)
}

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
	localKey := axiom.NewLocalKey[int]("retry-attempt")

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

	caseOptions := []axiom.CaseOption{
		axiom.WithCaseName("parallel retry case"),
		axiom.WithCaseContext(axiom.WithContextData("retry-attempt", "fresh")),
	}
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
		if value := axiom.MustContextValue[string](&cfg.Context, "retry-attempt"); value != "fresh" {
			cfg.T().Errorf("context leaked between attempts: %q", value)
		}
		if value, exists := axiom.GetLocal(cfg, localKey); exists {
			cfg.T().Errorf("local value leaked between attempts: %d", value)
		}
		cfg.Context.SetData("retry-attempt", "changed")
		axiom.SetLocal(cfg, localKey, attempts)

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
	assert.Contains(t, output, "parallel-retry skip attempts=0 plugins=2")
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
	var sourceNames []string
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			pluginInstalls++
			sourceNames = append(sourceNames, cfg.Case.Name)
			cfg.Case.Name = "plugin:" + cfg.Case.Name
			cfg.Case.Context.SetData("base-plugin", pluginInstalls)
			cfg.Context.SetData("plugin-install", pluginInstalls)
		}),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseName("case"),
		axiom.WithCaseContext(axiom.WithContextData("source", "original")),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		assert.Equal(t, "plugin:case", cfg.Case.Name)
		assert.Equal(t, 2, axiom.MustContextValue[int](&cfg.Context, "plugin-install"))
	})

	assert.Equal(t, []string{"case", "case"}, sourceNames)
	assert.Equal(t, "case", testCase.Name)
	assert.Equal(t, "original", axiom.MustContextValue[string](&testCase.Context, "source"))
	_, mutated := testCase.Context.Data["base-plugin"]
	assert.False(t, mutated)
	assert.Equal(t, 2, pluginInstalls, "planning and attempt configs install plugins independently")
}

func TestExecutionSelectionAndSkipInheritance(t *testing.T) {
	var outcomes []axiom.Attempt
	var raw []axiom.EventType
	var wrapped, before, after, bodies int
	runner := axiom.NewRunner(
		axiom.WithRunnerSkip(axiom.SkipBecause("runner disabled")),
		axiom.WithRunnerHooks(
			axiom.WithBeforeTest(func(*axiom.Config) { before++ }),
			axiom.WithAfterTest(func(*axiom.Config) { after++ }),
		),
		axiom.WithRunnerRuntime(
			axiom.WithRuntimeAttemptSink(func(e axiom.Attempt) {
				outcomes = append(outcomes, e)
			}),
			axiom.WithRuntimeEventSink(func(e axiom.Event) { raw = append(raw, e.Type) }),
			axiom.WithRuntimeTestWrap(func(next axiom.TestAction) axiom.TestAction {
				return func(c *axiom.Config) {
					wrapped++
					next(c)
				}
			}),
		),
	)
	for _, c := range []axiom.Case{
		axiom.NewCase(axiom.WithCaseName("inherited")),
		axiom.NewCase(axiom.WithCaseName("explicit"), axiom.WithCaseSkip(axiom.SkipBecause("feature disabled"))),
		axiom.NewCase(axiom.WithCaseName("enabled"), axiom.WithCaseSkip(axiom.WithSkipDisabled())),
		axiom.NewCase(axiom.WithCaseName("filtered"), axiom.WithCaseSkip(axiom.WithSkipDisabled()),
			axiom.WithCasePlugins(func(c *axiom.Config) { c.Attempt.Exclude("different country") })),
	} {
		runner.RunCase(t, c, func(*axiom.Config) { bodies++ })
	}
	assert.Equal(t, 3, wrapped)
	assert.Equal(t, 1, before)
	assert.Equal(t, 1, after)
	assert.Equal(t, 1, bodies)
	require.Len(t, outcomes, 4)
	assert.Equal(t, "runner disabled", outcomes[0].Decision.Reason)
	assert.Equal(t, "feature disabled", outcomes[1].Decision.Reason)
	assert.Equal(t, axiom.AttemptSkipped, outcomes[0].Outcome.Status)
	assert.Equal(t, axiom.AttemptSkipped, outcomes[1].Outcome.Status)
	assert.Equal(t, axiom.AttemptPassed, outcomes[2].Outcome.Status)
	assert.Equal(t, axiom.DecisionExclude, outcomes[3].Decision.Kind)
	assert.Nil(t, outcomes[3].Outcome)
	assert.Equal(t, []axiom.EventType{
		axiom.EventTypeRunnerBeforeAllStart, axiom.EventTypeRunnerBeforeAllFinish,
		axiom.EventTypeCaseSkip, axiom.EventTypeCaseSkip,
		axiom.EventTypeCaseStart, axiom.EventTypeCaseFinish, axiom.EventTypeCaseExcluded,
	}, raw)
}

func TestCaseExecution_WrapsCalled(t *testing.T) {
	var order []string

	rt := axiom.NewRuntime(
		axiom.WithRuntimeTestWrap(
			func(next axiom.TestAction) axiom.TestAction {
				return func(c *axiom.Config) {
					order = append(order, "wrap1")
					next(c)
				}
			},
		),
		axiom.WithRuntimeTestWrap(
			func(next axiom.TestAction) axiom.TestAction {
				return func(c *axiom.Config) {
					order = append(order, "wrap2")
					next(c)
				}
			},
		),
	)

	testCase := axiom.Case{
		Runtime: rt,
	}

	axiom.NewRunner().RunCase(t, testCase, func(_ *axiom.Config) {
		order = append(order, "body")
	})

	assert.Equal(t,
		[]string{"wrap1", "wrap2", "body"},
		order,
	)
}

func TestCaseExecution_HooksOrder(t *testing.T) {
	var calls []string

	testCase := axiom.Case{
		Name: "T",
		Hooks: axiom.Hooks{
			BeforeTest: []axiom.TestHook{
				func(_ *axiom.Config) { calls = append(calls, "before") },
			},
			AfterTest: []axiom.TestHook{
				func(_ *axiom.Config) { calls = append(calls, "after") },
			},
		},
	}

	axiom.NewRunner().RunCase(t, testCase, func(_ *axiom.Config) {
		calls = append(calls, "body")
	})

	assert.Equal(t,
		[]string{"before", "body", "after"},
		calls,
	)
}

func TestCaseExecution_EmitsStartAndFinishFacts(t *testing.T) {
	var events []axiom.Event
	testCase := axiom.Case{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
	}

	axiom.NewRunner().RunCase(t, testCase, func(_ *axiom.Config) {})

	require.Len(t, events, 2)
	assert.Equal(t, axiom.EventTypeCaseStart, events[0].Type)
	assert.Equal(t, axiom.EventTypeCaseFinish, events[1].Type)
}
