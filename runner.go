package axiom

import (
	"sync"
	"sync/atomic"
	"testing"
)

// Runner holds configuration shared by the cases it executes. Meta, Skip,
// Retry, Hooks, Context, Runtime, Plugins, Parallel, and Fixtures provide
// settings merged with each Case; Resources belong to the runner itself.
// Its BeforeAll and AfterAll hooks, and its resource cleanups, run once for the
// runner. Use a [Suite] or [RunPackage] to give a runner an explicit group or
// package boundary. Without one, cleanup is bound to the first test that calls
// [Runner.RunCase].
// Configure a Runner before executing cases. Its lifecycle is one-shot: do not
// copy it after first use or run new cases after it has finished.
type Runner struct {
	beforeOnce sync.Once
	afterOnce  sync.Once

	managed atomic.Bool

	Meta      Meta
	Skip      Skip
	Retry     Retry
	Hooks     Hooks
	Context   Context
	Runtime   Runtime
	Plugins   []Plugin
	Parallel  Parallel
	Fixtures  Fixtures
	Resources Resources
}

// RunnerOption configures a Runner during [NewRunner].
type RunnerOption func(*Runner)

// NewRunner creates a Runner with the supplied options and normalizes its
// metadata, retry policy, context, fixtures, and resources.
func NewRunner(options ...RunnerOption) *Runner {
	r := &Runner{}
	for _, option := range options {
		option(r)
	}

	r.Meta.Normalize()
	r.Retry.Normalize()
	r.Context.Normalize()
	r.Fixtures.Normalize()
	r.Resources.Normalize()

	return r
}

// WithRunnerMeta merges metadata into the Runner defaults.
func WithRunnerMeta(options ...MetaOption) RunnerOption {
	return func(r *Runner) {
		m := NewMeta(options...)
		r.Meta = r.Meta.Join(m)
	}
}

// WithRunnerSkip merges skip settings into the Runner defaults.
func WithRunnerSkip(options ...SkipOption) RunnerOption {
	return func(r *Runner) {
		s := NewSkip(options...)
		r.Skip = r.Skip.Join(s)
	}
}

// WithRunnerRetry merges retry settings into the Runner defaults.
func WithRunnerRetry(options ...RetryOption) RunnerOption {
	return func(r *Runner) {
		rr := NewRetry(options...)
		r.Retry = r.Retry.Join(rr)
	}
}

// WithRunnerHooks appends lifecycle hooks to the Runner.
func WithRunnerHooks(options ...HooksOption) RunnerOption {
	return func(r *Runner) {
		for _, option := range options {
			option(&r.Hooks)
		}
	}
}

// WithRunnerContext merges execution contexts and named data into the Runner.
func WithRunnerContext(options ...ContextOption) RunnerOption {
	return func(r *Runner) {
		c := NewContext(options...)
		r.Context = r.Context.Join(c)
	}
}

// WithRunnerRuntime appends wrappers and sinks to the Runner runtime.
func WithRunnerRuntime(options ...RuntimeOption) RunnerOption {
	return func(r *Runner) {
		c := NewRuntime(options...)
		r.Runtime = r.Runtime.Join(c)
	}
}

// WithRunnerPlugins appends plugins applied before Case plugins.
func WithRunnerPlugins(plugins ...Plugin) RunnerOption {
	return func(r *Runner) {
		r.Plugins = append(r.Plugins, plugins...)
	}
}

// WithRunnerParallel sets the default parallel policy for Cases.
func WithRunnerParallel(options ...ParallelOption) RunnerOption {
	return func(r *Runner) {
		p := NewParallel(options...)
		r.Parallel = r.Parallel.Join(p)
	}
}

// WithRunnerFixture registers a named fixture definition available to Cases.
// Each attempt still constructs and cleans up its own value.
func WithRunnerFixture(name string, fx Fixture) RunnerOption {
	return func(r *Runner) {
		if r.Fixtures.Registry == nil {
			r.Fixtures.Registry = map[string]Fixture{}
		}
		r.Fixtures.Registry[name] = fx
	}
}

// WithRunnerFixtureKey registers a typed fixture definition. It panics for an
// invalid key or nil constructor.
func WithRunnerFixtureKey[T any](key FixtureKey[T], build TypedFixture[T]) RunnerOption {
	key.validate()
	if build == nil {
		panic("fixture: nil constructor")
	}

	return WithRunnerFixture(key.name, func(cfg *Config) (any, func(), error) {
		return build(cfg)
	})
}

// WithRunnerFixtures registers a batch of typed fixture definitions.
func WithRunnerFixtures(defs ...RunnerFixtureRegistrar) RunnerOption {
	return func(r *Runner) {
		for _, def := range defs {
			def.registerRunnerFixture(r)
		}
	}
}

// WithRunnerResource registers a named resource constructed on first access
// and shared for the Runner lifecycle.
func WithRunnerResource(name string, rs Resource) RunnerOption {
	return func(r *Runner) {
		if r.Resources.Registry == nil {
			r.Resources.Registry = map[string]Resource{}
		}
		r.Resources.Registry[name] = rs
	}
}

// WithRunnerResourceKey registers a typed resource definition. It panics for
// an invalid key or nil constructor.
func WithRunnerResourceKey[T any](key ResourceKey[T], build TypedResource[T]) RunnerOption {
	key.validate()
	if build == nil {
		panic("resource: nil constructor")
	}

	return WithRunnerResource(key.name, func(r *Runner) (any, func(), error) {
		return build(r)
	})
}

// WithRunnerResources registers a batch of typed resource definitions.
func WithRunnerResources(defs ...ResourceRegistrar) RunnerOption {
	return func(r *Runner) {
		for _, def := range defs {
			def.registerResource(r)
		}
	}
}

// Join returns a new Runner with other merged over r and fresh lifecycle guards.
// Explicit policy fields in other override r; hooks, plugins, and runtime
// handlers append in that order. Fixture definitions are merged with fresh
// attempt caches. Resource definitions, cached values, and cleanup callbacks
// are copied from both inputs, including resources initialized before Join.
// The returned Runner has its own finish lifecycle, so a copied cleanup runs
// when it finishes even if a source Runner also runs that cleanup.
func (r *Runner) Join(other *Runner) *Runner {
	return &Runner{
		Meta:      r.Meta.Join(other.Meta),
		Skip:      r.Skip.Join(other.Skip),
		Retry:     r.Retry.Join(other.Retry),
		Hooks:     r.Hooks.Join(other.Hooks),
		Context:   r.Context.Join(other.Context),
		Runtime:   r.Runtime.Join(other.Runtime),
		Plugins:   append(r.Plugins, other.Plugins...),
		Fixtures:  r.Fixtures.Join(other.Fixtures),
		Parallel:  r.Parallel.Join(other.Parallel),
		Resources: r.Resources.Join(other.Resources),
	}
}

// RunCase executes c as a subtest of t, passing a fresh [Config] to action for
// each attempt. It starts the runner once and, unless an outer package boundary
// owns the runner, registers runner cleanup on t. When the same Runner is used
// with multiple top-level tests, [RunPackage] should own its lifecycle.
func (r *Runner) RunCase(t *testing.T, c Case, action TestAction) {
	r.ApplyStart()
	if !r.managed.Load() {
		t.Cleanup(r.ApplyFinish)
	}

	r.runCase(t, c, action)
}

func (r *Runner) runCase(t *testing.T, c Case, action TestAction) {
	execution := newCaseExecution(r, t, c, action)
	execution.run()
}

// BuildConfig merges runner and case settings into a Config. It does not run
// plugins or the test action. It panics if t or c is nil.
func (r *Runner) BuildConfig(t *testing.T, c *Case) *Config {
	if t == nil {
		panic("config: nil *testing.T")
	}
	if c == nil {
		panic("config: nil *Case")
	}

	meta := r.Meta.Join(c.Meta)
	skip := r.Skip.Join(c.Skip)
	retry := r.Retry.Join(c.Retry)
	hooks := r.Hooks.Join(c.Hooks)
	context := r.Context.Join(c.Context)
	runtime := r.Runtime.Join(c.Runtime)
	parallel := r.Parallel.Join(c.Parallel)
	fixtures := r.Fixtures.Join(c.Fixtures)

	cfg := &Config{
		Case:     c,
		Skip:     skip,
		Meta:     meta,
		Retry:    retry,
		Hooks:    hooks,
		RootT:    t,
		Runner:   r,
		Context:  context,
		Runtime:  runtime,
		Parallel: parallel,
		Fixtures: fixtures,
	}

	cfg.Meta.Normalize()
	cfg.Retry.Normalize()
	cfg.Context.Normalize()
	cfg.Fixtures.Normalize()

	return cfg
}

// ApplyStart invokes BeforeAll hooks at most once for this runner.
func (r *Runner) ApplyStart() {
	r.beforeOnce.Do(func() {
		r.Runtime.Event(NewEvent(EventTypeRunnerBeforeAllStart))
		defer func() {
			if v := recover(); v != nil {
				r.Runtime.Event(NewEvent(EventTypeRunnerBeforeAllPanic, WithEventMessage(v)))
				panic(v)
			}

			r.Runtime.Event(NewEvent(EventTypeRunnerBeforeAllFinish))
		}()

		r.Hooks.ApplyBeforeAll(r)
	})
}

// ApplyFinish invokes AfterAll hooks and then resource cleanups at most once
// for this runner. Resource cleanups run in reverse construction order.
func (r *Runner) ApplyFinish() {
	r.afterOnce.Do(func() {
		r.Runtime.Event(NewEvent(EventTypeRunnerAfterAllStart))
		defer func() {
			if v := recover(); v != nil {
				r.Runtime.Event(NewEvent(EventTypeRunnerAfterAllPanic, WithEventMessage(v)))
				panic(v)
			}

			r.Runtime.Event(NewEvent(EventTypeRunnerAfterAllFinish))
		}()

		defer r.Resources.Teardown(r)
		r.Hooks.ApplyAfterAll(r)
	})
}
