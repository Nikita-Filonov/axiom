package testenv_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testenv"
	"github.com/stretchr/testify/require"
)

func TestResourceIsLazyAndStable(t *testing.T) {
	var calls int
	values := []string{"CUSTOM=first", "EMPTY=", "IGNORED", "CUSTOM=second"}
	r := axiom.NewRunner(axiom.WithRunnerResources(testenv.Resource(testenv.WithSource(func() []string {
		calls++
		return values
	}))))
	require.Zero(t, calls)
	envs := testenv.Get(r)
	require.Equal(t, 1, calls)
	require.Same(t, envs, testenv.Get(r))
	value, ok := envs.Lookup("CUSTOM")
	require.True(t, ok)
	require.Equal(t, "second", value)
	value, ok = envs.Lookup("EMPTY")
	require.True(t, ok)
	require.Empty(t, value)
	_, ok = envs.Lookup("MISSING")
	require.False(t, ok)
	_, ok = envs.Lookup("IGNORED")
	require.False(t, ok)
	values = []string{"CUSTOM=changed"}
	require.Equal(t, "second", testenv.String(testenv.WithName("CUSTOM")).Get(r))
	require.Equal(t, 1, calls)
}

func TestEmptySnapshotAndCaseSensitiveNames(t *testing.T) {
	empty := testenv.Get(runnerWith())
	_, ok := empty.Lookup("CUSTOM")
	require.False(t, ok)

	r := runnerWith("CUSTOM=upper", "custom=lower")
	require.Equal(t, "upper", testenv.String(testenv.WithName("CUSTOM")).Get(r))
	require.Equal(t, "lower", testenv.String(testenv.WithName("custom")).Get(r))
}

func TestDefaultOSSourceCapturesOnFirstRead(t *testing.T) {
	const name = "AXIOM_TESTENV_SNAPSHOT_TEST"
	t.Setenv(name, "first")
	r := axiom.NewRunner(axiom.WithRunnerResources(testenv.Resource()))
	t.Setenv(name, "second")
	key := testenv.String(testenv.WithName(name))
	require.Equal(t, "second", key.Get(r))
	t.Setenv(name, "third")
	require.Equal(t, "second", key.Get(r))
}

func TestSnapshotAvailableThroughRunnerAndConfig(t *testing.T) {
	key := testenv.Int(testenv.WithName("COUNT"))
	resource := axiom.DefineResource("consumer", func(r *axiom.Runner) (int, func(), error) {
		return key.Get(r), nil, nil
	})
	fixture := axiom.DefineFixture("consumer", func(cfg *axiom.Config) (int, func(), error) {
		return key.Get(cfg.Runner), nil, nil
	})
	var pluginCalls int
	r := axiom.NewRunner(
		axiom.WithRunnerResources(testenv.Resource(testenv.WithSource(func() []string { return []string{"COUNT=55"} })), resource),
		axiom.WithRunnerFixtures(fixture),
		axiom.WithRunnerHooks(axiom.WithBeforeAll(func(r *axiom.Runner) { require.Equal(t, 55, key.Get(r)) })),
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			pluginCalls++
			require.Equal(t, 55, key.Get(cfg.Runner))
		}),
	)
	r.RunCase(t, axiom.NewCase(), func(cfg *axiom.Config) {
		require.Equal(t, 55, resource.Get(cfg.Runner))
		require.Equal(t, 55, fixture.Get(cfg))
		require.Equal(t, 55, key.Get(cfg.Runner))
	})
	require.Equal(t, 2, pluginCalls) // Planning and attempt Configs.
}

func TestConcurrentFirstReadAndRunnerCopies(t *testing.T) {
	var calls atomic.Int32
	source := func() []string {
		calls.Add(1)
		return []string{"VALUE=first"}
	}
	r := axiom.NewRunner(axiom.WithRunnerResources(testenv.Resource(testenv.WithSource(source))))
	before := r.Copy()
	const readers = 32
	var wg sync.WaitGroup
	for range readers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value := testenv.String(testenv.WithName("VALUE")).Get(r)
			if value != "first" {
				t.Errorf("value = %q", value)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, int32(1), calls.Load())
	after := r.Copy()
	require.Same(t, testenv.Get(r), testenv.Get(after))
	require.NotSame(t, testenv.Get(r), testenv.Get(before))
	require.Equal(t, int32(2), calls.Load())
}

func TestResourceValidation(t *testing.T) {
	require.PanicsWithValue(t, "testenv: nil config option", func() { testenv.Resource(nil) })
	require.PanicsWithValue(t, "testenv: nil environment source", func() { testenv.Resource(testenv.WithSource(nil)) })
	_, err := testenv.TryGet(nil)
	require.EqualError(t, err, "testenv: nil runner")
	require.PanicsWithError(t, "testenv: nil runner", func() { testenv.Get(nil) })
	_, err = testenv.TryGet(axiom.NewRunner())
	require.ErrorContains(t, err, "not found")
}
