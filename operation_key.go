package axiom

// OperationKey identifies a typed value on an Operation. Keys with the same
// name and type address the same entry. Use a package-qualified name to avoid
// collisions between libraries. Its zero value is invalid.
type OperationKey[T any] struct {
	name string
}

// NewOperationKey creates a typed key and panics if name is empty.
func NewOperationKey[T any](name string) OperationKey[T] {
	if name == "" {
		panic("operation: key name must not be empty")
	}
	return OperationKey[T]{name: name}
}

// Name returns the key's name.
func (k OperationKey[T]) Name() string { return k.name }

// Value returns an option that stores value under this key.
func (k OperationKey[T]) Value(value T) OperationOption {
	return WithOperationValue(k, value)
}

// Get returns the key's value or panics if it is absent or the key is invalid.
func (k OperationKey[T]) Get(operation Operation) T {
	return MustOperationValue(operation, k)
}

// TryGet returns the key's value and whether it is present. It panics for an
// invalid key.
func (k OperationKey[T]) TryGet(operation Operation) (T, bool) {
	return GetOperationValue(operation, k)
}

func (k OperationKey[T]) validate() {
	if k.name == "" {
		panic("operation: key must be created with NewOperationKey")
	}
}
