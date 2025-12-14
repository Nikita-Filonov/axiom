package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewCase_Defaults(t *testing.T) {
	c := axiom.NewCase()

	assert.Empty(t, c.ID)
	assert.Empty(t, c.Name)
	assert.Empty(t, c.Params)
	assert.Empty(t, c.Plugins)
	assert.False(t, c.Parallel.Enabled)
	assert.Empty(t, c.Fixtures.Registry)
}

func TestWithCaseID(t *testing.T) {
	c := axiom.NewCase(axiom.WithCaseID("123"))

	assert.Equal(t, "123", c.ID)
}

func TestWithCaseName(t *testing.T) {
	c := axiom.NewCase(axiom.WithCaseName("my test"))

	assert.Equal(t, "my test", c.Name)
}

func TestWithCaseSkip(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseSkip(axiom.WithSkipReason("first")),
		axiom.WithCaseSkip(axiom.WithSkipEnabled(true)),
	)

	assert.True(t, c.Skip.Enabled)
	assert.Equal(t, "first", c.Skip.Reason)
}

func TestWithCaseMeta(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseMeta(axiom.WithMetaEpic("A")),
		axiom.WithCaseMeta(axiom.WithMetaStory("S")),
	)

	assert.Equal(t, "A", c.Meta.Epic)
	assert.Equal(t, "S", c.Meta.Story)
}

func TestWithCaseRetry(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseRetry(axiom.WithRetryTimes(5)),
		axiom.WithCaseRetry(axiom.WithRetryDelay(10)),
	)

	assert.Equal(t, 5, c.Retry.Times)
	assert.Equal(t, 10, int(c.Retry.Delay))
}

func TestWithCaseParams(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseParams(map[string]any{"u": 1}),
	)

	p := c.Params.(map[string]any)

	assert.Equal(t, 1, p["u"])
}

func TestWithCaseContext(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseContext(axiom.WithContextData("a", 1)),
		axiom.WithCaseContext(axiom.WithContextData("b", 2)),
	)

	assert.Equal(t, 1, c.Context.Data["a"])
	assert.Equal(t, 2, c.Context.Data["b"])
}

func TestWithCasePlugins(t *testing.T) {
	p1 := func(cfg *axiom.Config) {}
	p2 := func(cfg *axiom.Config) {}

	c := axiom.NewCase(
		axiom.WithCasePlugins(p1, p2),
	)

	assert.Equal(t, 2, len(c.Plugins))
}

func TestWithCaseParallel(t *testing.T) {
	c := axiom.NewCase(axiom.WithCaseParallel())

	assert.True(t, c.Parallel.Enabled)
}

func TestWithCaseSequential(t *testing.T) {
	c := axiom.NewCase(axiom.WithCaseParallel(), axiom.WithCaseSequential())

	assert.False(t, c.Parallel.Enabled)
}

func TestWithCaseFixture(t *testing.T) {
	fx := func(cfg *axiom.Config) (any, func(), error) {
		return 100, nil, nil
	}

	c := axiom.NewCase(
		axiom.WithCaseFixture("user", fx),
	)

	assert.NotNil(t, c.Fixtures.Registry)
	assert.Contains(t, c.Fixtures.Registry, "user")
}

func TestWithCaseRuntime(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseRuntime(func(rt *axiom.Runtime) {
			rt.EmitLogSink(func(l axiom.Log) {})
		}),
	)

	assert.Len(t, c.Runtime.LogSinks, 1)
}

func TestWithCaseRuntime_MultipleOptions(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseRuntime(func(rt *axiom.Runtime) {
			rt.EmitLogSink(func(l axiom.Log) {})
		}),
		axiom.WithCaseRuntime(func(rt *axiom.Runtime) {
			rt.EmitArtefactSink(func(a axiom.Artefact) {})
		}),
	)

	assert.Len(t, c.Runtime.LogSinks, 1)
	assert.Len(t, c.Runtime.ArtefactSinks, 1)
}

func TestCaseRuntime_AppliedToConfig(t *testing.T) {
	r := axiom.NewRunner()

	c := axiom.NewCase(
		axiom.WithCaseRuntime(func(rt *axiom.Runtime) {
			rt.EmitLogSink(func(l axiom.Log) {})
		}),
	)

	cfg := r.BuildConfig(&testing.T{}, &c)

	assert.Len(t, cfg.Runtime.LogSinks, 1)
}

func TestCaseRuntime_JoinWithRunnerRuntime(t *testing.T) {
	r := axiom.NewRunner(
		axiom.WithRunnerRuntime(func(rt *axiom.Runtime) {
			rt.EmitLogSink(func(l axiom.Log) {})
		}),
	)

	c := axiom.NewCase(
		axiom.WithCaseRuntime(func(rt *axiom.Runtime) {
			rt.EmitArtefactSink(func(a axiom.Artefact) {})
		}),
	)

	cfg := r.BuildConfig(&testing.T{}, &c)

	assert.Len(t, cfg.Runtime.LogSinks, 1)
	assert.Len(t, cfg.Runtime.ArtefactSinks, 1)
}

func TestCaseRuntime_UsedDuringRun(t *testing.T) {
	var logCalled bool
	var artefactCalled bool

	r := axiom.NewRunner()

	c := axiom.NewCase(
		axiom.WithCaseName("runtime"),
		axiom.WithCaseRuntime(func(rt *axiom.Runtime) {
			rt.EmitLogSink(func(l axiom.Log) {
				logCalled = true
			})
			rt.EmitArtefactSink(func(a axiom.Artefact) {
				artefactCalled = true
			})
		}),
	)

	r.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Log(axiom.Log{Text: "hello"})
		cfg.Artefact(axiom.Artefact{Name: "file"})
	})

	assert.True(t, logCalled)
	assert.True(t, artefactCalled)
}

func TestCaseRuntime_IsolatedBetweenCases(t *testing.T) {
	var count int

	r := axiom.NewRunner()

	c1 := axiom.NewCase(
		axiom.WithCaseName("A"),
		axiom.WithCaseRuntime(func(rt *axiom.Runtime) {
			rt.EmitLogSink(func(l axiom.Log) {
				count++
			})
		}),
	)

	c2 := axiom.NewCase(
		axiom.WithCaseName("B"),
	)

	r.RunCase(t, c1, func(cfg *axiom.Config) {
		cfg.Log(axiom.Log{})
	})

	r.RunCase(t, c2, func(cfg *axiom.Config) {})

	assert.Equal(t, 1, count)
}
