package axiom

// TypedResource constructs a typed runner scoped value and optional cleanup.
type TypedResource[T any] func(r *Runner) (T, func(), error)

// ResourceKey names a typed resource in the ordinary resource registry. It
// does not include a constructor. Its zero value is invalid.
type ResourceKey[T any] struct {
	name string
}

// NewResourceKey creates a typed key and panics if name is empty.
func NewResourceKey[T any](name string) ResourceKey[T] {
	if name == "" {
		panic("resource: key name must not be empty")
	}

	return ResourceKey[T]{name: name}
}

// Name returns the resource key's name.
func (k ResourceKey[T]) Name() string { return k.name }

// Get resolves the resource and panics if it cannot be obtained.
func (k ResourceKey[T]) Get(runner *Runner) T {
	k.validate()
	return MustResource[T](runner, k.name)
}

// TryGet resolves the resource and returns any lookup or construction error.
func (k ResourceKey[T]) TryGet(runner *Runner) (T, error) {
	k.validate()
	return GetResource[T](runner, k.name)
}

func (k ResourceKey[T]) validate() {
	if k.name == "" {
		panic("resource: key must be created with NewResourceKey")
	}
}

// ResourceDef pairs a ResourceKey with its constructor for registration on a
// Runner through WithRunnerResources.
type ResourceDef[T any] struct {
	key   ResourceKey[T]
	build TypedResource[T]
}

// DefineResource creates a typed resource definition. It panics if name is
// empty or build is nil.
func DefineResource[T any](name string, build TypedResource[T]) ResourceDef[T] {
	if build == nil {
		panic("resource: nil constructor")
	}

	return ResourceDef[T]{key: NewResourceKey[T](name), build: build}
}

// Key returns the typed key for this resource definition.
func (d ResourceDef[T]) Key() ResourceKey[T] { return d.key }

// Name returns the resource definition's name.
func (d ResourceDef[T]) Name() string { return d.key.name }

// Get returns this resource's value, constructing it on first access.
func (d ResourceDef[T]) Get(runner *Runner) T { return d.key.Get(runner) }

// TryGet returns this resource's value or its construction error.
func (d ResourceDef[T]) TryGet(runner *Runner) (T, error) { return d.key.TryGet(runner) }

func (d ResourceDef[T]) registerResource(r *Runner) {
	WithRunnerResourceKey(d.key, d.build)(r)
}

// ResourceRegistrar registers a resource definition on a Runner.
type ResourceRegistrar interface {
	registerResource(*Runner)
}
