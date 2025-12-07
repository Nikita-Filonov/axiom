package axiom

import (
	"testing"
	"time"
)

type Runner struct {
	Meta     Meta
	Skip     Skip
	Retry    Retry
	Hooks    Hooks
	Context  Context
	Plugins  []Plugin
	Parallel Parallel
	Fixtures Fixtures
}

type RunnerOption func(*Runner)

func NewRunner(options ...RunnerOption) Runner {
	r := Runner{}
	for _, option := range options {
		option(&r)
	}

	r.Meta.Normalize()
	r.Retry.Normalize()
	r.Context.Normalize()
	r.Fixtures.Normalize()

	return r
}

func WithRunnerMeta(options ...MetaOption) RunnerOption {
	return func(r *Runner) {
		m := NewMeta(options...)
		r.Meta = r.Meta.Join(m)
	}
}

func WithRunnerSkip(options ...SkipOption) RunnerOption {
	return func(r *Runner) {
		s := NewSkip(options...)
		r.Skip = r.Skip.Join(s)
	}
}

func WithRunnerRetry(options ...RetryOption) RunnerOption {
	return func(r *Runner) {
		rr := NewRetry(options...)
		r.Retry = r.Retry.Join(rr)
	}
}

func WithRunnerHooks(options ...HooksOption) RunnerOption {
	return func(r *Runner) {
		for _, option := range options {
			option(&r.Hooks)
		}
	}
}

func WithRunnerContext(options ...ContextOption) RunnerOption {
	return func(r *Runner) {
		c := NewContext(options...)
		r.Context = r.Context.Join(c)
	}
}

func WithRunnerPlugins(plugins ...Plugin) RunnerOption {
	return func(r *Runner) {
		r.Plugins = append(r.Plugins, plugins...)
	}
}

func WithRunnerParallel() RunnerOption {
	return func(r *Runner) {
		r.Parallel.Enabled = true
	}
}

func WithRunnerFixture(name string, fx Fixture) RunnerOption {
	return func(r *Runner) {
		if r.Fixtures.Registry == nil {
			r.Fixtures.Registry = map[string]Fixture{}
		}
		r.Fixtures.Registry[name] = fx
	}
}

func (r *Runner) Join(other Runner) Runner {
	return Runner{
		Meta:     r.Meta.Join(other.Meta),
		Skip:     r.Skip.Join(other.Skip),
		Retry:    r.Retry.Join(other.Retry),
		Hooks:    r.Hooks.Join(other.Hooks),
		Context:  r.Context.Join(other.Context),
		Plugins:  append(r.Plugins, other.Plugins...),
		Fixtures: r.Fixtures.Join(other.Fixtures),
		Parallel: r.Parallel.Join(other.Parallel),
	}
}

func (r *Runner) RunCase(t *testing.T, c Case, action TestAction) {
	cfg := r.BuildConfig(t, &c)
	r.ApplyPlugins(cfg)

	if cfg.Skip.Enabled {
		t.Skip(cfg.Skip.Reason)
	}

	if cfg.Parallel.Enabled {
		t.Parallel()
	}

	cfg.Hooks.ApplyBeforeTest(cfg)

	var isSucceed bool
	for attempt := 1; attempt <= cfg.Retry.Times; attempt++ {
		if attempt > 1 && cfg.Retry.Delay > 0 {
			time.Sleep(cfg.Retry.Delay)
		}

		t.Run(cfg.Name, func(st *testing.T) {
			cfg.SubT = st
			cfg.SubTest(action)
			isSucceed = !st.Failed()
		})

		if isSucceed {
			break
		}
	}

	cfg.Hooks.ApplyAfterTest(cfg)
}

func (r *Runner) BuildConfig(t *testing.T, c *Case) *Config {
	meta := r.Meta.Join(c.Meta)
	skip := r.Skip.Join(c.Skip)
	retry := r.Retry.Join(c.Retry)
	hooks := r.Hooks.Join(c.Hooks)
	context := r.Context.Join(c.Context)
	parallel := r.Parallel.Join(c.Parallel)
	fixtures := r.Fixtures.Join(c.Fixtures)

	cfg := &Config{
		ID:       c.ID,
		Name:     c.Name,
		Case:     c,
		Skip:     skip,
		Meta:     meta,
		Retry:    retry,
		Hooks:    hooks,
		RootT:    t,
		Params:   c.Params,
		Runner:   r,
		Context:  context,
		Parallel: parallel,
		Fixtures: fixtures,
	}

	cfg.Meta.Normalize()
	cfg.Retry.Normalize()
	cfg.Context.Normalize()
	cfg.Fixtures.Normalize()

	return cfg
}

func (r *Runner) ApplyPlugins(cfg *Config) {
	for _, p := range cfg.Runner.Plugins {
		p(cfg)
	}
	for _, p := range cfg.Case.Plugins {
		p(cfg)
	}
}
