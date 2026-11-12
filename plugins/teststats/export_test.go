package teststats_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/teststats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These consumer-side types intentionally do not share the writer's wire types.
type exportedAttempt struct {
	SchemaVersion int              `json:"schema_version"`
	Kind          string           `json:"kind"`
	ProcessID     string           `json:"process_id"`
	RunID         string           `json:"run_id"`
	Number        int              `json:"number"`
	Revision      uint64           `json:"revision"`
	State         string           `json:"state"`
	Name          string           `json:"name"`
	Status        teststats.Status `json:"status"`
	Error         string           `json:"error"`
	Start         time.Time        `json:"started_at"`
	End           time.Time        `json:"finished_at"`
	Duration      time.Duration    `json:"duration_ns"`
}

type exportedRun struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	ProcessID     string `json:"process_id"`
	RunID         string `json:"run_id"`
	Attempts      []struct {
		Number   int    `json:"number"`
		Revision uint64 `json:"revision"`
		State    string `json:"state"`
	} `json:"attempts"`
	Summary *struct {
		Status   teststats.Status `json:"status"`
		Duration time.Duration    `json:"duration_ns"`
	} `json:"summary"`
}

func readSnapshots[T any](t *testing.T, dir, prefix string) []T {
	t.Helper()
	var records []T
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) || !strings.HasSuffix(entry.Name(), ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var record T
		if err := json.Unmarshal(data, &record); err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		records = append(records, record)
		return nil
	})
	require.NoError(t, err)
	return records
}

func TestPlugin_ExportsLifecycleAndDeduplicatesInstallations(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "results[1]")
	one, two := teststats.NewStats(), teststats.NewStats()
	plugin := teststats.Plugin(teststats.WithStats(one), teststats.WithOutputDir(dir))
	require.NoDirExists(t, dir, "constructing a plugin does not write files")
	t.Run("scope", func(t *testing.T) {
		runner := axiom.NewRunner(axiom.WithRunnerPlugins(
			teststats.Plugin(teststats.WithStats(one)),
			plugin,
			teststats.Plugin(teststats.WithStats(two)),
		))
		c := axiom.NewCase(axiom.WithCaseName("case"), axiom.WithCasePlugins(
			teststats.Plugin(teststats.WithStats(one), teststats.WithOutputDir(dir)),
		))
		runner.RunCase(t, c, func(cfg *axiom.Config) {
			attempts := readSnapshots[exportedAttempt](t, dir, "attempt-")
			require.Len(t, attempts, 1)
			assert.Equal(t, "running", attempts[0].State)
			assert.Equal(t, uint64(1), attempts[0].Revision)
			assert.Empty(t, attempts[0].Status)
			assert.True(t, attempts[0].End.IsZero())
			assert.Empty(t, one.Attempts())
			runs := readSnapshots[exportedRun](t, dir, "run.json")
			require.Len(t, runs, 1)
			assert.Nil(t, runs[0].Summary)
			loaded, err := teststats.Load(dir)
			require.NoError(t, err)
			assert.True(t, loaded.Incomplete())
			assert.Empty(t, loaded.Stats.Attempts())
			assert.Zero(t, loaded.Stats.Elapsed())
			require.Len(t, loaded.Unfinished, 1)
			assert.Equal(t, teststats.StateRunning, loaded.Unfinished[0].State)
			cfg.T().Cleanup(func() {
				assert.Equal(t, "running", readSnapshots[exportedAttempt](t, dir, "attempt-")[0].State)
			})
		})
		attempts := readSnapshots[exportedAttempt](t, dir, "attempt-")
		require.Len(t, attempts, 1)
		assert.Equal(t, "finished", attempts[0].State)
		assert.Equal(t, uint64(2), attempts[0].Revision)
		assert.Equal(t, teststats.StatusPassed, attempts[0].Status)
	})
	attempts := readSnapshots[exportedAttempt](t, dir, "attempt-")
	require.Len(t, attempts, 1)
	a := attempts[0]
	assert.Equal(t, 1, a.SchemaVersion)
	assert.Equal(t, "attempt", a.Kind)
	assert.Equal(t, "finalized", a.State)
	assert.Equal(t, uint64(3), a.Revision)
	require.Len(t, one.Attempts(), 1)
	require.Len(t, two.Attempts(), 1)
	assert.Equal(t, one.Attempts()[0].RunID, a.RunID)
	assert.Equal(t, one.Attempts()[0].Duration, a.Duration)
	runs := readSnapshots[exportedRun](t, dir, "run.json")
	require.Len(t, runs, 1)
	assert.Equal(t, "run", runs[0].Kind)
	assert.Equal(t, a.RunID, runs[0].RunID)
	assert.Equal(t, a.ProcessID, runs[0].ProcessID)
	require.Len(t, runs[0].Attempts, 1)
	assert.Equal(t, a.Revision, runs[0].Attempts[0].Revision)
	require.NotNil(t, runs[0].Summary)
	assert.Equal(t, teststats.StatusPassed, runs[0].Summary.Status)
	loaded, err := teststats.Load(dir)
	require.NoError(t, err)
	assert.False(t, loaded.Incomplete())
	assert.Equal(t, one.Summary(), loaded.Stats.Summary())
	assert.Equal(t, one.Attempts()[0].ProcessID, loaded.Stats.Attempts()[0].ProcessID)
	assert.True(t, one.Attempts()[0].Start.Equal(loaded.Stats.Attempts()[0].Start))
	assert.Equal(t, a.End.Sub(a.Start), loaded.Stats.Elapsed())
	recorded := one.Attempts()[0]
	assert.Equal(t, recorded.End.Sub(recorded.Start), one.Elapsed())
}

func TestPlugin_ExportsParallelRunsWithRepeatedNames(t *testing.T) {
	dir := t.TempDir()
	t.Run("scope", func(t *testing.T) {
		runner := axiom.NewRunner(
			axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
			axiom.WithRunnerPlugins(teststats.Plugin(teststats.WithOutputDir(dir))),
		)
		for range 12 {
			runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("same")), func(*axiom.Config) {})
		}
	})
	attempts := readSnapshots[exportedAttempt](t, dir, "attempt-")
	require.Len(t, attempts, 12)
	identities := make(map[string]bool)
	for _, a := range attempts {
		assert.Equal(t, "finalized", a.State)
		assert.Equal(t, teststats.StatusPassed, a.Status)
		assert.False(t, identities[a.RunID], "runs must not overwrite each other")
		identities[a.RunID] = true
	}
	assert.Len(t, readSnapshots[exportedRun](t, dir, "run.json"), 12)
}

func TestPlugin_ExportsSkippedAttempts(t *testing.T) {
	dir := t.TempDir()
	t.Run("scope", func(t *testing.T) {
		runner := axiom.NewRunner(axiom.WithRunnerPlugins(teststats.Plugin(teststats.WithOutputDir(dir))))
		runner.RunCase(t, axiom.NewCase(axiom.WithCaseSkip(axiom.SkipBecause("maintenance"))), func(*axiom.Config) {
			t.Error("skipped body ran")
		})
		runner.RunCase(t, axiom.NewCase(), func(cfg *axiom.Config) { cfg.T().Skip("runtime skip") })
	})
	attempts := readSnapshots[exportedAttempt](t, dir, "attempt-")
	require.Len(t, attempts, 2)
	for _, a := range attempts {
		assert.Equal(t, teststats.StatusSkipped, a.Status)
		assert.Equal(t, "finalized", a.State)
	}
}

func TestPlugin_ExportsIndependentConfigsWithoutExecutionID(t *testing.T) {
	dir := t.TempDir()
	stats := teststats.NewStats()
	plugin := teststats.Plugin(teststats.WithStats(stats), teststats.WithOutputDir(dir))
	for range 2 {
		t.Run("direct", func(t *testing.T) {
			cfg := &axiom.Config{
				RootT: t,
				Case:  &axiom.Case{},
			}
			plugin(cfg)
			cfg.Event(axiom.NewEvent(axiom.EventTypeCaseStart))
		})
	}
	attempts := readSnapshots[exportedAttempt](t, dir, "attempt-")
	require.Len(t, attempts, 2)
	assert.NotEqual(t, attempts[0].RunID, attempts[1].RunID)
	for _, a := range attempts {
		assert.Equal(t, "finished", a.State, "there is no parent to confirm late failures")
	}
	for _, a := range stats.Attempts() {
		assert.Empty(t, a.RunID, "artifact IDs do not change the in-memory API")
	}
}

func TestPlugin_OutputFailureDoesNotFailTestOrHoldRecorderLock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(dir, []byte("unchanged"), 0600))
	stats := teststats.NewStats()
	var failures []error
	t.Run("scope", func(t *testing.T) {
		runner := axiom.NewRunner(axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			teststats.Plugin(
				teststats.WithStats(stats),
				teststats.WithOutputDir(dir),
				teststats.WithErrorHandler(func(err error) {
					failures = append(failures, err)
					stats.Attempts()
					cfg.Event(axiom.NewEvent(axiom.EventTypeCaseStart))
				}),
			)(cfg)
		}))
		runner.RunCase(t, axiom.NewCase(), func(*axiom.Config) {})
	})
	require.Len(t, failures, 3)
	assert.Equal(t, teststats.StatusPassed, stats.Attempts()[0].Status)
	data, err := os.ReadFile(dir)
	require.NoError(t, err)
	assert.Equal(t, "unchanged", string(data))
}

func TestPlugin_ArtifactsSurviveInterruptedProcess(t *testing.T) {
	for _, mode := range []string{"kill", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			exe, err := os.Executable()
			require.NoError(t, err)
			cmd := exec.Command(exe, "-test.run=^TestStatsInterruptedProbe$", "-test.timeout=3s")
			cmd.Env = append(os.Environ(), "AXIOM_STATS_INTERRUPTED="+dir)
			stdout, err := cmd.StdoutPipe()
			require.NoError(t, err)
			require.NoError(t, cmd.Start())
			t.Cleanup(func() {
				if cmd.ProcessState == nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			})
			ready := make(chan struct{})
			readDone := make(chan struct{})
			go func() {
				defer close(readDone)
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					if scanner.Text() == "AXIOM_STATS_READY" {
						close(ready)
					}
				}
			}()
			select {
			case <-ready:
			case <-time.After(10 * time.Second):
				_ = cmd.Process.Kill()
				<-readDone
				_ = cmd.Wait()
				t.Fatal("probe did not start its second attempt")
			}
			if mode == "kill" {
				require.NoError(t, cmd.Process.Kill())
			}
			// Drain StdoutPipe before Wait closes it.
			<-readDone
			require.Error(t, cmd.Wait())
			attempts := readSnapshots[exportedAttempt](t, dir, "attempt-")
			require.Len(t, attempts, 2)
			for _, a := range attempts {
				if a.Name == "completed" {
					assert.Equal(t, "finished", a.State)
					assert.Equal(t, teststats.StatusPassed, a.Status)
				} else {
					assert.Equal(t, "interrupted", a.Name)
					assert.Equal(t, "running", a.State)
					assert.Empty(t, a.Status)
				}
			}
			loaded, err := teststats.Load(dir)
			require.NoError(t, err)
			assert.True(t, loaded.Incomplete())
			assert.Empty(t, loaded.Issues)
			assert.Len(t, loaded.Unfinished, 2)
			assert.Equal(t, 1, loaded.Stats.Summary().Attempts.Passed)
		})
	}
}

func TestStatsInterruptedProbe(t *testing.T) {
	dir := os.Getenv("AXIOM_STATS_INTERRUPTED")
	if dir == "" {
		t.Skip("subprocess only")
	}
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(teststats.Plugin(teststats.WithOutputDir(dir))))
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("completed")), func(*axiom.Config) {})
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("interrupted")), func(*axiom.Config) {
		fmt.Println("AXIOM_STATS_READY")
		time.Sleep(time.Minute)
	})
}
