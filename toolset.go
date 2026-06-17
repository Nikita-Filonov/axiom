package axiom

type ConfigWithTools[T any] struct {
	*Config
	Tools T
}

type Toolset[T any] struct {
	key   LocalKey[T]
	build func(*Config) T
}

func NewToolset[T any](name string, build func(*Config) T) Toolset[T] {
	if build == nil {
		panic("toolset: nil build")
	}

	return Toolset[T]{key: NewLocalKey[T](name), build: build}
}

func (t Toolset[T]) Bind(cfg *Config) {
	t.validate()
	if cfg == nil {
		panic("local: nil *Config")
	}

	SetLocal(cfg, t.key, t.build(cfg))
}

func (t Toolset[T]) Use(action func(*ConfigWithTools[T])) TestAction {
	t.validate()
	if action == nil {
		panic("toolset: nil action")
	}

	return func(cfg *Config) {
		action(&ConfigWithTools[T]{Config: cfg, Tools: t.Must(cfg)})
	}
}

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
