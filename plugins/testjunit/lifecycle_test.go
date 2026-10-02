package testjunit_test

import (
	"encoding/xml"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testjunit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailureReports(t *testing.T) {
	for _, probe := range []struct {
		mode           string
		cases, skipped int
		failure        string
	}{
		{mode: "direct", cases: 1, failure: "test failed"},
		{mode: "fatal", cases: 1, failure: "test failed"},
		{mode: "panic", cases: 1, failure: "body panic"},
		{mode: "retry", cases: 2, failure: "test failed"},
		{mode: "parallel-retry", cases: 2, failure: "test failed"},
		{mode: "fail-skip", cases: 2, skipped: 1, failure: "test failed"},
		{mode: "fail-and-skip", cases: 1, failure: "test failed"},
		{mode: "before-panic", cases: 1, failure: "before panic"},
		{mode: "after-panic", cases: 1, failure: "test failed"},
		{mode: "step-panic", cases: 1, failure: "step panic"},
		{mode: "setup-panic", cases: 1, failure: "setup panic"},
		{mode: "teardown-panic", cases: 1, failure: "teardown panic"},
		{mode: "cleanup-error", cases: 1, failure: "test failed"},
		{mode: "cleanup-fatal", cases: 1, failure: "test failed"},
		{mode: "late-cleanup", cases: 1, failure: "test failed"},
		{mode: "child-failure", cases: 1, failure: "test failed"},
		{mode: "fixture-error", cases: 1, failure: "fixture setup"},
		{mode: "fixture-cleanup-panic", cases: 1, failure: "fixture cleanup panic"},
		{mode: "native-cleanup-panic", cases: 1, failure: "test failed"},
	} {
		t.Run(probe.mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "junit.xml")
			exe, err := os.Executable()
			require.NoError(t, err)
			cmd := exec.Command(exe, "-test.run=^TestFailureProbe$")
			cmd.Env = append(os.Environ(), "AXIOM_JUNIT_PROBE="+probe.mode, "AXIOM_JUNIT_PATH="+path)
			output, err := cmd.CombinedOutput()
			require.Error(t, err, "failing test passed: %s", output)
			require.NotContains(t, string(output), "WARNING: DATA RACE", "data race in failure probe")
			data, err := os.ReadFile(path)
			require.NoError(t, err, "test output: %s", output)
			var report reportXML
			require.NoError(t, xml.Unmarshal(data, &report))
			assert.Equal(t, probe.cases, report.Tests)
			assert.Equal(t, 1, report.Failures)
			assert.Equal(t, probe.skipped, report.Skipped)
			require.Len(t, report.Suites, 1)
			cases := report.Suites[0].Cases
			require.Len(t, cases, probe.cases)
			require.NotNil(t, cases[0].Failure)
			assert.Equal(t, probe.failure, cases[0].Failure.Message, "test output: %s", output)
			if probe.cases == 2 {
				assert.NotEqual(t, cases[0].Name, cases[1].Name, "retry names are not unique")
			}
		})
	}
}

func TestFailureProbe(t *testing.T) {
	mode := os.Getenv("AXIOM_JUNIT_PROBE")
	if mode == "" {
		return
	}
	reporter := testjunit.NewReporter()
	t.Cleanup(func() {
		assert.NoError(t, reporter.WriteFile(os.Getenv("AXIOM_JUNIT_PATH")))
	})
	// Direct testing.T failures below are deliberate inputs to the recorder.
	// TestFailureReports verifies the resulting XML in the parent process.
	attempts := 1
	if mode == "retry" || mode == "parallel-retry" || mode == "fail-skip" {
		attempts = 2
	}
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			// This cleanup precedes the recorder's callback and therefore runs later.
			if mode == "late-cleanup" {
				cfg.Runtime.EmitEventSink(func(event axiom.Event) {
					if event.Type == axiom.EventTypeCaseStart {
						cfg.T().Cleanup(func() { cfg.T().Error("late cleanup") })
					}
				})
			}
		}, testjunit.Plugin(reporter)),
		axiom.WithRunnerRetry(axiom.WithRetryTimes(attempts)),
		axiom.WithRunnerHooks(
			axiom.WithBeforeTest(func(*axiom.Config) {
				if mode == "before-panic" {
					panic("before panic")
				}
			}),
			axiom.WithAfterTest(func(*axiom.Config) {
				if mode == "after-panic" {
					panic("after panic")
				}
			}),
		),
		axiom.WithRunnerFixture("fixture", func(*axiom.Config) (any, func(), error) {
			if mode == "fixture-error" {
				return nil, nil, errors.New("fixture setup")
			}
			return 1, func() {
				if mode == "fixture-cleanup-panic" {
					panic("fixture cleanup panic")
				}
			}, nil
		}),
	)
	if mode == "parallel-retry" {
		runner.Parallel = axiom.NewParallel(axiom.WithParallelEnabled())
	}
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("probe")), func(cfg *axiom.Config) {
		if cfg.Execution.Attempt > 1 {
			if mode == "fail-skip" {
				cfg.T().Skip("retry skipped")
			}
			return
		}
		switch mode {
		case "panic":
			panic("body panic")
		case "fatal":
			cfg.T().Fatal("direct fatal")
		case "fail-and-skip":
			cfg.T().Error("failure first")
			cfg.T().Skip("skip second")
		case "step-panic":
			cfg.Step("step", func() { panic("step panic") })
		case "setup-panic":
			cfg.Setup("setup", func() { panic("setup panic") })
		case "teardown-panic":
			cfg.Teardown("teardown", func() { panic("teardown panic") })
		case "cleanup-error":
			cfg.T().Cleanup(func() { cfg.T().Error("cleanup error") })
		case "cleanup-fatal":
			cfg.T().Cleanup(func() { cfg.T().Fatal("cleanup fatal") })
		case "native-cleanup-panic":
			cfg.T().Cleanup(func() { panic("cleanup panic") })
		case "child-failure":
			cfg.T().Run("child", func(t *testing.T) { t.Parallel(); t.Error("child failed") })
		case "fixture-error", "fixture-cleanup-panic":
			axiom.GetFixture[int](cfg, "fixture")
		case "late-cleanup", "after-panic": // Failure happens after the body.
		default:
			cfg.T().Error("direct test failure")
		}
	})
}
