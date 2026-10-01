package axiom

// Fixture constructs a value on first access through [GetFixture] for one test
// attempt. A successful construction may return a cleanup function; Axiom runs
// it after AfterTest hooks, even when later test work fails. A construction
// error fails the current subtest and does not register cleanup.
type Fixture func(cfg *Config) (any, func(), error)

// Fixtures stores definitions and attempt scoped values. Runner and Case
// definitions are merged into a fresh registry and cache for each attempt.
type Fixtures struct {
	Registry map[string]Fixture

	cache    map[string]any
	cleanups []func(*Config)
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

// Copy returns Fixtures with independent maps and cleanup list.
func (f *Fixtures) Copy() Fixtures {
	result := Fixtures{}

	if f.Registry != nil {
		result.Registry = make(map[string]Fixture, len(f.Registry))
		for k, v := range f.Registry {
			result.Registry[k] = v
		}
	}
	if f.cache != nil {
		result.cache = make(map[string]any, len(f.cache))
		for k, v := range f.cache {
			result.cache[k] = v
		}
	}
	if f.cleanups != nil {
		result.cleanups = append([]func(*Config){}, f.cleanups...)
	}
	return result
}

// Join merges definitions from other into a new registry with an empty
// attempt cache and cleanup list.
func (f *Fixtures) Join(other Fixtures) Fixtures {
	result := f.Copy()

	if result.Registry == nil {
		result.Registry = map[string]Fixture{}
	}
	for k, v := range other.Registry {
		result.Registry[k] = v
	}
	result.cache = map[string]any{}
	result.cleanups = nil

	return result
}

// Normalize initializes missing registry and cache maps.
func (f *Fixtures) Normalize() {
	if f.Registry == nil {
		f.Registry = map[string]Fixture{}
	}
	if f.cache == nil {
		f.cache = map[string]any{}
	}
}

// teardown runs registered cleanups in reverse order.
func (f *Fixtures) teardown(cfg *Config) {
	for i := len(f.cleanups) - 1; i >= 0; i-- {
		f.cleanups[i](cfg)
	}
	f.cleanups = nil
}

// GetFixture returns the named value, constructing and caching it on first use
// in cfg. A missing fixture, construction error, or type mismatch fails the
// current subtest. It panics if cfg is nil.
func GetFixture[T any](cfg *Config, name string) T {
	var zero T

	if cfg == nil {
		panic("fixture: nil config")
	}

	cfg.Fixtures.Normalize()

	if cached, ok := cfg.Fixtures.cache[name]; ok {
		out, ok := cached.(T)
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
		cfg.Fixtures.cleanups = append(cfg.Fixtures.cleanups, fixtureCleanupHook(name, cleanup))
	}

	out, ok := val.(T)
	if !ok {
		cfg.Event(NewEvent(EventTypeFixtureSetupFailed, WithEventName(name), WithEventMessage("unexpected type")))
		cfg.SubT.Fatalf("fixture %q has unexpected type", name)
		return zero
	}
	cfg.Fixtures.cache[name] = val
	cfg.Event(NewEvent(EventTypeFixtureSetupFinish, WithEventName(name)))

	return out
}

// UseFixtures returns a callback that initializes named fixtures for an attempt.
func UseFixtures(names ...string) func(cfg *Config) {
	return func(cfg *Config) {
		for _, name := range names {
			GetFixture[any](cfg, name)
		}
	}
}

func fixtureCleanupHook(name string, cleanup func()) func(*Config) {
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
