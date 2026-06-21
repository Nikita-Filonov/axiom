package axiom

type Fixture func(cfg *Config) (any, func(), error)

type FixtureResult struct {
	Value   any
	Cleanup func()
}

type Fixtures struct {
	Registry map[string]Fixture
	Cache    map[string]FixtureResult
}

type FixturesOption func(*Fixtures)

func NewFixtures(options ...FixturesOption) Fixtures {
	f := Fixtures{}
	for _, option := range options {
		option(&f)
	}

	return f
}

func WithFixture(name string, fixture Fixture) FixturesOption {
	return func(f *Fixtures) {
		if f.Registry == nil {
			f.Registry = map[string]Fixture{}
		}
		f.Registry[name] = fixture
	}
}

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

func GetFixture[T any](cfg *Config, name string) T {
	var zero T

	if cfg == nil {
		panic("fixture: nil config")
	}

	if res, ok := cfg.Fixtures.Cache[name]; ok {
		return res.Value.(T)
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
		cfg.Hooks.AfterTest = append(cfg.Hooks.AfterTest, fixtureCleanupHook(name, cleanup))
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

func fixtureCleanupHook(name string, cleanup func()) TestHook {
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
