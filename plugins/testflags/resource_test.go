package testflags_test

import (
	"sync"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testflags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEarlyReadCanRecoverAfterParse(t *testing.T) {
	fs := newSet(t)
	key := testflags.Int(testflags.WithName("custom"), testflags.WithDefault(7), testflags.WithFlagSet(fs))
	r := runnerFor(fs)
	flags, err := testflags.TryGet(r)
	require.Nil(t, flags)
	require.ErrorContains(t, err, "not been parsed")
	require.PanicsWithError(t, err.Error(), func() { testflags.Get(r) })
	value, err := key.TryGet(r)
	require.Zero(t, value)
	require.ErrorContains(t, err, "not been parsed")
	parse(t, fs, "-custom=55")
	require.Equal(t, 55, key.Get(r))
	snapshot := testflags.Get(r)
	require.Same(t, snapshot, testflags.Get(r))
}

func TestRunnerConfigurationValidation(t *testing.T) {
	require.PanicsWithValue(t, "testflags: nil flag source", func() { testflags.Resource(testflags.WithSource(nil)) })
	require.PanicsWithValue(t, "testflags: nil config option", func() { testflags.Resource(nil) })
	flags, err := testflags.TryGet(nil)
	require.Nil(t, flags)
	require.EqualError(t, err, "testflags: nil runner")
	require.PanicsWithError(t, "testflags: nil runner", func() { testflags.Get(nil) })
	_, err = testflags.TryGet(axiom.NewRunner())
	require.ErrorContains(t, err, "not found")
	require.PanicsWithError(t, err.Error(), func() { testflags.Get(axiom.NewRunner()) })
}

func TestFlagsAvailableToResourcesHooksFixturesAndPlugins(t *testing.T) {
	fs := newSet(t)
	key := testflags.Int(testflags.WithName("custom"), testflags.WithFlagSet(fs))
	var built, fixtures, hooks, plugins int
	resource := axiom.DefineResource("consumer", func(r *axiom.Runner) (int, func(), error) {
		built++
		return key.Get(r), nil, nil
	})
	fixture := axiom.DefineFixture("consumer", func(cfg *axiom.Config) (int, func(), error) {
		fixtures++
		return key.Get(cfg.Runner), nil, nil
	})
	r := axiom.NewRunner(
		axiom.WithRunnerResources(resource),
		axiom.WithRunnerFixtures(fixture),
		axiom.WithRunnerHooks(axiom.WithBeforeAll(func(r *axiom.Runner) {
			hooks++
			assert.Equal(t, 55, resource.Get(r))
			assert.Equal(t, 55, key.Get(r))
		})),
		axiom.WithRunnerResources(testflags.Resource(testflags.WithSource(fs))), // Hook order does not matter.
		axiom.WithRunnerPlugins(func(cfg *axiom.Config) {
			plugins++
			assert.Equal(t, 55, key.Get(cfg.Runner))
		}),
	)
	parse(t, fs, "-custom=55")
	// Resources can be resolved directly, before Runner.ApplyStart.
	require.Equal(t, 55, resource.Get(r))
	require.Equal(t, 0, hooks)
	for _, name := range []string{"first", "second"} {
		r.RunCase(t, axiom.NewCase(axiom.WithCaseName(name)), func(cfg *axiom.Config) {
			assert.Equal(t, 55, resource.Get(cfg.Runner))
			assert.Equal(t, 55, fixture.Get(cfg))
			assert.Equal(t, 55, key.Get(cfg.Runner))
		})
	}
	require.Equal(t, 1, built)
	require.Equal(t, 2, fixtures)
	require.Equal(t, 1, hooks)
	require.Equal(t, 4, plugins)
}

func TestConcurrentFirstRead(t *testing.T) {
	fs := newSet(t)
	key := testflags.Int(testflags.WithName("custom"), testflags.WithFlagSet(fs))
	parse(t, fs, "-custom=55")
	r := runnerFor(fs)
	const readers = 64
	snapshots := make(chan *testflags.Flags, readers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range readers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			assert.Equal(t, 55, key.Get(r))
			snapshot := testflags.Get(r)
			assert.Len(t, snapshot.All(), 1)
			snapshots <- snapshot
		}()
	}
	close(start)
	wg.Wait()
	close(snapshots)
	want := testflags.Get(r)
	for snapshot := range snapshots {
		require.Same(t, want, snapshot)
	}
}

func TestCopyJoinAndSourceIsolation(t *testing.T) {
	first := newSet(t)
	key := testflags.Int(testflags.WithName("custom"), testflags.WithDefault(1), testflags.WithFlagSet(first))
	second := newSet(t)
	testflags.Int(testflags.WithName("custom"), testflags.WithDefault(2), testflags.WithFlagSet(second))
	parse(t, first)
	parse(t, second)
	base := runnerFor(first)
	copyBeforeRead := base.Copy()
	joinedBeforeRead := base.Join(axiom.NewRunner())
	overlay := base.Join(runnerFor(second))
	require.Equal(t, 1, key.Get(base))
	require.Equal(t, 2, key.Get(overlay))
	copyAfterRead := base.Copy()
	joinedAfterRead := base.Join(axiom.NewRunner())
	require.Same(t, testflags.Get(base), testflags.Get(copyAfterRead))
	require.Same(t, testflags.Get(base), testflags.Get(joinedAfterRead))
	require.NoError(t, first.Set("custom", "3"))
	require.Equal(t, 3, key.Get(copyBeforeRead))
	require.Equal(t, 3, key.Get(joinedBeforeRead))
	require.Equal(t, 1, key.Get(base))
	require.Equal(t, 2, key.Get(overlay))
	// A cached resource survives Join even if an overlay replaces its definition.
	// This is the same rule as all Axiom resources.
	require.Equal(t, 1, key.Get(base.Join(runnerFor(second))))
	require.Equal(t, 2, key.Get(base.Join(overlay)))
	r := axiom.NewRunner(
		axiom.WithRunnerResources(testflags.Resource(testflags.WithSource(first))),
		axiom.WithRunnerResources(testflags.Resource(testflags.WithSource(second))),
	)
	require.Equal(t, 2, key.Get(r))
	r = axiom.NewRunner(axiom.WithRunnerResources(testflags.Resource(testflags.WithSource(first), testflags.WithSource(second))))
	require.Equal(t, 2, key.Get(r))
}
