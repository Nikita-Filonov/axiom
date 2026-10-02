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
	if err != nil || goroutineCount(first) != 1 {
		t.Fatalf("first attempt: count=%d, err=%v", goroutineCount(first), err)
	}
	second, err := findGoroutines(secondID, nil)
	if err != nil || goroutineCount(second) != 1 {
		t.Fatalf("second attempt: count=%d, err=%v", goroutineCount(second), err)
	}
	if !strings.Contains(first[0].stack, "waitOnChannel") {
		t.Fatalf("missing worker stack: %s", first[0].stack)
	}

	ignored, err := findGoroutines(firstID, []string{
		"github.com/Nikita-Filonov/axiom/plugins/testleaks.waitOnChannel",
	})
	if err != nil || len(ignored) != 0 {
		t.Fatalf("ignored goroutine: count=%d, err=%v", len(ignored), err)
	}
	close(firstDone)
	deadline := time.Now().Add(time.Second)
	for {
		first, err = findGoroutines(firstID, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(first) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("finished worker remains in profile: %v", first)
		}
		time.Sleep(time.Millisecond)
	}
	second, err = findGoroutines(secondID, nil)
	if err != nil || goroutineCount(second) != 1 {
		t.Fatalf("second attempt changed: count=%d, err=%v", goroutineCount(second), err)
	}
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
		got, err := io.ReadAll(body)
		if err != nil || string(got) != "body" {
			cfg.T().Fatalf("read: body=%q, err=%v", got, err)
		}
		cfg.T().Cleanup(func() {
			if err := body.Close(); err != nil {
				cfg.T().Error(err)
			}
		})
	})
	if source.closes != 1 {
		t.Fatalf("Close calls = %d, want 1", source.closes)
	}
}

func TestPolicySkipDoesNotRunCheck(t *testing.T) {
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(Plugin()))
	runner.RunCase(t, axiom.NewCase(
		axiom.WithCaseName("skipped"),
		axiom.WithCaseSkip(axiom.SkipBecause("disabled")),
	), func(*axiom.Config) { t.Fatal("skipped body ran") })
}

func TestDuplicateInstallationAndValidation(t *testing.T) {
	cfg := &axiom.Config{}
	plugin := Plugin()
	plugin(cfg)
	plugin(cfg)
	if len(cfg.Runtime.TestWraps) != 1 {
		t.Fatalf("test wraps = %d, want 1", len(cfg.Runtime.TestWraps))
	}
	assertPanic := func(name string, fn func()) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("expected panic")
				}
			}()
			fn()
		})
	}
	assertPanic("nil config", func() { plugin(nil) })
	assertPanic("nil option", func() { Plugin(nil) })
	assertPanic("negative grace", func() { WithGracePeriod(-time.Second) })
	assertPanic("missing plugin", func() { Track(&axiom.Config{}, "connection") })
	assertPanic("empty name", func() { Track(cfg, "") })
	assertPanic("nil read closer", func() { TrackReadCloser(cfg, "body", nil) })
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
			t.Fatalf("unknown probe mode: %s", mode)
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
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(exe, "-test.run=^TestLeakProbe$", "-test.v")
			cmd.Env = append(os.Environ(), "AXIOM_TESTLEAKS_PROBE="+probe.mode)
			output, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("leaking case passed:\n%s", output)
			}
			if !strings.Contains(string(output), probe.want) {
				t.Fatalf("missing %q in output:\n%s", probe.want, output)
			}
			if probe.mode == "resource" && !strings.Contains(string(output), "plugin_test.go:") {
				t.Fatalf("missing resource registration site:\n%s", output)
			}
		})
	}
}
