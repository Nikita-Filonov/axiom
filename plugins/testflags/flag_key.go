package testflags

import (
	"fmt"

	"github.com/Nikita-Filonov/axiom"
)

// FlagKey is a typed handle to a value in a runner's flag snapshot. Keys with
// the same name and type address the same value. The zero value is invalid.
type FlagKey[T any] struct {
	name string
}

// NewFlagKey creates a key without registering a flag. Use it to read flags
// already registered with the standard flag package. It panics for an empty name.
func NewFlagKey[T any](name string) FlagKey[T] {
	if name == "" {
		panic("testflags: flag name must not be empty")
	}
	return FlagKey[T]{name: name}
}

// Name returns the flag name.
func (k FlagKey[T]) Name() string { return k.name }

// Get reads the typed value, creating the runner snapshot on first access.
// It panics if TryGet returns an error or the key is invalid.
func (k FlagKey[T]) Get(r *axiom.Runner) T {
	value, err := k.TryGet(r)
	if err != nil {
		panic(err)
	}
	return value
}

// TryGet returns an error for a nil runner, missing installation, unparsed
// source, absent flag, or incompatible type. It panics for an invalid key.
// The same key works in resources, hooks, and cases through cfg.Runner.
func (k FlagKey[T]) TryGet(r *axiom.Runner) (T, error) {
	if k.name == "" {
		panic("testflags: key must be created with NewFlagKey or a flag constructor")
	}
	var zero T
	flags, err := TryGet(r)
	if err != nil {
		return zero, err
	}
	value, ok := flags.Get(k.name)
	if !ok {
		return zero, fmt.Errorf("testflags: flag %q not found", k.name)
	}
	out, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("testflags: flag %q has type %T, expected %T", k.name, value, zero)
	}
	return out, nil
}
