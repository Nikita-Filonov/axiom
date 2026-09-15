package axiom

type TypedFixture[T any] func(cfg *Config) (T, func(), error)

type FixtureKey[T any] struct {
	name string
}

func NewFixtureKey[T any](name string) FixtureKey[T] {
	if name == "" {
		panic("fixture: key name must not be empty")
	}

	return FixtureKey[T]{name: name}
}

func (k FixtureKey[T]) Name() string { return k.name }

func (k FixtureKey[T]) Get(cfg *Config) T {
	k.validate()
	return GetFixture[T](cfg, k.name)
}

func (k FixtureKey[T]) validate() {
	if k.name == "" {
		panic("fixture: key must be created with NewFixtureKey")
	}
}

type FixtureDef[T any] struct {
	key   FixtureKey[T]
	build TypedFixture[T]
}

func DefineFixture[T any](name string, build TypedFixture[T]) FixtureDef[T] {
	if build == nil {
		panic("fixture: nil constructor")
	}

	return FixtureDef[T]{key: NewFixtureKey[T](name), build: build}
}

func (d FixtureDef[T]) Key() FixtureKey[T] { return d.key }

func (d FixtureDef[T]) Name() string { return d.key.name }

func (d FixtureDef[T]) Get(cfg *Config) T { return d.key.Get(cfg) }

func (d FixtureDef[T]) registerRunnerFixture(r *Runner) {
	WithRunnerFixtureKey(d.key, d.build)(r)
}

func (d FixtureDef[T]) registerCaseFixture(c *Case) {
	WithCaseFixtureKey(d.key, d.build)(c)
}

type RunnerFixtureRegistrar interface {
	registerRunnerFixture(*Runner)
}

type CaseFixtureRegistrar interface {
	registerCaseFixture(*Case)
}
