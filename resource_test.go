package axiom_test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

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
	var events []axiom.Event

	runner := axiom.NewRunner(
		axiom.WithRunnerRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
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

	assert.Empty(t, runner.Hooks.AfterAll, "cleanups must not pollute user AfterAll hooks")
	assert.Len(t, runner.Resources.Cleanups, 1)
	requireEventTypes(t, events,
		axiom.EventTypeResourceSetupStart,
		axiom.EventTypeResourceSetupFinish,
	)

	runner.Resources.Teardown(runner)
	assert.True(t, cleanupCalled)
	assert.Empty(t, runner.Resources.Cleanups, "cleanups must be drained")
	requireEventTypes(t, events,
		axiom.EventTypeResourceSetupStart,
		axiom.EventTypeResourceSetupFinish,
		axiom.EventTypeResourceCleanupStart,
		axiom.EventTypeResourceCleanupFinish,
	)
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

	runner.Resources.Teardown(runner)
	assert.Equal(t, 1, cleanups)
	assert.Equal(t, 1, calls, "constructor must run exactly once under concurrent access")
}

func TestGetResource_ConstructorRunsExactlyOnceAcrossConcurrentRacers(t *testing.T) {
	var calls int32
	start := make(chan struct{})

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			atomic.AddInt32(&calls, 1)
			time.Sleep(20 * time.Millisecond)
			return "X", nil, nil
		}),
	)

	const workers = 50
	done := make(chan struct{}, workers)

	for i := 0; i < workers; i++ {
		go func() {
			<-start
			_ = axiom.MustResource[string](runner, "x")
			done <- struct{}{}
		}()
	}

	close(start)
	for i := 0; i < workers; i++ {
		<-done
	}

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
	assert.Len(t, runner.Resources.Cache, 1)
}

func TestGetResource_ConstructorErrorIsCachedAndReturnedToAllCallers(t *testing.T) {
	var calls int32

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			atomic.AddInt32(&calls, 1)
			return nil, nil, fmt.Errorf("boom")
		}),
	)

	_, err1 := axiom.GetResource[string](runner, "x")
	_, err2 := axiom.GetResource[string](runner, "x")

	assert.Error(t, err1)
	assert.Error(t, err2)
	assert.Contains(t, err1.Error(), "boom")
	assert.Contains(t, err2.Error(), "boom")
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "failed constructor must not be retried")
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
	var events []axiom.Event
	runner := axiom.NewRunner(
		axiom.WithRunnerRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
	)

	_, err := axiom.GetResource[int](runner, "missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	requireEventTypes(t, events, axiom.EventTypeResourceSetupFailed)
	assert.Equal(t, "not found", events[0].Message)
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

func TestGetResource_WrongType_RegistersCleanupAndCachesValue(t *testing.T) {
	// On type mismatch the resource itself was constructed successfully — the
	// failure is purely on the caller's expected T. The lifecycle events reflect
	// a successful setup, the value is cached for callers with the matching
	// type, and the cleanup is registered exactly once on AfterAll.
	var events []axiom.Event
	var cleanupCalled bool

	runner := axiom.NewRunner(
		axiom.WithRunnerRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			return "string", func() { cleanupCalled = true }, nil
		}),
	)

	_, err := axiom.GetResource[int](runner, "x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected type")

	requireEventTypes(t, events,
		axiom.EventTypeResourceSetupStart,
		axiom.EventTypeResourceSetupFinish,
	)

	value, err := axiom.GetResource[string](runner, "x")
	assert.NoError(t, err)
	assert.Equal(t, "string", value)

	assert.Len(t, runner.Resources.Cleanups, 1)
	runner.Resources.Teardown(runner)
	assert.True(t, cleanupCalled)
}

func TestGetResource_FactoryError(t *testing.T) {
	var events []axiom.Event
	runner := axiom.NewRunner(
		axiom.WithRunnerRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			return nil, nil, fmt.Errorf("boom")
		}),
	)

	_, err := axiom.GetResource[int](runner, "x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
	requireEventTypes(t, events,
		axiom.EventTypeResourceSetupStart,
		axiom.EventTypeResourceSetupFailed,
	)
	assert.Equal(t, "boom", events[1].Message)
}

func TestGetResource_CleanupPanic_EmitsPanicFact(t *testing.T) {
	var events []axiom.Event
	runner := axiom.NewRunner(
		axiom.WithRunnerRuntime(
			axiom.WithRuntimeEventSink(func(e axiom.Event) {
				events = append(events, e)
			}),
		),
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			return "X", func() { panic("boom") }, nil
		}),
	)

	assert.Equal(t, "X", axiom.MustResource[string](runner, "x"))

	assert.PanicsWithValue(t, "boom", func() {
		runner.Resources.Teardown(runner)
	})

	requireEventTypes(t, events,
		axiom.EventTypeResourceSetupStart,
		axiom.EventTypeResourceSetupFinish,
		axiom.EventTypeResourceCleanupStart,
		axiom.EventTypeResourceCleanupPanic,
	)
	assert.Equal(t, "boom", events[3].Message)
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

	assert.Len(t, runner.Resources.Cleanups, 1)

	runner.Resources.Teardown(runner)

	assert.Equal(t, 1, cleanups)
}

func TestResourcesTeardown_LIFOOrder(t *testing.T) {
	var order []string

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("db", func(r *axiom.Runner) (any, func(), error) {
			return "db", func() { order = append(order, "db") }, nil
		}),
		axiom.WithRunnerResource("client", func(r *axiom.Runner) (any, func(), error) {
			_ = axiom.MustResource[string](r, "db")
			return "client", func() { order = append(order, "client") }, nil
		}),
		axiom.WithRunnerResource("session", func(r *axiom.Runner) (any, func(), error) {
			_ = axiom.MustResource[string](r, "client")
			return "session", func() { order = append(order, "session") }, nil
		}),
	)

	_ = axiom.MustResource[string](runner, "session")

	runner.Resources.Teardown(runner)
	assert.Equal(t, []string{"session", "client", "db"}, order,
		"resource cleanups must run in reverse setup order")
}

func TestResourcesTeardown_RunsAfterUserAfterAllHooks(t *testing.T) {
	var order []string

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("db", func(r *axiom.Runner) (any, func(), error) {
			return "db", func() { order = append(order, "resource-cleanup") }, nil
		}),
		axiom.WithRunnerHooks(
			axiom.WithAfterAll(func(r *axiom.Runner) {
				_ = axiom.MustResource[string](r, "db")
				order = append(order, "user-after-all")
			}),
		),
	)

	_ = axiom.MustResource[string](runner, "db")

	runner.ApplyFinish()

	assert.Equal(t, []string{"user-after-all", "resource-cleanup"}, order,
		"user AfterAll hooks must observe live resources before cleanups run")
}

func TestResourcesTeardown_RunsWhenAfterAllPanics(t *testing.T) {
	var order []string

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("db", func(r *axiom.Runner) (any, func(), error) {
			return "db", func() { order = append(order, "resource-cleanup") }, nil
		}),
		axiom.WithRunnerHooks(
			axiom.WithAfterAll(func(r *axiom.Runner) {
				order = append(order, "after")
				panic("after boom")
			}),
		),
	)

	_ = axiom.MustResource[string](runner, "db")

	assert.PanicsWithValue(t, "after boom", runner.ApplyFinish)
	assert.Equal(t, []string{"after", "resource-cleanup"}, order,
		"resource cleanup must still run after a panicking AfterAll hook")
	assert.Empty(t, runner.Resources.Cleanups,
		"resource cleanups must be drained even when AfterAll panics")
}

func TestResourcesTeardown_IntegratedThroughApplyFinish(t *testing.T) {
	var order []string

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("a", func(r *axiom.Runner) (any, func(), error) {
			return "a", func() { order = append(order, "a") }, nil
		}),
		axiom.WithRunnerResource("b", func(r *axiom.Runner) (any, func(), error) {
			_ = axiom.MustResource[string](r, "a")
			return "b", func() { order = append(order, "b") }, nil
		}),
	)

	_ = axiom.MustResource[string](runner, "b")

	runner.ApplyFinish()
	assert.Equal(t, []string{"b", "a"}, order)

	runner.ApplyFinish()
	assert.Equal(t, []string{"b", "a"}, order, "ApplyFinish must be idempotent via sync.Once")
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

func TestGetResource_FactoryError_DoesNotRegisterCleanup(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			return nil, func() { t.Fatal("cleanup must not be registered on factory error") }, fmt.Errorf("boom")
		}),
	)

	_, err := axiom.GetResource[int](runner, "x")
	assert.Error(t, err)

	assert.Empty(t, runner.Resources.Cleanups,
		"cleanup must not be registered when constructor returned an error")
	assert.NotContains(t, runner.Resources.Cache, "x",
		"value must not be cached when constructor returned an error")

	runner.Resources.Teardown(runner)
}

func TestGetResource_NilCleanup_DoesNotRegisterAnything(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			return "X", nil, nil
		}),
	)

	v := axiom.MustResource[string](runner, "x")
	assert.Equal(t, "X", v)

	assert.Empty(t, runner.Resources.Cleanups,
		"nil cleanup must not be appended to the cleanup stack")
	assert.Contains(t, runner.Resources.Cache, "x",
		"value must still be cached even with nil cleanup")
}

func TestGetResource_CachedWrongType_ReturnsError(t *testing.T) {
	calls := 0
	runner := axiom.NewRunner(
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			calls++
			return "X", nil, nil
		}),
	)

	first, err := axiom.GetResource[string](runner, "x")
	assert.NoError(t, err)
	assert.Equal(t, "X", first)

	_, err = axiom.GetResource[int](runner, "x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected type")

	assert.Equal(t, 1, calls, "cached lookup with wrong type must not re-run the factory")
}

func TestGetResource_ConcurrentWaiters_AllSeeSameValue(t *testing.T) {
	var calls int32
	start := make(chan struct{})

	runner := axiom.NewRunner(
		axiom.WithRunnerResource("x", func(r *axiom.Runner) (any, func(), error) {
			atomic.AddInt32(&calls, 1)
			time.Sleep(20 * time.Millisecond)
			return &struct{ v int }{v: 42}, nil, nil
		}),
	)

	type ptrT = *struct{ v int }
	const workers = 30
	results := make(chan ptrT, workers)

	for i := 0; i < workers; i++ {
		go func() {
			<-start
			results <- axiom.MustResource[ptrT](runner, "x")
		}()
	}
	close(start)

	first := <-results
	assert.NotNil(t, first)
	for i := 1; i < workers; i++ {
		got := <-results
		assert.Same(t, first, got,
			"every concurrent waiter must observe the same pointer produced by sync.Once")
	}

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls),
		"constructor must run exactly once under heavy concurrency")
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
