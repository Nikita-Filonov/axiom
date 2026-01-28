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

func (f *Fixtures) Join(other Fixtures) Fixtures {
	result := Fixtures{
		Registry: map[string]Fixture{},
		Cache:    map[string]FixtureResult{},
	}

	for k, v := range f.Registry {
		result.Registry[k] = v
	}
	for k, v := range other.Registry {
		result.Registry[k] = v
	}

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
	if res, ok := cfg.Fixtures.Cache[name]; ok {
		return res.Value.(T)
	}

	fx, ok := cfg.Fixtures.Registry[name]
	if !ok {
		cfg.SubT.Fatalf("fixture %q not found", name)
	}

	val, cleanup, err := fx(cfg)
	if err != nil {
		cfg.SubT.Fatalf("fixture %q failed: %v", name, err)
	}

	result := FixtureResult{Value: val, Cleanup: cleanup}
	cfg.Fixtures.Cache[name] = result

	if cleanup != nil {
		cfg.Hooks.AfterTest = append(cfg.Hooks.AfterTest, func(_ *Config) { cleanup() })
	}

	out, ok := val.(T)
	if !ok {
		cfg.SubT.Fatalf("fixture %q has unexpected type", name)
	}
	return out
}

func UseFixtures(names ...string) func(cfg *Config) {
	return func(cfg *Config) {
		for _, name := range names {
			GetFixture[any](cfg, name)
		}
	}
}
