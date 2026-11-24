package testjunit

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type outputTotals struct {
	Tests    int `xml:"tests,attr"`
	Failures int `xml:"failures,attr"`
	Skipped  int `xml:"skipped,attr"`
}

func readOutput(t *testing.T, path string) outputTotals {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var result outputTotals
	require.NoError(t, xml.Unmarshal(data, &result))
	return result
}

func TestAutomaticOutputWithoutReporter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "junit.xml")
	plugin := Plugin(WithOutputFile(path), WithSuiteName("service"))
	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist, "plugin construction must not create files")
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(plugin))
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("passed")), func(*axiom.Config) {})
	assert.Equal(t, 1, readOutput(t, path).Tests)
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseSkip(axiom.SkipBecause("disabled"))), func(*axiom.Config) {
		t.Fatal("skipped case executed")
	})
	assert.Equal(t, outputTotals{Tests: 2, Skipped: 1}, readOutput(t, path))
}

func TestOutputRegistrationWaitsForInstallation(t *testing.T) {
	reporter := NewReporter()
	path := filepath.Join(t.TempDir(), "junit.xml")
	pending := Plugin(WithReporter(reporter), WithOutputFile(path))
	assert.Empty(t, reporter.outputs)
	assert.Panics(t, func() { pending(nil) })
	assert.Empty(t, reporter.outputs, "invalid installation changed the reporter")

	runner := axiom.NewRunner(axiom.WithRunnerPlugins(Plugin(WithReporter(reporter))))
	runner.RunCase(t, axiom.NewCase(), func(*axiom.Config) {})
	_, err := os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist, "unused plugin affected an active reporter")

	pending(&axiom.Config{})
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist, "installation must not write files")
	runner.RunCase(t, axiom.NewCase(), func(*axiom.Config) {})
	assert.Equal(t, 2, readOutput(t, path).Tests, "output must include the entire shared reporter")
}

func TestRepeatedInstallationAddsOutputWithoutAnotherRecorder(t *testing.T) {
	reporter := NewReporter()
	first := filepath.Join(t.TempDir(), "first.xml")
	second := filepath.Join(t.TempDir(), "second.xml")
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(
		Plugin(WithReporter(reporter), WithSuiteName("first"), WithOutputFile(first)),
		Plugin(WithReporter(reporter), WithSuiteName("ignored"), WithOutputFile(second)),
	))
	runner.RunCase(t, axiom.NewCase(), func(*axiom.Config) {})
	assert.Equal(t, 1, readOutput(t, first).Tests)
	assert.Equal(t, 1, readOutput(t, second).Tests)
	results := reporter.snapshot()
	require.Len(t, results, 1)
	assert.Equal(t, "first", results[0].SuiteName)
}

func TestSharedReporterPublishesAllSuitesAndOutputs(t *testing.T) {
	reporter := NewReporter()
	first := filepath.Join(t.TempDir(), "first.xml")
	second := filepath.Join(t.TempDir(), "second.xml")
	plugins := []axiom.Plugin{
		Plugin(WithReporter(reporter), WithSuiteName("first"), WithOutputFile(first)),
		Plugin(WithReporter(reporter), WithSuiteName("second"), WithOutputFile(second)),
		Plugin(WithReporter(reporter), WithSuiteName("memory")),
	}
	t.Run("parallel", func(t *testing.T) {
		for i := range 24 {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				t.Parallel()
				runner := axiom.NewRunner(axiom.WithRunnerPlugins(plugins[i%len(plugins)]))
				runner.RunCase(t, axiom.NewCase(), func(*axiom.Config) {})
				// A manual export participates in the same serialization.
				require.NoError(t, reporter.WriteFile(first))
			})
		}
	})
	assert.Equal(t, 24, readOutput(t, first).Tests)
	assert.Equal(t, 24, readOutput(t, second).Tests)
	assert.Len(t, reporter.snapshot().document().Suites, 3)
}

func TestOutputFailureHandlerCanExportAndRecovery(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	require.NoError(t, os.WriteFile(blocked, []byte("file"), 0600))
	path := filepath.Join(blocked, "junit.xml")
	reporter := NewReporter()
	var failures []error
	// Constructed first, installed last: installation order selects the handler.
	duplicate := Plugin(WithReporter(reporter), WithOutputFile(path), WithErrorHandler(func(error) {
		t.Error("duplicate handler used")
	}))
	plugin := Plugin(WithReporter(reporter), WithOutputFile(path), WithErrorHandler(func(err error) {
		failures = append(failures, err)
		var buffer bytes.Buffer
		require.NoError(t, reporter.Write(&buffer))
		require.NoError(t, reporter.WriteFile(filepath.Join(root, "fallback.xml")))
	}))
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(plugin, duplicate, plugin))
	runner.RunCase(t, axiom.NewCase(), func(*axiom.Config) {})
	require.Len(t, failures, 2)
	assert.Contains(t, failures[0].Error(), path)
	assert.Equal(t, 1, readOutput(t, filepath.Join(root, "fallback.xml")).Tests)
	require.NoError(t, os.Remove(blocked))
	runner.RunCase(t, axiom.NewCase(), func(*axiom.Config) {})
	assert.Equal(t, 2, readOutput(t, path).Tests)
	assert.Len(t, failures, 2)
}

func TestConcurrentOutputRegistration(t *testing.T) {
	reporter := NewReporter()
	path := filepath.Join(t.TempDir(), "report.xml")
	var group sync.WaitGroup
	for range 10 {
		group.Go(func() {
			Plugin(WithReporter(reporter), WithOutputFile(path))(&axiom.Config{})
			reporter.publish()
		})
	}
	group.Wait()
	assert.Len(t, reporter.outputs, 1)
	assert.Zero(t, readOutput(t, path).Tests)
}

func TestDefaultOutputErrorLogging(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocked, []byte("keep"), 0600))
	var buffer bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buffer)
	t.Cleanup(func() { log.SetOutput(previous) })
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(Plugin(WithOutputFile(filepath.Join(blocked, "junit.xml")))))
	runner.RunCase(t, axiom.NewCase(), func(*axiom.Config) {})
	assert.Contains(t, buffer.String(), "testjunit: export")
	assert.Contains(t, buffer.String(), blocked)
}

func TestCompletedOutputSurvivesProcessExit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "junit.xml")
	executable, err := os.Executable()
	require.NoError(t, err)
	command := exec.Command(executable, "-test.run=^TestOutputExitProbe$")
	command.Env = append(os.Environ(), "AXIOM_JUNIT_EXIT_PATH="+path)
	output, err := command.CombinedOutput()
	var exitError *exec.ExitError
	require.ErrorAs(t, err, &exitError, "%s", output)
	assert.Equal(t, 23, exitError.ExitCode())
	assert.Equal(t, outputTotals{Tests: 1}, readOutput(t, path))
}

func TestOutputExitProbe(t *testing.T) {
	path := os.Getenv("AXIOM_JUNIT_EXIT_PATH")
	if path == "" {
		return
	}
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(Plugin(WithOutputFile(path))))
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("completed")), func(*axiom.Config) {})
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("interrupted")), func(*axiom.Config) {
		os.Exit(23)
	})
}

func TestParallelOutputSurvivesInterruptedSibling(t *testing.T) {
	for _, mode := range []string{"exit", "panic", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "junit.xml")
			command := exec.Command(os.Args[0], "-test.run=^TestParallelOutputProbe$", "-test.parallel=2", "-test.timeout=2s")
			command.Env = append(os.Environ(), "AXIOM_JUNIT_PARALLEL_MODE="+mode, "AXIOM_JUNIT_PARALLEL_PATH="+path)
			output, err := command.CombinedOutput()
			require.Error(t, err, "%s", output)
			require.Contains(t, string(output), "completed attempt persisted before final sink")
			require.NotContains(t, string(output), "WARNING: DATA RACE")
			assert.GreaterOrEqual(t, readOutput(t, path).Tests, 1, "completed attempts must survive an interrupted sibling")
			if mode == "timeout" {
				assert.Contains(t, string(output), "test timed out")
			}
		})
	}
}

func TestParallelOutputProbe(t *testing.T) {
	mode := os.Getenv("AXIOM_JUNIT_PARALLEL_MODE")
	if mode == "" {
		return
	}

	path := os.Getenv("AXIOM_JUNIT_PARALLEL_PATH")
	published := make(chan struct{})
	var confirmed atomic.Bool
	runner := axiom.NewRunner(
		axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
		axiom.WithRunnerPlugins(
			func(cfg *axiom.Config) {
				if cfg.Case.Name != "completed" {
					return
				}
				cfg.Runtime.EmitEventSink(func(event axiom.Event) {
					if event.Type == axiom.EventTypeCaseStart {
						// Registered before the recorder's cleanup, so publication
						// finishes before the sibling is released.
						cfg.T().Cleanup(func() { close(published) })
					}
				})
				cfg.Runtime.EmitAttemptSink(func(axiom.Attempt) { confirmed.Store(true) })
			},
			Plugin(WithOutputFile(path)),
		),
	)
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("completed")), func(*axiom.Config) {})
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("interrupted")), func(cfg *axiom.Config) {
		<-published
		require.False(cfg.T(), confirmed.Load(), "parent has not reached final result delivery")
		require.Equal(cfg.T(), outputTotals{Tests: 1}, readOutput(cfg.T(), path))
		fmt.Println("completed attempt persisted before final sink")
		switch mode {
		case "exit":
			os.Exit(23)
		case "panic":
			cfg.T().Cleanup(func() { panic("sibling cleanup panic") })
		case "timeout":
			select {}
		}
	})
}
