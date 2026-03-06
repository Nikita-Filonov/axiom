package axiom_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewResources_Defaults(t *testing.T) {
	r := axiom.NewResources()

	assert.Nil(t, r.Registry)
	assert.Nil(t, r.Cache)
}

func TestWithResource(t *testing.T) {
	r := axiom.NewResources(
		axiom.WithResource("x", func(r *axiom.Runner) (any, func(), error) {
			return 1, nil, nil
		}),
	)

	assert.Contains(t, r.Registry, "x")
}

func TestWithResourcesMap(t *testing.T) {
	m := map[string]axiom.Resource{
		"a": func(r *axiom.Runner) (any, func(), error) { return "A", nil, nil },
		"b": func(r *axiom.Runner) (any, func(), error) { return "B", nil, nil },
	}

	r := axiom.NewResources(
		axiom.WithResourcesMap(m),
	)

	assert.Len(t, r.Registry, 2)
}

func TestResourcesJoin(t *testing.T) {
	r1 := axiom.NewResources(
		axiom.WithResource("a", func(r *axiom.Runner) (any, func(), error) {
			return "A1", nil, nil
		}),
		axiom.WithResource("b", func(r *axiom.Runner) (any, func(), error) {
			return "B1", nil, nil
		}),
	)

	r2 := axiom.NewResources(
		axiom.WithResource("b", func(r *axiom.Runner) (any, func(), error) {
			return "B2", nil, nil // override
		}),
		axiom.WithResource("c", func(r *axiom.Runner) (any, func(), error) {
			return "C", nil, nil
		}),
	)

	result := r1.Join(r2)
	assert.Empty(t, result.Cache, "cache must be empty right after Join")

	runner := axiom.NewRunner()
	runner.Resources = result

	a := axiom.MustResource[string](runner, "a")
	b := axiom.MustResource[string](runner, "b")
	c := axiom.MustResource[string](runner, "c")

	assert.Equal(t, "A1", a)
	assert.Equal(t, "B2", b)
	assert.Equal(t, "C", c)

	assert.Len(t, runner.Resources.Cache, 3)
}

func TestGetResource_HappyPath(t *testing.T) {
	calls := 0
	cleanupCalled := false

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("num", func(r *axiom.Runner) (any, func(), error) {
			calls++
			return 42, func() { cleanupCalled = true }, nil
		}),
	)

	v := axiom.MustResource[int](runner, "num")
	assert.Equal(t, 42, v)
	assert.Equal(t, 1, calls)

	v2 := axiom.MustResource[int](runner, "num")
	assert.Equal(t, 42, v2)
	assert.Equal(t, 1, calls, "resource must be created only once")

	assert.Len(t, runner.Hooks.AfterAll, 1)

	runner.Hooks.AfterAll[0](runner)
	assert.True(t, cleanupCalled)
}

func TestGetResource_Dependency_NoDeadlock(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerResource("a", func(r *axiom.Runner) (any, func(), error) {
			return "A", nil, nil
		}),
		axiom.WithRunnerResource("b", func(r *axiom.Runner) (any, func(), error) {
			a := axiom.MustResource[string](r, "a")
			return a + "B", nil, nil
		}),
	)

	v := axiom.MustResource[string](runner, "b")
	assert.Equal(t, "AB", v)
}

func TestGetResource_ConcurrentAccess(t *testing.T) {
	calls := 0
	cleanups := 0

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			calls++
			return "X", func() { cleanups++ }, nil
		}),
	)

	const workers = 10
	done := make(chan struct{}, workers)

	for i := 0; i < workers; i++ {
		go func() {
			_ = axiom.MustResource[string](runner, "x")
			done <- struct{}{}
		}()
	}

	for i := 0; i < workers; i++ {
		<-done
	}

	assert.Len(t, runner.Resources.Cache, 1)

	runner.Hooks.ApplyAfterAll(runner)
	assert.Equal(t, 1, cleanups)
	assert.GreaterOrEqual(t, calls, 1)
}

func TestUseResources(t *testing.T) {
	calls := map[string]int{}

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("a", func(r *axiom.Runner) (any, func(), error) {
			calls["a"]++
			return "A", nil, nil
		}),
		axiom.WithRunnerResource("b", func(r *axiom.Runner) (any, func(), error) {
			calls["b"]++
			return "B", nil, nil
		}),
	)

	hook := axiom.UseResources("a", "b")
	hook(runner)

	assert.Equal(t, 1, calls["a"])
	assert.Equal(t, 1, calls["b"])
	assert.Contains(t, runner.Resources.Cache, "a")
	assert.Contains(t, runner.Resources.Cache, "b")
}

func TestGetResource_NotFound(t *testing.T) {
	runner := axiom.NewRunner()

	_, err := axiom.GetResource[int](runner, "missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetResource_WrongType(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			return "string", nil, nil
		}),
	)

	_, err := axiom.GetResource[int](runner, "x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected type")
}

func TestGetResource_FactoryError(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			return nil, nil, fmt.Errorf("boom")
		}),
	)

	_, err := axiom.GetResource[int](runner, "x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestGetResource_CleanupRegisteredOnce(t *testing.T) {
	cleanups := 0

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			return "X", func() { cleanups++ }, nil
		}),
	)

	_ = axiom.MustResource[string](runner, "x")
	_ = axiom.MustResource[string](runner, "x")

	assert.Len(t, runner.Hooks.AfterAll, 1)

	for _, hook := range runner.Hooks.AfterAll {
		hook(runner)
	}

	assert.Equal(t, 1, cleanups)
}

func TestResourcesCopy_DeepCopyMaps(t *testing.T) {
	r := axiom.Resources{
		Registry: map[string]axiom.Resource{
			"x": func(rr *axiom.Runner) (any, func(), error) { return 1, nil, nil },
		},
		Cache: map[string]axiom.ResourceResult{
			"x": {Value: 1},
		},
	}

	cp := r.Copy()

	v, ok := cp.Cache["x"]
	assert.True(t, ok)
	assert.Equal(t, 1, v.Value)

	cp.Registry["y"] = func(rr *axiom.Runner) (any, func(), error) { return 2, nil, nil }
	cp.Cache["y"] = axiom.ResourceResult{Value: 2}

	x := cp.Cache["x"]
	x.Value = 100
	cp.Cache["x"] = x

	assert.NotContains(t, r.Registry, "y")
	assert.NotContains(t, r.Cache, "y")
	assert.Equal(t, 1, r.Cache["x"].Value)
}

func TestResourcesCopy_DeepCopyRegistryAndCache(t *testing.T) {
	r := axiom.Resources{
		Registry: map[string]axiom.Resource{
			"a": func(rr *axiom.Runner) (any, func(), error) { return "A", nil, nil },
		},
		Cache: map[string]axiom.ResourceResult{
			"cached": {Value: "C"},
		},
	}

	cp := r.Copy()

	assert.Contains(t, cp.Registry, "a")
	assert.Contains(t, cp.Cache, "cached")
	assert.Equal(t, "C", cp.Cache["cached"].Value)

	cp.Registry["b"] = func(rr *axiom.Runner) (any, func(), error) { return "B", nil, nil }
	cp.Cache["cached2"] = axiom.ResourceResult{Value: "X"}

	assert.NotContains(t, r.Registry, "b")
	assert.NotContains(t, r.Cache, "cached2")
}

func TestResourcesJoin_MergesRegistryAndCache(t *testing.T) {
	r1 := axiom.Resources{
		Registry: map[string]axiom.Resource{
			"a": func(rr *axiom.Runner) (any, func(), error) { return "A1", nil, nil },
			"b": func(rr *axiom.Runner) (any, func(), error) { return "B1", nil, nil },
		},
		Cache: map[string]axiom.ResourceResult{
			"x": {Value: "X1"},
			"y": {Value: "Y1"},
		},
	}

	r2 := axiom.Resources{
		Registry: map[string]axiom.Resource{
			"b": func(rr *axiom.Runner) (any, func(), error) { return "B2", nil, nil }, // override
			"c": func(rr *axiom.Runner) (any, func(), error) { return "C2", nil, nil },
		},
		Cache: map[string]axiom.ResourceResult{
			"y": {Value: "Y2"}, // override
			"z": {Value: "Z2"},
		},
	}

	joined := r1.Join(r2)

	assert.Contains(t, joined.Registry, "a")
	assert.Contains(t, joined.Registry, "b")
	assert.Contains(t, joined.Registry, "c")

	assert.Equal(t, "X1", joined.Cache["x"].Value)
	assert.Equal(t, "Y2", joined.Cache["y"].Value)
	assert.Equal(t, "Z2", joined.Cache["z"].Value)
}

func TestResourcesJoin_DoesNotMutateSources(t *testing.T) {
	r1 := axiom.Resources{
		Registry: map[string]axiom.Resource{
			"a": func(rr *axiom.Runner) (any, func(), error) { return "A1", nil, nil },
		},
		Cache: map[string]axiom.ResourceResult{
			"x": {Value: "X1"},
		},
	}
	r2 := axiom.Resources{
		Registry: map[string]axiom.Resource{
			"b": func(rr *axiom.Runner) (any, func(), error) { return "B2", nil, nil },
		},
		Cache: map[string]axiom.ResourceResult{
			"y": {Value: "Y2"},
		},
	}

	joined := r1.Join(r2)
	joined.Registry["c"] = func(rr *axiom.Runner) (any, func(), error) { return "C3", nil, nil }
	joined.Cache["z"] = axiom.ResourceResult{Value: "Z3"}

	assert.NotContains(t, r1.Registry, "c")
	assert.NotContains(t, r2.Registry, "c")
	assert.NotContains(t, r1.Cache, "z")
	assert.NotContains(t, r2.Cache, "z")
}

func TestGetResource_UsesPrewarmedCacheWithoutFactoryCall(t *testing.T) {
	calls := 0

	r := axiom.NewRunner(
		axiom.WithRunnerResource("x", func(rr *axiom.Runner) (any, func(), error) {
			calls++
			return "from-factory", nil, nil
		}),
	)
	r.Resources.Cache["x"] = axiom.ResourceResult{Value: "from-cache"}

	v, err := axiom.GetResource[string](r, "x")
	assert.NoError(t, err)
	assert.Equal(t, "from-cache", v)
	assert.Equal(t, 0, calls, "factory must not run when cache already has value")
}

func TestGetResource_JoinedCacheOverrideVisibleViaAPI(t *testing.T) {
	r1 := axiom.Resources{
		Registry: map[string]axiom.Resource{
			"x": func(rr *axiom.Runner) (any, func(), error) { return "A", nil, nil },
		},
		Cache: map[string]axiom.ResourceResult{
			"x": {Value: "A-cached"},
		},
	}
	r2 := axiom.Resources{
		Registry: map[string]axiom.Resource{
			"x": func(rr *axiom.Runner) (any, func(), error) { return "B", nil, nil },
		},
		Cache: map[string]axiom.ResourceResult{
			"x": {Value: "B-cached"},
		},
	}

	joined := r1.Join(r2)
	runner := axiom.NewRunner()
	runner.Resources = joined

	v, err := axiom.GetResource[string](runner, "x")
	assert.NoError(t, err)
	assert.Equal(t, "B-cached", v, "other cache value must override base during join")
}
