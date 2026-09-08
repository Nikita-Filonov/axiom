package testallure_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testallure"
	"github.com/allure-framework/allure-go/commons/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const lifecycleProbeEnv = "AXIOM_TESTALLURE_LIFECYCLE_PROBE"

func TestPlugin_RetryWritesOneResultPerAttempt(t *testing.T) {
	results := runLifecycleProbe(t, "retry", true)
	require.Len(t, results, 2)

	firstAttempt := resultWithStep(t, results, "attempt 1")
	secondAttempt := resultWithStep(t, results, "attempt 2")

	assert.Equal(t, model.StatusFailed, firstAttempt.Status)
	assert.Equal(t, model.StatusPassed, secondAttempt.Status)
	assert.Equal(t, "retry succeeds on the second attempt", firstAttempt.Name)
	assert.Equal(t, firstAttempt.Name, secondAttempt.Name)
	assert.NotEqual(t, firstAttempt.UUID, secondAttempt.UUID)
	assert.NotEqual(t, firstAttempt.FullName, secondAttempt.FullName)
	assert.Equal(t, "RETRY-1", firstAttempt.TestCaseID)
	assert.Equal(t, firstAttempt.TestCaseID, secondAttempt.TestCaseID)
	assert.Equal(t, firstAttempt.HistoryID, secondAttempt.HistoryID)
	assertResultStep(t, firstAttempt, "attempt 1", model.StatusFailed)
	assertResultStep(t, secondAttempt, "attempt 2", model.StatusPassed)
	assertResultAttachment(t, firstAttempt, "attempt-cleanup.txt")
	assertResultAttachment(t, secondAttempt, "attempt-cleanup.txt")
}

func TestPlugin_ParallelRetryKeepsAttemptResultsIsolated(t *testing.T) {
	results := runLifecycleProbe(t, "parallel-retry", true)
	require.Len(t, results, 4)

	byName := map[string][]model.TestResult{}
	for _, result := range results {
		byName[result.Name] = append(byName[result.Name], result)
	}

	for _, name := range []string{"parallel retry alpha", "parallel retry beta"} {
		caseResults := byName[name]
		require.Len(t, caseResults, 2, "results for %q", name)

		firstAttempt := resultWithStep(t, caseResults, name+" attempt 1")
		secondAttempt := resultWithStep(t, caseResults, name+" attempt 2")
		assert.Equal(t, model.StatusFailed, firstAttempt.Status)
		assert.Equal(t, model.StatusPassed, secondAttempt.Status)
		assert.Equal(t, firstAttempt.TestCaseID, secondAttempt.TestCaseID)
		assert.Equal(t, firstAttempt.HistoryID, secondAttempt.HistoryID)
		assertResultStep(t, firstAttempt, name+" attempt 1", model.StatusFailed)
		assertResultStep(t, secondAttempt, name+" attempt 2", model.StatusPassed)
		assert.Equal(t, name+".txt", firstAttempt.Steps[0].Attachments[0].Name)
		assert.Equal(t, name+".txt", secondAttempt.Steps[0].Attachments[0].Name)
	}
}

func TestPlugin_FailedCaseWritesFailedResult(t *testing.T) {
	results := runLifecycleProbe(t, "failed", true)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, model.StatusFailed, result.Status)
	assertResultStep(t, result, "compare inventory", model.StatusFailed)
	require.Len(t, result.Steps[0].Attachments, 1)
	assert.Equal(t, "actual-inventory.json", result.Steps[0].Attachments[0].Name)
	assertResultAttachment(t, result, "failed-case-cleanup.txt")
}

func TestPlugin_AssertionFailureRecordsMessageAndStack(t *testing.T) {
	results := runLifecycleProbe(t, "assert-fail", true)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, model.StatusFailed, result.Status)

	require.NotNil(t, result.StatusDetails)
	assert.Contains(t, result.StatusDetails.Message, "values must match")
	assert.Contains(t, result.StatusDetails.Message, "expected-value")
	assert.Contains(t, result.StatusDetails.Message, "actual-value")
	assert.Contains(t, result.StatusDetails.Message, "--- Stack trace ---")

	assertResultStep(t, result, "compare values", model.StatusFailed)
	require.NotNil(t, result.Steps[0].StatusDetails)
	assert.Contains(t, result.Steps[0].StatusDetails.Message, "values must match")

	require.Len(t, result.Steps[0].Attachments, 1)
	attachment := result.Steps[0].Attachments[0]
	assert.Equal(t, "Stacktrace", attachment.Name)
	assert.Equal(t, "text/plain", attachment.Type)
}

func TestPlugin_PanicInsideStepWritesBrokenStep(t *testing.T) {
	results := runLifecycleProbe(t, "step-panic", true)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, model.StatusFailed, result.Status)
	assertResultStep(t, result, "decode response", model.StatusBroken)
	require.NotNil(t, result.Steps[0].StatusDetails)
	assert.Contains(t, result.Steps[0].StatusDetails.Message, "malformed response")
	assert.NotEmpty(t, result.Steps[0].StatusDetails.Trace)
}

func TestPlugin_RuntimeSkipWritesSkippedResult(t *testing.T) {
	results := runLifecycleProbe(t, "runtime-skip", false)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, model.StatusSkipped, result.Status)
	assertResultStep(t, result, "check feature flag", model.StatusSkipped)
	require.Len(t, result.Steps[0].Attachments, 1)
	assert.Equal(t, "feature-flag.txt", result.Steps[0].Attachments[0].Name)
	assertResultAttachment(t, result, "skipped-case-cleanup.txt")
}

func TestPlugin_TestBodyPanicKeepsFixtureCleanupArtefact(t *testing.T) {
	results := runLifecycleProbe(t, "test-panic", true)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, model.StatusBroken, result.Status)
	assertResultAttachment(t, result, "body-panic-cleanup.txt")
}

func TestPlugin_BeforeTestPanicKeepsFixtureCleanupArtefact(t *testing.T) {
	results := runLifecycleProbe(t, "before-test-panic", true)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, model.StatusBroken, result.Status)
	assertResultAttachment(t, result, "before-test-panic-cleanup.txt")
}

func TestPlugin_AfterTestPanicKeepsFixtureCleanupArtefact(t *testing.T) {
	results := runLifecycleProbe(t, "after-test-panic", true)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, model.StatusBroken, result.Status)
	assertResultAttachment(t, result, "after-test-panic-cleanup.txt")
}

func TestPlugin_FixtureCleanupPanicStillWritesEarlierArtefact(t *testing.T) {
	results := runLifecycleProbe(t, "fixture-cleanup-panic", true)
	require.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, model.StatusBroken, result.Status)
	assertResultAttachment(t, result, "before-cleanup-panic.txt")
}

func TestPlugin_PreExecutionCaseSkipDoesNotEnterAllureLifecycle(t *testing.T) {
	results := runLifecycleProbe(t, "case-skip", false)
	assert.Empty(t, results)
}

func TestPlugin_LifecycleProbe(t *testing.T) {
	switch os.Getenv(lifecycleProbeEnv) {
	case "":
		t.Skip("lifecycle helper process")
	case "retry":
		runRetryProbe(t)
	case "parallel-retry":
		runParallelRetryProbe(t)
	case "failed":
		runFailedProbe(t)
	case "assert-fail":
		runAssertFailProbe(t)
	case "step-panic":
		runStepPanicProbe(t)
	case "runtime-skip":
		runRuntimeSkipProbe(t)
	case "case-skip":
		runCaseSkipProbe(t)
	case "test-panic":
		runTestPanicProbe(t)
	case "before-test-panic":
		runBeforeTestPanicProbe(t)
	case "after-test-panic":
		runAfterTestPanicProbe(t)
	case "fixture-cleanup-panic":
		runFixtureCleanupPanicProbe(t)
	default:
		t.Fatalf("unknown lifecycle probe %q", os.Getenv(lifecycleProbeEnv))
	}
}

func runRetryProbe(t *testing.T) {
	var attempt atomic.Int64
	runner := axiom.NewRunner(
		axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
		axiom.WithRunnerFixture("attempt-report", func(cfg *axiom.Config) (any, func(), error) {
			return struct{}{}, func() {
				cfg.Artefact(axiom.NewTextArtefact("attempt-cleanup.txt", "finished"))
			}, nil
		}),
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("RETRY-1"),
		axiom.WithCaseName("retry succeeds on the second attempt"),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		current := attempt.Add(1)
		axiom.GetFixture[struct{}](cfg, "attempt-report")
		cfg.Step(fmt.Sprintf("attempt %d", current), func() {
			cfg.Artefact(axiom.NewTextArtefact(
				"attempt.txt",
				fmt.Sprintf("attempt=%d", current),
			))
			if current == 1 {
				cfg.SubT.Error("intentional first-attempt failure")
			}
		})
	})
}

func runParallelRetryProbe(t *testing.T) {
	var firstAttempts sync.WaitGroup
	firstAttempts.Add(2)

	runner := axiom.NewRunner(
		axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
		axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)

	runParallelRetryProbeCase(t, runner, "alpha", &firstAttempts)
	runParallelRetryProbeCase(t, runner, "beta", &firstAttempts)
}

func runParallelRetryProbeCase(
	t *testing.T,
	runner *axiom.Runner,
	name string,
	firstAttempts *sync.WaitGroup,
) {
	var attempt atomic.Int64
	testName := "parallel retry " + name
	testCase := axiom.NewCase(
		axiom.WithCaseID("PARALLEL-RETRY-"+strings.ToUpper(name)),
		axiom.WithCaseName(testName),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		current := attempt.Add(1)
		if current == 1 {
			firstAttempts.Done()
			firstAttempts.Wait()
		}

		cfg.Step(fmt.Sprintf("%s attempt %d", testName, current), func() {
			cfg.Artefact(axiom.NewTextArtefact(
				testName+".txt",
				fmt.Sprintf("%s:%d", name, current),
			))
			if current == 1 {
				cfg.SubT.Errorf("intentional first attempt failure for %s", name)
			}
		})
	})
}

func runFailedProbe(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerFixture("failure-report", func(cfg *axiom.Config) (any, func(), error) {
			return struct{}{}, func() {
				cfg.Artefact(axiom.NewTextArtefact("failed-case-cleanup.txt", "finished"))
			}, nil
		}),
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("FAILED-1"),
		axiom.WithCaseName("inventory mismatch is reported"),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		axiom.GetFixture[struct{}](cfg, "failure-report")
		cfg.Step("compare inventory", func() {
			cfg.Artefact(mustLifecycleJSONArtefact(cfg, "actual-inventory.json", map[string]int{
				"expected": 7,
				"actual":   8,
			}))
			cfg.SubT.Error("inventory mismatch: expected 7, got 8")
		})
	})
}

func runAssertFailProbe(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("ASSERT-FAIL-1"),
		axiom.WithCaseName("assertion failure records message and stack"),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		cfg.Step("compare values", func() {
			assertions := require.New(testallure.T(cfg))
			assertions.Equal("expected-value", "actual-value", "values must match")
		})
	})
}

func runStepPanicProbe(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("PANIC-1"),
		axiom.WithCaseName("panic inside step is reported"),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		cfg.Step("decode response", func() {
			cfg.Artefact(axiom.NewTextArtefact("response.txt", "{"))
			panic("malformed response")
		})
	})
}

func runRuntimeSkipProbe(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerFixture("skip-report", func(cfg *axiom.Config) (any, func(), error) {
			return struct{}{}, func() {
				cfg.Artefact(axiom.NewTextArtefact("skipped-case-cleanup.txt", "finished"))
			}, nil
		}),
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("SKIP-1"),
		axiom.WithCaseName("disabled feature is skipped"),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		axiom.GetFixture[struct{}](cfg, "skip-report")
		cfg.Step("check feature flag", func() {
			cfg.Artefact(axiom.NewTextArtefact("feature-flag.txt", "enabled=false"))
			cfg.SubT.Skip("feature is disabled")
		})
	})
}

func runTestPanicProbe(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerFixture("panic-report", func(cfg *axiom.Config) (any, func(), error) {
			return struct{}{}, func() {
				cfg.Artefact(axiom.NewTextArtefact("body-panic-cleanup.txt", "finished"))
			}, nil
		}),
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("TEST-PANIC-1"),
		axiom.WithCaseName("test body panic keeps cleanup reporting active"),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		axiom.GetFixture[struct{}](cfg, "panic-report")
		panic("test body boom")
	})
}

func runBeforeTestPanicProbe(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeTest(axiom.UseFixtures("panic-report")),
			axiom.WithBeforeTest(func(*axiom.Config) {
				panic("before-test boom")
			}),
		),
		axiom.WithRunnerFixture("panic-report", func(cfg *axiom.Config) (any, func(), error) {
			return struct{}{}, func() {
				cfg.Artefact(axiom.NewTextArtefact("before-test-panic-cleanup.txt", "finished"))
			}, nil
		}),
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("BEFORE-TEST-PANIC-1"),
		axiom.WithCaseName("before-test panic keeps cleanup reporting active"),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		cfg.T().Fatal("test action must not run after BeforeTest panic")
	})
}

func runAfterTestPanicProbe(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithAfterTest(func(*axiom.Config) {
				panic("after-test boom")
			}),
		),
		axiom.WithRunnerFixture("panic-report", func(cfg *axiom.Config) (any, func(), error) {
			return struct{}{}, func() {
				cfg.Artefact(axiom.NewTextArtefact("after-test-panic-cleanup.txt", "finished"))
			}, nil
		}),
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("AFTER-TEST-PANIC-1"),
		axiom.WithCaseName("after-test panic keeps cleanup reporting active"),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		axiom.GetFixture[struct{}](cfg, "panic-report")
	})
}

func runFixtureCleanupPanicProbe(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerFixture("panic-report", func(cfg *axiom.Config) (any, func(), error) {
			return struct{}{}, func() {
				cfg.Artefact(axiom.NewTextArtefact("before-cleanup-panic.txt", "finished"))
				panic("fixture cleanup boom")
			}, nil
		}),
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("FIXTURE-CLEANUP-PANIC-1"),
		axiom.WithCaseName("fixture cleanup panic keeps earlier reporting"),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		axiom.GetFixture[struct{}](cfg, "panic-report")
	})
}

func runCaseSkipProbe(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("PRE-SKIP-1"),
		axiom.WithCaseName("unsupported environment is skipped"),
		axiom.WithCaseSkip(axiom.SkipBecause("environment is unavailable")),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		cfg.SubT.Fatal("pre-skipped action must not run")
	})
}

func runLifecycleProbe(t *testing.T, mode string, wantFailure bool) []model.TestResult {
	t.Helper()

	resultsDir := t.TempDir()
	command := exec.Command(
		os.Args[0],
		"-test.run=^TestPlugin_LifecycleProbe$",
		"-test.count=1",
		"-test.v",
	)
	command.Env = lifecycleProbeEnvironment(mode, resultsDir)

	output, err := command.CombinedOutput()
	if wantFailure {
		require.Error(t, err, "probe output:\n%s", output)
	} else {
		require.NoError(t, err, "probe output:\n%s", output)
	}

	return readAllureResults(t, resultsDir)
}

func lifecycleProbeEnvironment(mode, resultsDir string) []string {
	const resultsEnv = "ALLURE_RESULTS_DIR"

	environment := make([]string, 0, len(os.Environ())+2)
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, lifecycleProbeEnv+"=") ||
			strings.HasPrefix(item, resultsEnv+"=") {
			continue
		}
		environment = append(environment, item)
	}

	return append(
		environment,
		lifecycleProbeEnv+"="+mode,
		resultsEnv+"="+resultsDir,
	)
}

func readAllureResults(t *testing.T, resultsDir string) []model.TestResult {
	t.Helper()

	resultFiles, err := filepath.Glob(filepath.Join(resultsDir, "*-result.json"))
	require.NoError(t, err)
	sort.Strings(resultFiles)

	results := make([]model.TestResult, 0, len(resultFiles))
	for _, resultFile := range resultFiles {
		data, err := os.ReadFile(resultFile)
		require.NoError(t, err)

		var result model.TestResult
		require.NoError(t, json.Unmarshal(data, &result))
		results = append(results, result)
	}

	return results
}

func assertResultStep(
	t *testing.T,
	result model.TestResult,
	name string,
	status model.Status,
) {
	t.Helper()

	require.Len(t, result.Steps, 1)
	assert.Equal(t, name, result.Steps[0].Name)
	assert.Equal(t, status, result.Steps[0].Status)
}

func assertResultAttachment(t *testing.T, result model.TestResult, name string) {
	t.Helper()

	for _, attachment := range result.Attachments {
		if attachment.Name == name {
			return
		}
	}

	require.FailNow(t, "attachment not found", "attachment %q", name)
}

func resultWithStep(
	t *testing.T,
	results []model.TestResult,
	stepName string,
) model.TestResult {
	t.Helper()

	for _, result := range results {
		if len(result.Steps) == 1 && result.Steps[0].Name == stepName {
			return result
		}
	}

	require.FailNow(t, "result not found", "step %q", stepName)
	return model.TestResult{}
}

func mustLifecycleJSONArtefact(
	cfg *axiom.Config,
	name string,
	value any,
) axiom.Artefact {
	cfg.SubT.Helper()

	artefact, err := axiom.NewJSONArtefact(name, value)
	require.NoError(cfg.SubT, err)
	return artefact
}
