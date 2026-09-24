package axiom

// Case describes one logical test without containing its test action. A
// [Runner] merges its settings with the Case to build a [Config] for each
// attempt. Each attempt receives a copy of the Case, including on retries.
type Case struct {
	ID          string
	Name        string
	Skip        Skip
	Meta        Meta
	Retry       Retry
	Hooks       Hooks
	Params      any
	Context     Context
	Runtime     Runtime
	Plugins     []Plugin
	Parallel    Parallel
	Fixtures    Fixtures
	Description string
}

// CaseOption configures a Case during [NewCase].
type CaseOption func(*Case)

// NewCase returns a declarative Case with the supplied options. The test action
// is passed separately to [Runner.RunCase] or [Suite.RunCase].
func NewCase(options ...CaseOption) Case {
	c := Case{}
	for _, option := range options {
		option(&c)
	}

	return c
}

// WithCaseID sets the case identifier used by reporters and integrations.
func WithCaseID(id string) CaseOption {
	return func(c *Case) { c.ID = id }
}

// WithCaseName sets the name used for the case's Go subtest.
func WithCaseName(name string) CaseOption {
	return func(c *Case) { c.Name = name }
}

// WithCaseSkip merges skip settings into the Case. Explicit Case settings can
// override Runner skip settings during execution.
func WithCaseSkip(opts ...SkipOption) CaseOption {
	return func(c *Case) {
		s := NewSkip(opts...)
		c.Skip = c.Skip.Join(s)
	}
}

// WithCaseMeta merges metadata into the Case.
func WithCaseMeta(opts ...MetaOption) CaseOption {
	return func(c *Case) {
		m := NewMeta(opts...)
		c.Meta = c.Meta.Join(m)
	}
}

// WithCaseRetry merges retry settings into the Case. Explicit Case fields
// override corresponding Runner fields.
func WithCaseRetry(opts ...RetryOption) CaseOption {
	return func(c *Case) {
		r := NewRetry(opts...)
		c.Retry = c.Retry.Join(r)
	}
}

// WithCaseHooks appends test and step hooks to the Case. Runner hooks run first.
// BeforeAll and AfterAll hooks are runner scoped and have no case-level effect.
func WithCaseHooks(options ...HooksOption) CaseOption {
	return func(c *Case) {
		for _, option := range options {
			option(&c.Hooks)
		}
	}
}

// WithCaseParams stores parameters read through [GetParams] during execution.
func WithCaseParams(params any) CaseOption {
	return func(c *Case) { c.Params = params }
}

// WithCaseContext merges execution contexts and named data into the Case.
func WithCaseContext(opts ...ContextOption) CaseOption {
	return func(c *Case) {
		ctx := NewContext(opts...)
		c.Context = c.Context.Join(ctx)
	}
}

// WithCaseRuntime appends runtime wrappers and sinks to the Case.
func WithCaseRuntime(opts ...RuntimeOption) CaseOption {
	return func(c *Case) {
		r := NewRuntime(opts...)
		c.Runtime = c.Runtime.Join(r)
	}
}

// WithCasePlugins appends plugins that run after Runner plugins.
func WithCasePlugins(plugins ...Plugin) CaseOption {
	return func(c *Case) { c.Plugins = append(c.Plugins, plugins...) }
}

// WithCaseParallel sets the Case's parallel policy, including an explicit
// override of the Runner policy.
func WithCaseParallel(opts ...ParallelOption) CaseOption {
	return func(c *Case) {
		p := NewParallel(opts...)
		c.Parallel = c.Parallel.Join(p)
	}
}

// WithCaseFixture registers a named fixture for this Case. It overrides a
// Runner fixture with the same name for this case's attempts.
func WithCaseFixture(name string, fx Fixture) CaseOption {
	return func(c *Case) {
		if c.Fixtures.Registry == nil {
			c.Fixtures.Registry = map[string]Fixture{}
		}
		c.Fixtures.Registry[name] = fx
	}
}

// WithCaseFixtureKey registers a typed fixture under key. It panics for an
// invalid key or nil constructor.
func WithCaseFixtureKey[T any](key FixtureKey[T], build TypedFixture[T]) CaseOption {
	key.validate()
	if build == nil {
		panic("fixture: nil constructor")
	}

	return WithCaseFixture(key.name, func(cfg *Config) (any, func(), error) {
		return build(cfg)
	})
}

// WithCaseFixtures registers a batch of typed fixture definitions on the Case.
func WithCaseFixtures(fixtures ...CaseFixtureRegistrar) CaseOption {
	return func(c *Case) {
		for _, fixture := range fixtures {
			fixture.registerCaseFixture(c)
		}
	}
}

// WithCaseDescription sets descriptive text for the Case.
func WithCaseDescription(desc string) CaseOption {
	return func(c *Case) { c.Description = desc }
}

// Copy returns a Case with independent copies of its configuration containers.
// Values stored in Params and other user supplied values are not deep copied.
func (c Case) Copy() Case {
	result := Case{
		ID:          c.ID,
		Name:        c.Name,
		Skip:        c.Skip.Copy(),
		Meta:        c.Meta.Copy(),
		Retry:       c.Retry.Copy(),
		Hooks:       c.Hooks.Copy(),
		Params:      c.Params,
		Context:     c.Context.Copy(),
		Runtime:     c.Runtime.Copy(),
		Parallel:    c.Parallel.Copy(),
		Fixtures:    c.Fixtures.Copy(),
		Description: c.Description,
	}

	if c.Plugins == nil {
		result.Plugins = nil
	} else {
		result.Plugins = append([]Plugin{}, c.Plugins...)
	}

	return result
}
