package axiom

import "fmt"

// Local stores typed values for one Config and is fresh for each retry attempt.
// It is not safe for concurrent mutation; synchronize access when sharing a
// Config across goroutines.
type Local struct {
	values map[any]any
}

// LocalKey identifies a typed slot in Local by name and type. Keys with the
// same name and type address the same slot. Its zero value is invalid.
type LocalKey[T any] struct {
	name string
}

// NewLocalKey creates a typed key and panics if name is empty.
func NewLocalKey[T any](name string) LocalKey[T] {
	if name == "" {
		panic("local: key name must not be empty")
	}

	return LocalKey[T]{name: name}
}

// SetLocal stores value in the current attempt's Local state. It panics if cfg
// is nil or key is invalid.
func SetLocal[T any](cfg *Config, key LocalKey[T], value T) {
	if cfg == nil {
		panic("local: nil *Config")
	}
	if key.name == "" {
		panic("local: key must be created with NewLocalKey")
	}

	if cfg.Local.values == nil {
		cfg.Local.values = map[any]any{}
	}
	cfg.Local.values[key] = value
}

// GetLocal returns the value under key and whether it was set. A stored nil
// value is present and returns the zero value with true.
func GetLocal[T any](cfg *Config, key LocalKey[T]) (T, bool) {
	if cfg == nil {
		panic("local: nil *Config")
	}
	if key.name == "" {
		panic("local: key must be created with NewLocalKey")
	}

	v, ok := cfg.Local.values[key]
	if !ok {
		var zero T
		return zero, false
	}
	if v == nil {
		var zero T
		return zero, true
	}

	return v.(T), true
}

// MustLocal returns the value under key or panics if no value was set.
func MustLocal[T any](cfg *Config, key LocalKey[T]) T {
	v, ok := GetLocal(cfg, key)
	if !ok {
		panic(fmt.Sprintf("local: missing value for key %q", key.name))
	}

	return v
}
