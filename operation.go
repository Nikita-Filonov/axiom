package axiom

import "fmt"

// Operation describes one Step, Setup, or Teardown invocation. Its name and
// values are immutable after construction. Nested operations do not inherit
// values, and concurrent invocations have independent storage. Values are not
// deep copied: callers must not mutate shared values without synchronization.
// The zero value has an empty name and no values.
type Operation struct {
	name   string
	values map[any]any
}

// Name returns the human-readable operation name.
func (o Operation) Name() string { return o.name }

type operationBuilder struct {
	values map[any]any
}

// OperationOption supplies values for a single operation. Options run in order
// before lifecycle events or hooks; the last value for a key wins. Create options
// with WithOperationValue or OperationKey.Value. Nil options are invalid.
type OperationOption func(*operationBuilder)

func newOperation(name string, options ...OperationOption) Operation {
	b := operationBuilder{}
	for _, option := range options {
		option(&b)
	}
	return Operation{name: name, values: b.values}
}

// WithOperationValue stores value under key for this invocation only. It panics
// for an invalid key. The returned option may be reused concurrently provided
// the value itself is safe to share.
func WithOperationValue[T any](key OperationKey[T], value T) OperationOption {
	key.validate()
	return func(b *operationBuilder) {
		if b.values == nil {
			b.values = make(map[any]any)
		}
		b.values[key] = value
	}
}

// GetOperationValue returns the value and whether key is present. A stored nil
// interface is present and returns the zero value with true. It panics for an
// invalid key. Reading values does not modify the operation.
func GetOperationValue[T any](operation Operation, key OperationKey[T]) (T, bool) {
	key.validate()
	value, ok := operation.values[key]
	if !ok || value == nil {
		var zero T
		return zero, ok
	}
	return value.(T), true
}

// MustOperationValue returns the value under key or panics if it is absent or
// the key is invalid. A stored nil value is present and does not cause a panic.
func MustOperationValue[T any](operation Operation, key OperationKey[T]) T {
	value, ok := GetOperationValue(operation, key)
	if !ok {
		panic(fmt.Sprintf("operation: missing value for key %q", key.name))
	}
	return value
}
