package axiom

// Fixture constructs a value on first access through [GetFixture] for one test
// attempt. A successful construction may return a cleanup function; Axiom runs
// it after AfterTest hooks, even when later test work fails. A construction
// error fails the current subtest and does not register cleanup.
type Fixture func(cfg *Config) (any, func(), error)

// FixtureResult holds a constructed value and its optional cleanup in a
// Config's fixture cache.
type FixtureResult struct {
	Value   any
	Cleanup func()
}

// FixtureCleanup is an attempt cleanup callback run in reverse setup order.
type FixtureCleanup func(*Config)

// Fixtures stores definitions and attempt scoped values. Runner and Case
// definitions are merged into a fresh registry and cache for each attempt.
type Fixtures struct {
	Registry map[string]Fixture
	Cache    map[string]FixtureResult
	Cleanups []FixtureCleanup
}

// FixturesOption configures a Fixtures registry.
type FixturesOption func(*Fixtures)

// NewFixtures creates a fixture registry with the supplied options.
func NewFixtures(options ...FixturesOption) Fixtures {
	f := Fixtures{}
	for _, option := range options {
		option(&f)
	}

	return f
}

// WithFixture registers a named fixture definition in Fixtures.
func WithFixture(name string, fixture Fixture) FixturesOption {
	return func(f *Fixtures) {
		if f.Registry == nil {
			f.Registry = map[string]Fixture{}
		}
		f.Registry[name] = fixture
	}
}

// WithFixturesMap copies named definitions into Fixtures, replacing matching
// names without taking ownership of the input map.
func WithFixturesMap(fixtures map[string]Fixture) FixturesOption {
	return func(f *Fixtures) {
		if f.Registry == nil {
			f.Registry = map[string]Fixture{}
		}
		for k, v := range fixtures {
			f.Registry[k] = v
		}
	}
}

func (f *Fixtures) Copy() Fixtures {
	result := Fixtures{}

	if f.Registry != nil {
		result.Registry = make(map[string]Fixture, len(f.Registry))
		for k, v := range f.Registry {
			result.Registry[k] = v
		}
	}
	if f.Cache != nil {
		result.Cache = make(map[string]FixtureResult, len(f.Cache))
		for k, v := range f.Cache {
			result.Cache[k] = v
		}
	}
	if f.Cleanups != nil {
		result.Cleanups = append([]FixtureCleanup{}, f.Cleanups...)
	}
	return result
}

func (f *Fixtures) Join(other Fixtures) Fixtures {
	result := f.Copy()

	if result.Registry == nil {
		result.Registry = map[string]Fixture{}
	}
	for k, v := range other.Registry {
		result.Registry[k] = v
	}
	result.Cache = map[string]FixtureResult{}
	result.Cleanups = nil

	return result
}

func (f *Fixtures) Normalize() {
	if f.Registry == nil {
		f.Registry = map[string]Fixture{}
	}
	if f.Cache == nil {
		f.Cache = map[string]FixtureResult{}
	}
}

func (f *Fixtures) Teardown(cfg *Config) {
	for i := len(f.Cleanups) - 1; i >= 0; i-- {
		f.Cleanups[i](cfg)
	}
	f.Cleanups = nil
}

// GetFixture returns the named value, constructing and caching it on first use
// in cfg. A missing fixture, construction error, or type mismatch fails the
// current subtest. It panics if cfg is nil.
func GetFixture[T any](cfg *Config, name string) T {
	var zero T

	if cfg == nil {
		panic("fixture: nil config")
	}

	if res, ok := cfg.Fixtures.Cache[name]; ok {
		out, ok := res.Value.(T)
		if !ok {
			cfg.Event(NewEvent(EventTypeFixtureSetupFailed, WithEventName(name), WithEventMessage("unexpected type")))
			cfg.SubT.Fatalf("fixture %q has unexpected type", name)
			return zero
		}
		return out
	}

	fx, ok := cfg.Fixtures.Registry[name]
	if !ok {
		cfg.Event(NewEvent(EventTypeFixtureSetupFailed, WithEventName(name), WithEventMessage("not found")))
		cfg.SubT.Fatalf("fixture %q not found", name)
		return zero
	}
	if fx == nil {
		cfg.Event(NewEvent(EventTypeFixtureSetupFailed, WithEventName(name), WithEventMessage("nil fixture")))
		cfg.SubT.Fatalf("fixture %q is nil", name)
		return zero
	}

	cfg.Event(NewEvent(EventTypeFixtureSetupStart, WithEventName(name)))
	val, cleanup, err := fx(cfg)
	if err != nil {
		cfg.Event(NewEvent(EventTypeFixtureSetupFailed, WithEventName(name), WithEventMessage(err.Error())))
		cfg.SubT.Fatalf("fixture %q failed: %v", name, err)
		return zero
	}

	if cleanup != nil {
		cfg.Fixtures.Cleanups = append(cfg.Fixtures.Cleanups, fixtureCleanupHook(name, cleanup))
	}

	out, ok := val.(T)
	if !ok {
		cfg.Event(NewEvent(EventTypeFixtureSetupFailed, WithEventName(name), WithEventMessage("unexpected type")))
		cfg.SubT.Fatalf("fixture %q has unexpected type", name)
		return zero
	}
	cfg.Fixtures.Cache[name] = FixtureResult{Value: val, Cleanup: cleanup}
	cfg.Event(NewEvent(EventTypeFixtureSetupFinish, WithEventName(name)))

	return out
}

func UseFixtures(names ...string) func(cfg *Config) {
	return func(cfg *Config) {
		for _, name := range names {
			GetFixture[any](cfg, name)
		}
	}
}

func fixtureCleanupHook(name string, cleanup func()) FixtureCleanup {
	return func(c *Config) {
		c.Event(NewEvent(EventTypeFixtureCleanupStart, WithEventName(name)))
		defer func() {
			if v := recover(); v != nil {
				c.Event(NewEvent(EventTypeFixtureCleanupPanic, WithEventName(name), WithEventMessage(v)))
				panic(v)
			}

			c.Event(NewEvent(EventTypeFixtureCleanupFinish, WithEventName(name)))
		}()

		cleanup()
	}
}
