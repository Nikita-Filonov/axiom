package axiom_test

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewRunner_Defaults(t *testing.T) {
	r := axiom.NewRunner()

	// Meta defaults
	assert.NotNil(t, r.Meta.Labels)
	assert.NotNil(t, r.Meta.Tags)

	// Skip defaults
	assert.False(t, r.Skip.Enabled)
	assert.Equal(t, "", r.Skip.Reason)

	// Retry defaults
	assert.Equal(t, 3, r.Retry.Times)
	assert.Equal(t, time.Second*2, r.Retry.Delay)

	// Context defaults
	assert.NotNil(t, r.Context.Raw)
	assert.NotNil(t, r.Context.Data)

	// Fixtures
	assert.NotNil(t, r.Fixtures.Registry)
	assert.NotNil(t, r.Fixtures.Cache)
}

func TestWithRunnerMeta(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerMeta(
			axiom.WithMetaEpic("EPIC"),
			axiom.WithMetaStory("STORY"),
		),
	)

	assert.Equal(t, "EPIC", r.Meta.Epic)
	assert.Equal(t, "STORY", r.Meta.Story)
}

func TestWithRunnerSkip(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerSkip(axiom.WithSkipEnabled(true)),
		axiom.WithRunnerSkip(axiom.WithSkipReason("cause")),
	)

	assert.True(t, r.Skip.Enabled)
	assert.Equal(t, "cause", r.Skip.Reason)
}

func TestWithRunnerRetry(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerRetry(axiom.WithRetryTimes(10)),
		axiom.WithRunnerRetry(axiom.WithRetryDelay(5*time.Second)),
	)

	assert.Equal(t, 10, r.Retry.Times)
	assert.Equal(t, 5*time.Second, r.Retry.Delay)
}

func TestWithRunnerContext(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerContext(axiom.WithContextData("a", 1)),
	)

	assert.Equal(t, 1, r.Context.Data["a"])
}

func TestWithRunnerPlugins(t *testing.T) {
	p1 := func(cfg *axiom.Config) {}
	p2 := func(cfg *axiom.Config) {}

	r := axiom.NewRunner(
		axiom.WithRunnerPlugins(p1, p2),
	)

	assert.Len(t, r.Plugins, 2)
}

func TestWithRunnerParallel(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerParallel(),
	)

	assert.True(t, r.Parallel.Enabled)
}

func TestWithRunnerFixture(t *testing.T) {
	fx := func(cfg *axiom.Config) (any, func(), error) { return 123, nil, nil }

	r := axiom.NewRunner(
		axiom.WithRunnerFixture("num", fx),
	)

	assert.Contains(t, r.Fixtures.Registry, "num")
}

func TestRunnerJoin(t *testing.T) {
	r1 := axiom.NewRunner(
		axiom.WithRunnerMeta(axiom.WithMetaEpic("A")),
		axiom.WithRunnerSkip(axiom.WithSkipReason("r1")),
		axiom.WithRunnerRetry(axiom.WithRetryTimes(3)),
	)

	r2 := axiom.NewRunner(
		axiom.WithRunnerMeta(axiom.WithMetaStory("B")),
		axiom.WithRunnerSkip(axiom.WithSkipEnabled(true)),
		axiom.WithRunnerRetry(axiom.WithRetryDelay(7*time.Second)),
	)

	result := r1.Join(r2)

	assert.Equal(t, "A", result.Meta.Epic)
	assert.Equal(t, "B", result.Meta.Story)

	assert.True(t, result.Skip.Enabled)
	assert.Equal(t, "r1", result.Skip.Reason)

	assert.Equal(t, 3, result.Retry.Times)
	assert.Equal(t, 7*time.Second, result.Retry.Delay)
}

func TestRunnerBuildConfig(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerMeta(axiom.WithMetaEpic("RunnerEpic")),
		axiom.WithRunnerSkip(axiom.WithSkipReason("runner skip")),
		axiom.WithRunnerRetry(axiom.WithRetryTimes(10)),
		axiom.WithRunnerParallel(),
		axiom.WithRunnerContext(axiom.WithContextData("x", 1)),
	)

	c := axiom.NewCase(
		axiom.WithCaseID("CASE-ID"),
		axiom.WithCaseName("CaseName"),
		axiom.WithCaseMeta(axiom.WithMetaStory("Story")),
		axiom.WithCaseSkip(axiom.WithSkipEnabled(true)),
		axiom.WithCaseRetry(axiom.WithRetryDelay(7)),
		axiom.WithCaseContext(axiom.WithContextData("y", 2)),
	)

	cfg := r.BuildConfig(&testing.T{}, &c)

	assert.Equal(t, "CASE-ID", cfg.ID)
	assert.Equal(t, "CaseName", cfg.Name)

	// Meta merge
	assert.Equal(t, "RunnerEpic", cfg.Meta.Epic)
	assert.Equal(t, "Story", cfg.Meta.Story)

	// Skip merge
	assert.True(t, cfg.Skip.Enabled)
	assert.Equal(t, "runner skip", cfg.Skip.Reason)

	// Retry merge
	assert.Equal(t, 10, cfg.Retry.Times)
	assert.Equal(t, 7, int(cfg.Retry.Delay))

	// Context merge
	assert.Equal(t, 1, cfg.Context.Data["x"])
	assert.Equal(t, 2, cfg.Context.Data["y"])

	assert.True(t, cfg.Parallel.Enabled)

	assert.Equal(t, &c, cfg.Case)
	assert.Equal(t, r, cfg.Runner)
	assert.NotNil(t, cfg.RootT)
}

func TestRunnerApplyPlugins(t *testing.T) {
	var calls []string

	r := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			func(cfg *axiom.Config) { calls = append(calls, "runner1") },
			func(cfg *axiom.Config) { calls = append(calls, "runner2") },
		),
	)

	c := axiom.NewCase(
		axiom.WithCasePlugins(
			func(cfg *axiom.Config) { calls = append(calls, "case1") },
			func(cfg *axiom.Config) { calls = append(calls, "case2") },
		),
	)

	cfg := &axiom.Config{
		Runner: r,
		Case:   &c,
	}

	cfg.ApplyPlugins()

	assert.Equal(t,
		[]string{"runner1", "runner2", "case1", "case2"},
		calls,
	)
}

func TestRunner_BuildConfigInsideRun(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerMeta(axiom.WithMetaEpic("EPIC")),
		axiom.WithRunnerRetry(axiom.WithRetryTimes(1)),
	)

	c := axiom.NewCase(
		axiom.WithCaseName("MyCase"),
		axiom.WithCaseMeta(axiom.WithMetaStory("STORY")),
	)

	called := false

	r.RunCase(t, c, func(cfg *axiom.Config) {
		called = true

		assert.Equal(t, "MyCase", cfg.Name)
		assert.Equal(t, "EPIC", cfg.Meta.Epic)
		assert.Equal(t, "STORY", cfg.Meta.Story)

		assert.Equal(t, 1, cfg.Retry.Times)
		assert.Equal(t, c, *cfg.Case)
	})

	assert.True(t, called)
}

func TestRunner_MultipleCases(t *testing.T) {
	r := axiom.NewRunner()

	tests := []axiom.Case{
		axiom.NewCase(axiom.WithCaseName("A")),
		axiom.NewCase(axiom.WithCaseName("B")),
		axiom.NewCase(axiom.WithCaseName("C")),
	}

	var visited []string

	for _, tc := range tests {
		r.RunCase(t, tc, func(cfg *axiom.Config) {
			visited = append(visited, cfg.Name)
		})
	}

	assert.Equal(t, []string{"A", "B", "C"}, visited)
}

func TestRunner_MergeMetaDuringRun(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerMeta(axiom.WithMetaEpic("GLOBAL")),
	)

	c := axiom.NewCase(
		axiom.WithCaseMeta(axiom.WithMetaFeature("CASE")),
	)

	r.RunCase(t, c, func(cfg *axiom.Config) {
		assert.Equal(t, "GLOBAL", cfg.Meta.Epic)
		assert.Equal(t, "CASE", cfg.Meta.Feature)
	})
}

func TestRunner_FixturesInsideRun(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerFixture("num", func(cfg *axiom.Config) (any, func(), error) {
			return 42, nil, nil
		}),
	)

	c := axiom.NewCase()

	r.RunCase(t, c, func(cfg *axiom.Config) {
		v := axiom.GetFixture[int](cfg, "num")
		assert.Equal(t, 42, v)
	})
}

func TestRunner_BeforeAll_AfterAll_CalledOnce(t *testing.T) {
	var beforeCount, afterCount int

	r := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeAll(func(r *axiom.Runner) { beforeCount++ }),
			axiom.WithAfterAll(func(r *axiom.Runner) { afterCount++ }),
		),
	)

	c := axiom.NewCase(axiom.WithCaseName("dummy"))

	r.RunCase(t, c, func(cfg *axiom.Config) {})
	r.RunCase(t, c, func(cfg *axiom.Config) {})
	r.RunCase(t, c, func(cfg *axiom.Config) {})

	assert.Equal(t, 1, beforeCount, "BeforeAll should run once")
}

func TestRunner_BeforeAll_ExecutesBeforeTestLogic(t *testing.T) {
	var order []string

	r := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeAll(func(r *axiom.Runner) { order = append(order, "before") }),
		),
	)

	c := axiom.NewCase(axiom.WithCaseName("test"))

	r.RunCase(t, c, func(cfg *axiom.Config) {
		order = append(order, "action")
	})

	assert.Equal(t, "before", order[0])
	assert.Equal(t, "action", order[1])
}
