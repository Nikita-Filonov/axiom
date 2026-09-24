package axiom

// ConfigWithTools gives a test action its Config and a typed helper bundle.
type ConfigWithTools[T any] struct {
	*Config
	Tools T
}

// Toolset builds and stores a typed helper bundle in a Config's Local state.
// Bind must run for each attempt before Use, Action, or Must reads the bundle.
type Toolset[T any] struct {
	key   LocalKey[T]
	build func(*Config) T
}

// NewToolset creates a named helper bundle definition. It panics if name is
// empty or build is nil.
func NewToolset[T any](name string, build func(*Config) T) Toolset[T] {
	if build == nil {
		panic("toolset: nil build")
	}

	return Toolset[T]{key: NewLocalKey[T](name), build: build}
}

// Bind builds and stores the helper bundle for cfg's current attempt.
func (t Toolset[T]) Bind(cfg *Config) {
	t.validate()
	if cfg == nil {
		panic("local: nil *Config")
	}

	SetLocal(cfg, t.key, t.build(cfg))
}

// Use adapts action into a TestAction with a ConfigWithTools value. The bundle
// must already have been bound for the attempt.
func (t Toolset[T]) Use(action func(*ConfigWithTools[T])) TestAction {
	t.validate()
	if action == nil {
		panic("toolset: nil action")
	}

	return func(cfg *Config) {
		action(&ConfigWithTools[T]{Config: cfg, Tools: t.Must(cfg)})
	}
}

// Action adapts action into a TestAction that receives Config and the bound
// helper bundle separately.
func (t Toolset[T]) Action(action func(*Config, T)) TestAction {
	t.validate()
	if action == nil {
		panic("toolset: nil action")
	}

	return func(cfg *Config) {
		action(cfg, t.Must(cfg))
	}
}

func (t Toolset[T]) Get(cfg *Config) (T, bool) {
	t.validate()
	return GetLocal(cfg, t.key)
}

func (t Toolset[T]) Must(cfg *Config) T {
	t.validate()
	return MustLocal(cfg, t.key)
}

func (t Toolset[T]) validate() {
	if t.key.name == "" {
		panic("toolset: empty toolset")
	}
}
