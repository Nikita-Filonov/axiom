package teststats_test

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/teststats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_RecordsAttemptAfterItsCleanups(t *testing.T) {
	stats := teststats.NewStats()
	var start, fixtureEnd, cleanupEnd time.Time
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(teststats.Plugin(stats)))
	c := axiom.NewCase(
		axiom.WithCaseID("C-1"),
		axiom.WithCaseName("case"),
		axiom.WithCaseFixture("value", func(*axiom.Config) (any, func(), error) {
			return 1, func() { fixtureEnd = time.Now() }, nil
		}),
	)

	runner.RunCase(t, c, func(cfg *axiom.Config) {
		start = time.Now()
		axiom.GetFixture[int](cfg, "value")
		cfg.T().Cleanup(func() {
			assert.Empty(t, stats.Attempts(), "the attempt is recorded after its cleanups")
			cleanupEnd = time.Now()
		})
	})

	attempts := stats.Attempts()
	require.Len(t, attempts, 1)
	a := attempts[0]
	assert.NotEmpty(t, a.RunID)
	assert.Equal(t, 1, a.Number)
	assert.Equal(t, "C-1", a.CaseID)
	assert.Equal(t, "case", a.Name)
	assert.Equal(t, t.Name()+"/case", a.TestName)
	assert.Equal(t, teststats.StatusPassed, a.Status)
	assert.Empty(t, a.Error)
	assert.False(t, a.Start.After(start))
	assert.False(t, a.End.Before(fixtureEnd))
	assert.False(t, a.End.Before(cleanupEnd))
	assert.Equal(t, a.End.Sub(a.Start), a.Duration)
}

func TestPlugin_ParallelTimingAndRepeatedNames(t *testing.T) {
	stats := teststats.NewStats()
	var released time.Time
	t.Run("group", func(t *testing.T) {
		runner := axiom.NewRunner(
			axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
			axiom.WithRunnerPlugins(teststats.Plugin(stats)),
		)
		for range 3 {
			runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("same")), func(*axiom.Config) {})
		}
		assert.Empty(t, stats.Attempts())
		released = time.Now()
	})

	attempts := stats.Attempts()
	require.Len(t, attempts, 3)
	for _, a := range attempts {
		assert.False(t, a.Start.Before(released), "start excludes the wait for parallel scheduling")
	}
	assert.Len(t, stats.Runs(), 3, "every RunCase call is its own run")
}

func TestPlugin_PolicySkipIsOneSkippedAttempt(t *testing.T) {
	for _, parallel := range []bool{false, true} {
		for _, retries := range []int{1, 3} {
			stats := teststats.NewStats()
			t.Run("scope", func(t *testing.T) {
				runner := axiom.NewRunner(
					axiom.WithRunnerRetry(axiom.WithRetryTimes(retries)),
					axiom.WithRunnerPlugins(teststats.Plugin(stats)),
				)
				if parallel {
					runner.Parallel = axiom.NewParallel(axiom.WithParallelEnabled())
				}
				runner.RunCase(t, axiom.NewCase(axiom.WithCaseSkip(axiom.SkipBecause("maintenance"))), func(*axiom.Config) {
					t.Error("skipped body ran")
				})
			})

			attempts := stats.Attempts()
			require.Len(t, attempts, 1, "parallel=%v retries=%d", parallel, retries)
			assert.Equal(t, teststats.StatusSkipped, attempts[0].Status)
			assert.Equal(t, "maintenance", attempts[0].SkipReason)
			assert.Equal(t, 1, attempts[0].Number)
			assert.Equal(t, teststats.StatusSkipped, stats.Runs()[0].Status)
		}
	}
}

func TestPlugin_RuntimeSkip(t *testing.T) {
	stats := teststats.NewStats()
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(teststats.Plugin(stats)))

	runner.RunCase(t, axiom.NewCase(), func(cfg *axiom.Config) { cfg.T().Skip("runtime skip") })

	attempts := stats.Attempts()
	require.Len(t, attempts, 1)
	assert.Equal(t, teststats.StatusSkipped, attempts[0].Status)
	assert.Empty(t, attempts[0].SkipReason)
}

func TestPlugin_PlanningConfigRecordsNothingAndMetaIncludesLaterPlugins(t *testing.T) {
	stats := teststats.NewStats()
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(teststats.Plugin(stats)))
	c := axiom.NewCase(axiom.WithCasePlugins(func(cfg *axiom.Config) { cfg.Meta.Owner = "team" }))

	runner.RunCase(t, c, func(*axiom.Config) {})

	attempts := stats.Attempts()
	require.Len(t, attempts, 1, "the planning Config has the plugin installed too")
	assert.Equal(t, "team", attempts[0].Meta.Owner)
	assert.Panics(t, func() { teststats.Plugin(nil) })
}

func TestPlugin_ConfigsOutsideRunCaseAreSeparateRuns(t *testing.T) {
	stats := teststats.NewStats()
	for recorded := range 2 {
		t.Run("direct", func(t *testing.T) {
			cfg := &axiom.Config{RootT: t, Case: &axiom.Case{}}
			teststats.Plugin(stats)(cfg)
			cfg.Event(axiom.NewEvent(axiom.EventTypeCaseStart))
			assert.Len(t, stats.Attempts(), recorded, "recorded when the test finishes")
		})
	}

	runs := stats.Runs()
	require.Len(t, runs, 2)
	for _, run := range runs {
		assert.Empty(t, run.ID)
		assert.Equal(t, 1, run.Attempts[0].Number)
	}
}

func TestPlugin_FirstLifecycleFailureFailsTheAttempt(t *testing.T) {
	stats := teststats.NewStats()
	t.Run("attempt", func(t *testing.T) {
		cfg := &axiom.Config{RootT: t, Case: &axiom.Case{}}
		teststats.Plugin(stats)(cfg)
		cfg.Event(axiom.NewEvent(axiom.EventTypeFixtureSetupFailed, axiom.WithEventMessage("before start")))
		cfg.Event(axiom.NewEvent(axiom.EventTypeCaseStart))
		cfg.Event(axiom.NewEvent(axiom.EventTypeCaseStart))
		cfg.Event(axiom.NewEvent(axiom.EventTypeFixtureCleanupPanic, axiom.WithEventMessage("cleanup")))
		cfg.Event(axiom.NewEvent(axiom.EventTypeStepPanic, axiom.WithEventMessage("step")))
	})

	attempts := stats.Attempts()
	require.Len(t, attempts, 1)
	assert.Equal(t, teststats.StatusFailed, attempts[0].Status, "a panic can unwind before Go marks the test failed")
	assert.Equal(t, "cleanup", attempts[0].Error)
}

func TestPlugin_DuplicateInstallationAndIndependentCollectors(t *testing.T) {
	one, two := teststats.NewStats(), teststats.NewStats()
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(teststats.Plugin(one), teststats.Plugin(two)))
	c := axiom.NewCase(axiom.WithCasePlugins(teststats.Plugin(one)))

	runner.RunCase(t, c, func(*axiom.Config) {})

	require.Len(t, one.Attempts(), 1)
	require.Len(t, two.Attempts(), 1)
	assert.Equal(t, one.Attempts()[0].RunID, two.Attempts()[0].RunID)
}
