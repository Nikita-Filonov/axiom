package teststats_test

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/teststats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_RealFailureLifecycles(t *testing.T) {
	for _, tc := range []struct {
		mode     string
		statuses []teststats.Status
		status   teststats.Status
		err      string
	}{
		{mode: "retry", statuses: []teststats.Status{"failed", "passed"}, status: "flaky"},
		{mode: "parallel-retry", statuses: []teststats.Status{"failed", "passed"}, status: "flaky"},
		{mode: "fail-skip", statuses: []teststats.Status{"failed", "skipped"}, status: "failed"},
		{mode: "exhausted", statuses: []teststats.Status{"failed", "failed", "failed"}, status: "failed"},
		{mode: "fatal", statuses: []teststats.Status{"failed", "passed"}, status: "flaky"},
		{mode: "before-panic", statuses: []teststats.Status{"failed", "passed"}, status: "flaky", err: "before test panic"},
		{mode: "failure-and-skip", statuses: []teststats.Status{"failed", "passed"}, status: "flaky"},
		{mode: "cleanup-fatal", statuses: []teststats.Status{"failed", "passed"}, status: "flaky"},
		{mode: "late-cleanup", statuses: []teststats.Status{"failed", "passed"}, status: "flaky"},
		{mode: "panic", statuses: []teststats.Status{"failed", "passed"}, status: "flaky", err: "body panic"},
		{mode: "fixture-error", statuses: []teststats.Status{"failed", "passed"}, status: "flaky", err: "fixture setup"},
		{mode: "fixture-cleanup", statuses: []teststats.Status{"failed", "passed"}, status: "flaky"},
		{mode: "testing-cleanup", statuses: []teststats.Status{"failed", "passed"}, status: "flaky"},
		{mode: "child-failure", statuses: []teststats.Status{"failed", "passed"}, status: "flaky"},
		{mode: "after-panic", statuses: []teststats.Status{"failed"}, status: "failed"},
		{mode: "fixture-panic", statuses: []teststats.Status{"failed"}, status: "failed", err: "fixture cleanup panic"},
		{mode: "native-cleanup-panic", statuses: []teststats.Status{"failed"}, status: "failed"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "runs.json")
			exe, err := os.Executable()
			require.NoError(t, err)
			cmd := exec.Command(exe, "-test.run=^TestStatsLifecycleProbe$")
			cmd.Env = append(os.Environ(), "AXIOM_STATS_PROBE="+tc.mode, "AXIOM_STATS_OUTPUT="+path)
			output, err := cmd.CombinedOutput()
			require.Error(t, err, "an earlier Go test failure must remain a failure: %s", output)

			data, err := os.ReadFile(path)
			require.NoError(t, err, "%s", output)
			var runs []teststats.Run
			require.NoError(t, json.Unmarshal(data, &runs))
			require.Len(t, runs, 1, "%s", output)
			run := runs[0]
			require.Len(t, run.Attempts, len(tc.statuses), "%s", output)
			assert.Equal(t, tc.status, run.Status)
			assert.Equal(t, tc.err, run.Attempts[0].Error)

			var sum time.Duration
			for i, a := range run.Attempts {
				assert.Equal(t, tc.statuses[i], a.Status)
				assert.Equal(t, i+1, a.Number)
				assert.Equal(t, run.ID, a.RunID)
				// JSON drops the monotonic clock, so exact Start/End subtraction
				// is checked in-process; Duration survives as an integer.
				assert.GreaterOrEqual(t, a.Duration, time.Duration(0))
				sum += a.Duration
				if i > 0 {
					assert.False(t, a.Start.Before(run.Attempts[i-1].End))
				}
			}
			assert.Equal(t, sum, run.Duration)
			assert.GreaterOrEqual(t, run.Elapsed, sum)
		})
	}
}

func TestStatsLifecycleProbe(t *testing.T) {
	mode := os.Getenv("AXIOM_STATS_PROBE")
	if mode == "" {
		t.Skip("subprocess only")
	}

	stats := teststats.NewStats()
	t.Cleanup(func() {
		data, err := json.Marshal(stats.Runs())
		if err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(os.Getenv("AXIOM_STATS_OUTPUT"), data, 0600); err != nil {
			t.Error(err)
		}
	})

	runner := axiom.NewRunner(
		axiom.WithRunnerRetry(axiom.WithRetryTimes(3), axiom.WithRetryDelay(time.Millisecond)),
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			if mode != "late-cleanup" {
				return
			}
			cfg.Runtime.EmitEventSink(func(event axiom.Event) {
				if event.Type == axiom.EventTypeCaseStart && cfg.Execution.Attempt == 1 {
					// Registered before teststats' cleanup, so it fails the attempt
					// after the attempt has already been recorded.
					cfg.T().Cleanup(func() { cfg.T().Error("late cleanup") })
				}
			})
		}, teststats.Plugin(stats)),
		axiom.WithRunnerHooks(axiom.WithBeforeTest(func(cfg *axiom.Config) {
			if mode == "before-panic" && cfg.Execution.Attempt == 1 {
				panic("before test panic")
			}
		}), axiom.WithAfterTest(func(*axiom.Config) {
			if mode == "after-panic" {
				panic("after test panic")
			}
		})),
		axiom.WithRunnerFixture("fixture", func(cfg *axiom.Config) (any, func(), error) {
			if cfg.Execution.Attempt == 1 && mode == "fixture-error" {
				return nil, nil, errors.New("fixture setup")
			}
			return 1, func() {
				if cfg.Execution.Attempt == 1 && mode == "fixture-cleanup" {
					cfg.T().Error("fixture cleanup")
				}
				if mode == "fixture-panic" {
					panic("fixture cleanup panic")
				}
			}, nil
		}),
	)
	if mode == "parallel-retry" {
		runner.Parallel = axiom.NewParallel(axiom.WithParallelEnabled())
	}

	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("case")), func(cfg *axiom.Config) {
		axiom.GetFixture[int](cfg, "fixture")
		if cfg.Execution.Attempt > 1 && mode != "exhausted" {
			if mode == "fail-skip" {
				cfg.T().Skip("second attempt skipped")
			}
			return
		}
		switch mode {
		case "failure-and-skip":
			cfg.T().Error("failure before skip")
			cfg.T().Skip("skip must not hide failure")
		case "cleanup-fatal":
			cfg.T().Cleanup(func() { cfg.T().Fatal("fatal cleanup") })
		case "fatal":
			cfg.T().Fatal("fatal")
		case "panic":
			panic("body panic")
		case "native-cleanup-panic":
			cfg.T().Cleanup(func() { panic("native cleanup panic") })
		case "testing-cleanup":
			cfg.T().Cleanup(func() { cfg.T().Error("testing cleanup") })
		case "child-failure":
			cfg.T().Run("child", func(t *testing.T) { t.Parallel(); t.Error("child failure") })
		case "retry", "parallel-retry", "fail-skip", "exhausted":
			cfg.T().Error("attempt failure")
		}
	})
}

func TestPlugin_GoTestSelectionDoesNotRecordExcludedCases(t *testing.T) {
	exe, err := os.Executable()
	require.NoError(t, err)
	cmd := exec.Command(exe, "-test.run=^TestStatsSelectionProbe$/^included$")
	cmd.Env = append(os.Environ(), "AXIOM_STATS_SELECTION=1")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
}

func TestStatsSelectionProbe(t *testing.T) {
	if os.Getenv("AXIOM_STATS_SELECTION") == "" {
		t.Skip("subprocess only")
	}

	stats := teststats.NewStats()
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(teststats.Plugin(stats)))
	for _, name := range []string{"included", "excluded"} {
		runner.RunCase(t, axiom.NewCase(axiom.WithCaseName(name)), func(*axiom.Config) {})
	}

	attempts := stats.Attempts()
	require.Len(t, attempts, 1)
	assert.Equal(t, "included", attempts[0].Name)
}
