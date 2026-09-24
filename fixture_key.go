package axiom

// TypedFixture constructs a typed fixture value and optional cleanup for one
// test attempt.
type TypedFixture[T any] func(cfg *Config) (T, func(), error)

// FixtureKey names a typed fixture in the ordinary fixture registry. It does
// not include a constructor and has the same per-attempt lifecycle as Fixture.
// Its zero value is invalid.
type FixtureKey[T any] struct {
	name string
}

// NewFixtureKey creates a typed key and panics if name is empty.
func NewFixtureKey[T any](name string) FixtureKey[T] {
	if name == "" {
		panic("fixture: key name must not be empty")
	}

	return FixtureKey[T]{name: name}
}

// Name returns the fixture key's name.
func (k FixtureKey[T]) Name() string { return k.name }

// Get resolves the fixture in cfg through [GetFixture].
func (k FixtureKey[T]) Get(cfg *Config) T {
	k.validate()
	return GetFixture[T](cfg, k.name)
}

func (k FixtureKey[T]) validate() {
	if k.name == "" {
		panic("fixture: key must be created with NewFixtureKey")
	}
}

// FixtureDef pairs a FixtureKey with its constructor for registration on a
// Runner or Case. Register it with WithRunnerFixtures or WithCaseFixtures.
type FixtureDef[T any] struct {
	key   FixtureKey[T]
	build TypedFixture[T]
}

// DefineFixture creates a typed fixture definition. It panics if name is empty
// or build is nil.
func DefineFixture[T any](name string, build TypedFixture[T]) FixtureDef[T] {
	if build == nil {
		panic("fixture: nil constructor")
	}

	return FixtureDef[T]{key: NewFixtureKey[T](name), build: build}
}

// Key returns the typed key for this fixture definition.
func (d FixtureDef[T]) Key() FixtureKey[T] { return d.key }

// Name returns the fixture definition's name.
func (d FixtureDef[T]) Name() string { return d.key.name }

// Get returns this fixture's value for the current attempt.
func (d FixtureDef[T]) Get(cfg *Config) T { return d.key.Get(cfg) }

func (d FixtureDef[T]) registerRunnerFixture(r *Runner) {
	WithRunnerFixtureKey(d.key, d.build)(r)
}

func (d FixtureDef[T]) registerCaseFixture(c *Case) {
	WithCaseFixtureKey(d.key, d.build)(c)
}

// RunnerFixtureRegistrar registers a fixture definition on a Runner.
type RunnerFixtureRegistrar interface {
	registerRunnerFixture(*Runner)
}

// CaseFixtureRegistrar registers a fixture definition on a Case.
type CaseFixtureRegistrar interface {
	registerCaseFixture(*Case)
}
