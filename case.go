package axiom

type Case struct {
	ID       string
	Name     string
	Skip     Skip
	Meta     Meta
	Retry    Retry
	Hooks    Hooks
	Params   any
	Context  Context
	Plugins  []Plugin
	Parallel Parallel
	Fixtures Fixtures
}

type CaseOption func(*Case)

func NewCase(options ...CaseOption) Case {
	c := Case{}
	for _, option := range options {
		option(&c)
	}

	return c
}

func WithCaseID(id string) CaseOption {
	return func(c *Case) { c.ID = id }
}

func WithCaseName(name string) CaseOption {
	return func(c *Case) { c.Name = name }
}

func WithCaseSkip(opts ...SkipOption) CaseOption {
	return func(c *Case) {
		s := NewSkip(opts...)
		c.Skip = c.Skip.Join(s)
	}
}

func WithCaseMeta(opts ...MetaOption) CaseOption {
	return func(c *Case) {
		m := NewMeta(opts...)
		c.Meta = c.Meta.Join(m)
	}
}

func WithCaseRetry(opts ...RetryOption) CaseOption {
	return func(c *Case) {
		r := NewRetry(opts...)
		c.Retry = c.Retry.Join(r)
	}
}

func WithCaseParams(params any) CaseOption {
	return func(c *Case) { c.Params = params }
}

func WithCaseContext(opts ...ContextOption) CaseOption {
	return func(c *Case) {
		ctx := NewContext(opts...)
		c.Context = c.Context.Join(ctx)
	}
}

func WithCasePlugins(plugins ...Plugin) CaseOption {
	return func(c *Case) {
		c.Plugins = append(c.Plugins, plugins...)
	}
}

func WithCaseParallel() CaseOption {
	return func(c *Case) {
		c.Parallel.Enabled = true
	}
}

func WithCaseSequential() CaseOption {
	return func(c *Case) {
		c.Parallel.Enabled = false
	}
}

func WithCaseFixture(name string, fx Fixture) CaseOption {
	return func(c *Case) {
		if c.Fixtures.Registry == nil {
			c.Fixtures.Registry = map[string]Fixture{}
		}
		c.Fixtures.Registry[name] = fx
	}
}
