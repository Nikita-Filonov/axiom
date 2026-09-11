package axiom

type ContextKey[T any] struct {
	name string
}

func NewContextKey[T any](name string) ContextKey[T] {
	if name == "" {
		panic("context: key name must not be empty")
	}

	return ContextKey[T]{name: name}
}

func (k ContextKey[T]) Name() string { return k.name }

func (k ContextKey[T]) Value(value T) ContextOption {
	k.validate()
	return WithContextData(k.name, value)
}

func (k ContextKey[T]) Get(c *Context) T {
	k.validate()
	return MustContextValue[T](c, k.name)
}

func (k ContextKey[T]) TryGet(c *Context) (T, bool) {
	k.validate()
	return GetContextValue[T](c, k.name)
}

func (k ContextKey[T]) validate() {
	if k.name == "" {
		panic("context: key must be created with NewContextKey")
	}
}
