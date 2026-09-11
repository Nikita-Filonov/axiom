package testtimeout

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_NoTimeout_DoesNotWrap(t *testing.T) {
	cfg := &axiom.Config{}

	Plugin()(cfg)

	assert.Empty(t, cfg.Runtime.TestWraps, "a non-positive timeout must be a no-op")
}

func TestPlugin_WithTimeout_AddsWrap(t *testing.T) {
	cfg := &axiom.Config{}

	Plugin(WithTimeout(time.Second))(cfg)

	assert.Len(t, cfg.Runtime.TestWraps, 1)
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

func TestRunWithTimeout_RePanicsBodyPanic(t *testing.T) {
	c := &axiom.Config{}

	assert.PanicsWithValue(t, "boom", func() {
		runWithTimeout(c, Config{Timeout: time.Second}, func(*axiom.Config) {
			panic("boom")
		})
	})
}

func TestRunWithTimeout_TimesOut(t *testing.T) {
	// c.T() == nil, so the timeout branch is exercised without failing anything.
	c := &axiom.Config{}

	release := make(chan struct{})
	defer close(release)

	runWithTimeout(c, Config{Timeout: 5 * time.Millisecond, Message: "custom deadline"}, func(*axiom.Config) {
		<-release
	})
}

func TestReportTimeout_DumpsGoroutineArtefact(t *testing.T) {
	var captured []axiom.Artefact
	c := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeArtefactSink(func(a axiom.Artefact) { captured = append(captured, a) }),
		),
	}

	reportTimeout(c, Config{Timeout: time.Second, DumpGoroutines: true})

	require.Len(t, captured, 1)
	assert.Equal(t, "testtimeout-goroutines.txt", captured[0].Name)
}

func TestReportTimeout_FailsThroughT(t *testing.T) {
	orphan := &testing.T{}

	reportTimeout(&axiom.Config{RootT: orphan}, Config{Timeout: time.Second})

	assert.True(t, orphan.Failed(), "the case must be marked failed on timeout")
}

func TestGoroutineDump_ContainsGoroutines(t *testing.T) {
	assert.Contains(t, goroutineDump(), "goroutine")
}
