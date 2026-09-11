package axiom

type TypedResource[T any] func(r *Runner) (T, func(), error)

type ResourceKey[T any] struct {
	name string
}

func NewResourceKey[T any](name string) ResourceKey[T] {
	if name == "" {
		panic("resource: key name must not be empty")
	}

	return ResourceKey[T]{name: name}
}

func (k ResourceKey[T]) Name() string { return k.name }

func (k ResourceKey[T]) Get(runner *Runner) T {
	k.validate()
	return MustResource[T](runner, k.name)
}

func (k ResourceKey[T]) TryGet(runner *Runner) (T, error) {
	k.validate()
	return GetResource[T](runner, k.name)
}

func (k ResourceKey[T]) validate() {
	if k.name == "" {
		panic("resource: key must be created with NewResourceKey")
	}
}

type ResourceDef[T any] struct {
	key   ResourceKey[T]
	build TypedResource[T]
}

func DefineResource[T any](name string, build TypedResource[T]) ResourceDef[T] {
	if build == nil {
		panic("resource: nil constructor")
	}

	return ResourceDef[T]{key: NewResourceKey[T](name), build: build}
}

func (d ResourceDef[T]) Key() ResourceKey[T] { return d.key }

func (d ResourceDef[T]) Name() string { return d.key.name }

func (d ResourceDef[T]) Get(runner *Runner) T { return d.key.Get(runner) }

func (d ResourceDef[T]) TryGet(runner *Runner) (T, error) { return d.key.TryGet(runner) }

func (d ResourceDef[T]) registerResource(r *Runner) {
	WithRunnerResourceKey(d.key, d.build)(r)
}

type ResourceRegistrar interface {
	registerResource(*Runner)
}
