package testenv

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

// EnvKey is a typed handle to one variable in a Runner's environment snapshot.
// The zero value is invalid. A key is safe to reuse across runners.
type EnvKey[T any] struct {
	name       string
	parse      func(string) (T, error)
	fallback   T
	hasDefault bool
}

// Name returns the environment variable name.
func (k EnvKey[T]) Name() string { return k.name }

// Get reads the typed value and panics if TryGet fails.
func (k EnvKey[T]) Get(runner *axiom.Runner) T {
	value, err := k.TryGet(runner)
	if err != nil {
		panic(err)
	}
	return value
}

// TryGet reads and parses the value from the Runner's snapshot. An explicitly
// empty value is parsed as empty, not replaced by the default. It returns an
// error for a missing required variable or malformed value.
func (k EnvKey[T]) TryGet(runner *axiom.Runner) (T, error) {
	var zero T
	if k.name == "" || k.parse == nil {
		panic("testenv: key must be created with a typed constructor")
	}
	envs, err := TryGet(runner)
	if err != nil {
		return zero, err
	}
	raw, ok := envs.Lookup(k.name)
	if !ok {
		if k.hasDefault {
			return k.fallback, nil
		}
		return zero, fmt.Errorf("testenv: required variable %q is not set", k.name)
	}
	value, err := k.parse(raw)
	if err != nil {
		// Parser errors often include the raw value, which may be a secret.
		return zero, fmt.Errorf("testenv: invalid value for %q (%T)", k.name, zero)
	}
	return value, nil
}

// String creates a string key. Empty strings are valid when explicitly set.
func String(options ...EnvOption) EnvKey[string] {
	return define(func(raw string) (string, error) { return raw, nil }, options)
}

// Bool creates a boolean key using strconv.ParseBool syntax.
func Bool(options ...EnvOption) EnvKey[bool] { return define(strconv.ParseBool, options) }

// Int creates an int key using base-10 syntax.
func Int(options ...EnvOption) EnvKey[int] { return define(strconv.Atoi, options) }

// Int64 creates an int64 key using base-10 syntax.
func Int64(options ...EnvOption) EnvKey[int64] {
	return define(func(raw string) (int64, error) { return strconv.ParseInt(raw, 10, 64) }, options)
}

// Uint creates a uint key using base-10 syntax.
func Uint(options ...EnvOption) EnvKey[uint] {
	return define(func(raw string) (uint, error) {
		value, err := strconv.ParseUint(raw, 10, 0)
		return uint(value), err
	}, options)
}

// Uint64 creates a uint64 key using base-10 syntax.
func Uint64(options ...EnvOption) EnvKey[uint64] {
	return define(func(raw string) (uint64, error) { return strconv.ParseUint(raw, 10, 64) }, options)
}

// Float64 creates a float64 key using strconv.ParseFloat syntax.
func Float64(options ...EnvOption) EnvKey[float64] {
	return define(func(raw string) (float64, error) { return strconv.ParseFloat(raw, 64) }, options)
}

// Duration creates a time.Duration key using time.ParseDuration syntax.
func Duration(options ...EnvOption) EnvKey[time.Duration] { return define(time.ParseDuration, options) }

func define[T any](parse func(string) (T, error), options []EnvOption) EnvKey[T] {
	c := newEnvConfig(options...)
	if c.Name == "" || strings.ContainsAny(c.Name, "=\x00") {
		panic(fmt.Sprintf("testenv: invalid environment variable name %q", c.Name))
	}
	key := EnvKey[T]{name: c.Name, parse: parse}
	if c.Default != nil {
		var ok bool
		key.fallback, ok = c.Default.(T)
		if !ok {
			var zero T
			panic(fmt.Sprintf("testenv: default for %q must have type %T, got %T", c.Name, zero, c.Default))
		}
		key.hasDefault = true
	}
	return key
}
