package testleaks

import (
	"context"
	"io"
	"os"
	"os/exec"
	"runtime/pprof"
	"strings"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func waitOnChannel(ready chan<- struct{}, done <-chan struct{}) {
	close(ready)
	<-done
}

func TestGoroutineLabelsIsolateAttempts(t *testing.T) {
	firstDone := make(chan struct{})
	secondDone := make(chan struct{})
	defer func() {
		select {
		case <-firstDone:
		default:
			close(firstDone)
		}
	}()
	defer close(secondDone)
	firstID := "first-" + t.Name()
	secondID := "second-" + t.Name()

	start := func(id string, done <-chan struct{}) {
		ready := make(chan struct{})
		pprof.Do(context.Background(), pprof.Labels(labelKey, id), func(context.Context) {
			go waitOnChannel(ready, done)
		})
		<-ready
	}
	start(firstID, firstDone)
	start(secondID, secondDone)

	first, err := findGoroutines(firstID, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), goroutineCount(first))
	second, err := findGoroutines(secondID, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), goroutineCount(second))
	require.Len(t, first, 1)
	assert.Contains(t, first[0].stack, "waitOnChannel")

	ignored, err := findGoroutines(firstID, []string{
		"github.com/Nikita-Filonov/axiom/plugins/testleaks.waitOnChannel",
	})
	require.NoError(t, err)
	assert.Empty(t, ignored)
	close(firstDone)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		remaining, err := findGoroutines(firstID, nil)
		if assert.NoError(c, err) {
			assert.Empty(c, remaining, "finished worker remains in profile")
		}
	}, time.Second, time.Millisecond)
	second, err = findGoroutines(secondID, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), goroutineCount(second), "second attempt changed")
}

func TestResourceCleanupRunsBeforeVerification(t *testing.T) {
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(Plugin(
		WithoutGoroutines(), WithGracePeriod(0),
	)))
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("released in cleanup")), func(cfg *axiom.Config) {
		handle := Track(cfg, "temporary socket")
		cfg.T().Cleanup(handle.Release)
	})
}

func TestGoroutineFinishesInCleanup(t *testing.T) {
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(Plugin(WithGracePeriod(0))))
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("worker stops in cleanup")), func(cfg *axiom.Config) {
		ready := make(chan struct{})
		done := make(chan struct{})
		finished := make(chan struct{})
		go func() {
			defer close(finished)
			waitOnChannel(ready, done)
		}()
		<-ready
		cfg.T().Cleanup(func() {
			close(done)
			<-finished
		})
	})
}

type countingReadCloser struct {
	*strings.Reader
	closes int
}

func (c *countingReadCloser) Close() error {
	c.closes++
	return nil
}

func TestTrackReadCloser(t *testing.T) {
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(Plugin(
		WithoutGoroutines(), WithGracePeriod(0),
	)))
	source := &countingReadCloser{Reader: strings.NewReader("body")}
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("close response body")), func(cfg *axiom.Config) {
		body := TrackReadCloser(cfg, "response body", source)
		cfg.T().Cleanup(func() {
			assert.NoError(cfg.T(), body.Close())
		})
		got, err := io.ReadAll(body)
		require.NoError(cfg.T(), err)
		assert.Equal(cfg.T(), "body", string(got))
	})
	assert.Equal(t, 1, source.closes)
}

func TestPolicySkipDoesNotRunCheck(t *testing.T) {
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(Plugin()))
	runner.RunCase(t, axiom.NewCase(
		axiom.WithCaseName("skipped"),
		axiom.WithCaseSkip(axiom.SkipBecause("disabled")),
	), func(cfg *axiom.Config) { assert.Fail(cfg.T(), "skipped body ran") })
}

func TestDuplicateInstallationAndValidation(t *testing.T) {
	cfg := &axiom.Config{}
	plugin := Plugin()
	plugin(cfg)
	plugin(cfg)
	assert.Len(t, cfg.Runtime.TestWraps, 1)
	for _, tc := range []struct {
		fn   func()
		name string
	}{
		{name: "nil config", fn: func() { plugin(nil) }},
		{name: "nil option", fn: func() { Plugin(nil) }},
		{name: "negative grace", fn: func() { WithGracePeriod(-time.Second) }},
		{name: "missing plugin", fn: func() { Track(&axiom.Config{}, "connection") }},
		{name: "empty name", fn: func() { Track(cfg, "") }},
		{name: "nil read closer", fn: func() { TrackReadCloser(cfg, "body", nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Panics(t, tc.fn)
		})
	}
}

func TestLeakProbe(t *testing.T) {
	mode := os.Getenv("AXIOM_TESTLEAKS_PROBE")
	if mode == "" {
		return
	}
	options := []Option{WithGracePeriod(20 * time.Millisecond)}
	if mode == "resource" {
		options = append(options, WithoutGoroutines())
	}
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(Plugin(options...)))
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName(mode)), func(cfg *axiom.Config) {
		switch mode {
		case "resource":
			Track(cfg, "database cursor")
		case "goroutine":
			ready := make(chan struct{})
			done := make(chan struct{})
			go waitOnChannel(ready, done)
			<-ready
		default:
			require.FailNow(cfg.T(), "unknown probe mode", "mode: %s", mode)
		}
	})
}

func TestLeakFailures(t *testing.T) {
	for _, probe := range []struct {
		mode string
		want string
	}{
		{mode: "resource", want: "tracked resource(s) not released"},
		{mode: "goroutine", want: "goroutine(s) still running"},
	} {
		t.Run(probe.mode, func(t *testing.T) {
			exe, err := os.Executable()
			require.NoError(t, err)
			cmd := exec.Command(exe, "-test.run=^TestLeakProbe$", "-test.v")
			cmd.Env = append(os.Environ(), "AXIOM_TESTLEAKS_PROBE="+probe.mode)
			output, err := cmd.CombinedOutput()
			require.Error(t, err, "leaking case passed:\n%s", output)
			require.NotContains(t, string(output), "WARNING: DATA RACE", "data race in leak probe")
			assert.Contains(t, string(output), probe.want)
			if probe.mode == "resource" {
				assert.Contains(t, string(output), "plugin_test.go:", "missing resource registration site")
			}
		})
	}
}
