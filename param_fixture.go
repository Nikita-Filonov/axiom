package axiom

type ParamFixture[P, T any] struct {
	key   FixtureKey[T]
	build func(*Config, P) (T, func(), error)
}

func DefineParamFixture[P, T any](
	name string,
	build func(*Config, P) (T, func(), error),
) ParamFixture[P, T] {
	if build == nil {
		panic("fixture: nil constructor")
	}

	return ParamFixture[P, T]{key: NewFixtureKey[T](name), build: build}
}

func (f ParamFixture[P, T]) Name() string { return f.key.name }

func (f ParamFixture[P, T]) Key() FixtureKey[T] { return f.key }

func (f ParamFixture[P, T]) For(params P) CaseFixtureRegistrar {
	f.validate()
	return FixtureDef[T]{key: f.key, build: f.typed(params)}
}

func (f ParamFixture[P, T]) Default(params P) RunnerOption {
	f.validate()
	return WithRunnerFixtureKey(f.key, f.typed(params))
}

func (f ParamFixture[P, T]) Get(cfg *Config) T {
	f.validate()
	return f.key.Get(cfg)
}

func (f ParamFixture[P, T]) typed(params P) TypedFixture[T] {
	return func(cfg *Config) (T, func(), error) {
		return f.build(cfg, params)
	}
}

func (f ParamFixture[P, T]) validate() {
	if f.key.name == "" {
		panic("fixture: param fixture must be created with DefineParamFixture")
	}
}
