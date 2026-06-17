package axiom

import "fmt"

type Local struct {
	values map[any]any
}

type LocalKey[T any] struct {
	name string
}

func NewLocalKey[T any](name string) LocalKey[T] {
	if name == "" {
		panic("local: key name must not be empty")
	}

	return LocalKey[T]{name: name}
}

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

func MustLocal[T any](cfg *Config, key LocalKey[T]) T {
	v, ok := GetLocal(cfg, key)
	if !ok {
		panic(fmt.Sprintf("local: missing value for key %q", key.name))
	}

	return v
}
