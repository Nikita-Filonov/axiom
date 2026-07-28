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
	case "step-panic":
		runStepPanicProbe(t)
	case "runtime-skip":
		runRuntimeSkipProbe(t)
	case "case-skip":
		runCaseSkipProbe(t)
	default:
		t.Fatalf("unknown lifecycle probe %q", os.Getenv(lifecycleProbeEnv))
	}
}

func runRetryProbe(t *testing.T) {
	var attempt atomic.Int64
	runner := axiom.NewRunner(
		axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("RETRY-1"),
		axiom.WithCaseName("retry succeeds on the second attempt"),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		current := attempt.Add(1)
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
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("FAILED-1"),
		axiom.WithCaseName("inventory mismatch is reported"),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		cfg.Step("compare inventory", func() {
			cfg.Artefact(mustLifecycleJSONArtefact(cfg, "actual-inventory.json", map[string]int{
				"expected": 7,
				"actual":   8,
			}))
			cfg.SubT.Error("inventory mismatch: expected 7, got 8")
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
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("SKIP-1"),
		axiom.WithCaseName("disabled feature is skipped"),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		cfg.Step("check feature flag", func() {
			cfg.Artefact(axiom.NewTextArtefact("feature-flag.txt", "enabled=false"))
			cfg.SubT.Skip("feature is disabled")
		})
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
