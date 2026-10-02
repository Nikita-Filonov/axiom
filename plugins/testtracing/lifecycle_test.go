package testtracing_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testtracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_RunnerAndCaseInstallation(t *testing.T) {
	for _, parallel := range []bool{false, true} {
		t.Run(fmt.Sprintf("parallel=%t", parallel), func(t *testing.T) {
			trace := testtracing.NewTrace()
			t.Run("suite", func(t *testing.T) {
				runner := axiom.NewRunner(axiom.WithRunnerPlugins(testtracing.Plugin(trace)))
				if parallel {
					runner.Parallel = axiom.NewParallel(axiom.WithParallelEnabled())
				}
				for i := range 8 {
					runner.RunCase(t, axiom.NewCase(
						axiom.WithCaseName(fmt.Sprintf("case-%d", i)),
						axiom.WithCasePlugins(testtracing.Plugin(trace)),
					), func(cfg *axiom.Config) {
						event := axiom.Event{Type: axiom.EventTypeLog, Message: "body"}
						cfg.Event(event)
						cfg.Event(event)
						cfg.T().Cleanup(func() {
							cfg.Event(axiom.Event{Type: axiom.EventTypeLog, Message: "cleanup"})
						})
					})
				}
			})
			records := trace.Snapshot()
			require.Len(t, records, 8, "planning must not create records without events")
			seen := make(map[string]bool)
			for _, record := range records {
				assert.NotContains(t, seen, record.Case.Name)
				seen[record.Case.Name] = true
				require.Len(t, record.Events, 5)
				assert.Equal(t, axiom.EventTypeCaseStart, record.Events[0].Type)
				assert.Equal(t, axiom.Event{Type: axiom.EventTypeLog, Message: "body"}, record.Events[1])
				assert.Equal(t, record.Events[1], record.Events[2])
				assert.Equal(t, axiom.EventTypeCaseFinish, record.Events[3].Type)
				assert.Equal(t, axiom.Event{Type: axiom.EventTypeLog, Message: "cleanup"}, record.Events[4])
			}
		})
	}
}

func TestPlugin_PolicySkipRecordedOnce(t *testing.T) {
	for _, parallel := range []bool{false, true} {
		t.Run(fmt.Sprintf("parallel=%t", parallel), func(t *testing.T) {
			trace := testtracing.NewTrace()
			runner := axiom.NewRunner(
				axiom.WithRunnerPlugins(testtracing.Plugin(trace)),
				axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
			)
			if parallel {
				runner.Parallel = axiom.NewParallel(axiom.WithParallelEnabled())
			}
			runner.RunCase(t, axiom.NewCase(
				axiom.WithCaseName("skipped"),
				axiom.WithCasePlugins(testtracing.Plugin(trace)),
				axiom.WithCaseSkip(axiom.SkipBecause("disabled")),
			), func(cfg *axiom.Config) { assert.Fail(cfg.T(), "skipped body ran") })
			records := trace.Snapshot()
			require.Len(t, records, 1)
			require.Len(t, records[0].Events, 1)
			assert.Equal(t, axiom.EventTypeCaseSkip, records[0].Events[0].Type)
			assert.Equal(t, "disabled", records[0].Events[0].Message)
		})
	}
}

func TestPlugin_RetryAttemptsRemainSeparate(t *testing.T) {
	for _, mode := range []string{"sequential", "parallel"} {
		t.Run(mode, func(t *testing.T) {
			exe, err := os.Executable()
			require.NoError(t, err)
			path := filepath.Join(t.TempDir(), "events.json")
			cmd := exec.Command(exe, "-test.run=^TestPluginRetryProbe$")
			cmd.Env = append(os.Environ(), "AXIOM_TRACING_RETRY="+mode, "AXIOM_TRACING_PATH="+path)
			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr, "probe output: %s", output)
			assert.Equal(t, 1, exitErr.ExitCode())
			require.NotContains(t, string(output), "WARNING: DATA RACE")
			data, err := os.ReadFile(path)
			require.NoError(t, err, "probe output: %s", output)
			var records [][]axiom.Event
			require.NoError(t, json.Unmarshal(data, &records))
			require.Len(t, records, 2)
			for i, events := range records {
				require.Len(t, events, 3)
				assert.Equal(t, axiom.EventTypeCaseStart, events[0].Type)
				assert.Equal(t, axiom.Event{Type: axiom.EventTypeLog, Message: strconv.Itoa(i + 1)}, events[1])
				assert.Equal(t, axiom.EventTypeCaseFinish, events[2].Type)
			}
		})
	}
}

func TestPluginRetryProbe(t *testing.T) {
	mode := os.Getenv("AXIOM_TRACING_RETRY")
	if mode == "" {
		return
	}
	trace := testtracing.NewTrace()
	t.Cleanup(func() {
		var events [][]axiom.Event
		for _, record := range trace.Snapshot() {
			events = append(events, record.Events)
		}
		data, err := json.Marshal(events)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(os.Getenv("AXIOM_TRACING_PATH"), data, 0600))
	})
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(testtracing.Plugin(trace)),
		axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
	)
	if mode == "parallel" {
		runner.Parallel = axiom.NewParallel(axiom.WithParallelEnabled())
	}
	runner.RunCase(t, axiom.NewCase(
		axiom.WithCaseName("retried"),
		axiom.WithCasePlugins(testtracing.Plugin(trace)),
	), func(cfg *axiom.Config) {
		cfg.Event(axiom.Event{Type: axiom.EventTypeLog, Message: strconv.Itoa(cfg.Execution.Attempt)})
		if cfg.Execution.Attempt == 1 {
			assert.Fail(cfg.T(), "intentional first attempt failure")
		}
	})
}
