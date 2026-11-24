package testtimeout

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_NoTimeout_DoesNotWrap(t *testing.T) {
	cfg := &axiom.Config{}

	Plugin(WithContextDeadline())(cfg)

	assert.Empty(t, cfg.Runtime.TestWraps, "a non-positive timeout must be a no-op")
}

func TestPlugin_WithTimeout_AddsWrap(t *testing.T) {
	cfg := &axiom.Config{}

	Plugin(WithTimeout(time.Second))(cfg)

	assert.Len(t, cfg.Runtime.TestWraps, 1)
}

func TestPlugin_PreparedAttemptRuns(t *testing.T) {
	cfg := &axiom.Config{
		SubT:    t,
		Attempt: axiom.Attempt{Decision: axiom.Decision{Kind: axiom.DecisionRun}},
	}
	Plugin(WithTimeout(time.Second))(cfg)

	ran := false
	cfg.Runtime.TestWraps[0](func(*axiom.Config) { ran = true })(cfg)

	assert.True(t, ran)
}

func TestPlugin_PassingCaseRunsBody(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(Plugin(WithTimeout(5 * time.Second))),
	)

	ran := false
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("fast")), func(cfg *axiom.Config) {
		ran = true
	})

	assert.True(t, ran, "a case that finishes within the budget must run normally")
}

func TestPolicySkipBypassesExecutionInstrumentation(t *testing.T) {
	var outcomes []axiom.Attempt
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(Plugin(WithTimeout(time.Nanosecond), WithContextDeadline())),
		axiom.WithRunnerRuntime(axiom.WithRuntimeAttemptSink(func(e axiom.Attempt) {
			outcomes = append(outcomes, e)
		})),
	)
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseSkip(axiom.SkipBecause("disabled"))), func(c *axiom.Config) {
		c.T().Error("skipped body ran")
	})
	if assert.Len(t, outcomes, 1) {
		assert.Equal(t, axiom.AttemptSkipped, outcomes[0].Outcome.Status)
	}
}

func TestPlugin_TimeoutDoesNotRaceWithAttemptIdentity(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestPlugin_TimeoutIdentityHelperProcess$", "-test.v")
	cmd.Env = append(os.Environ(), "AXIOM_TIMEOUT_IDENTITY_HELPER=1")

	output, err := cmd.CombinedOutput()
	require.Error(t, err, "the helper attempt intentionally times out")
	assert.NotContains(t, string(output), "WARNING: DATA RACE")
	assert.Contains(t, string(output), "timeout identity reader stopped")
}

func TestPlugin_TimeoutIdentityHelperProcess(t *testing.T) {
	if os.Getenv("AXIOM_TIMEOUT_IDENTITY_HELPER") != "1" {
		t.Skip("helper process")
	}

	stop := make(chan struct{})
	readerDone := make(chan struct{})
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(Plugin(WithTimeout(20*time.Millisecond), WithoutGoroutineDump())),
		axiom.WithRunnerRuntime(axiom.WithRuntimeAttemptSink(func(axiom.Attempt) {
			close(stop)
			<-readerDone
		})),
	)

	runner.RunCase(t, axiom.NewCase(), func(cfg *axiom.Config) {
		defer close(readerDone)
		runID := cfg.Attempt.RunID
		number := cfg.Attempt.Number
		for {
			select {
			case <-stop:
				return
			default:
				if cfg.Attempt.RunID != runID || cfg.Attempt.Number != number {
					panic("attempt identity changed")
				}
				runtime.Gosched()
			}
		}
	})

	t.Log("timeout identity reader stopped")
}
