package axiom

// ContextKey is a typed handle to a value in Context.Data. Keys with the same
// name and type address the same entry. Its zero value is invalid.
type ContextKey[T any] struct {
	name string
}

// NewContextKey creates a typed key and panics if name is empty.
func NewContextKey[T any](name string) ContextKey[T] {
	if name == "" {
		panic("context: key name must not be empty")
	}

	return ContextKey[T]{name: name}
}

// Name returns the key's name.
func (k ContextKey[T]) Name() string { return k.name }

// Value returns an option that stores value under this key.
func (k ContextKey[T]) Value(value T) ContextOption {
	k.validate()
	return WithContextData(k.name, value)
}

// Get returns the key's value or panics if it is absent or has the wrong type.
func (k ContextKey[T]) Get(c *Context) T {
	k.validate()
	return MustContextValue[T](c, k.name)
}

// TryGet returns the key's value and whether a value of type T was found.
func (k ContextKey[T]) TryGet(c *Context) (T, bool) {
	k.validate()
	return GetContextValue[T](c, k.name)
}

func (k ContextKey[T]) validate() {
	if k.name == "" {
		panic("context: key must be created with NewContextKey")
	}
}
