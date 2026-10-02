package testflags

import (
	"flag"
	"fmt"
	"time"
)

// Bool registers a boolean flag and returns its typed key. The default is false.
// Like the other constructors, it panics for a missing or invalid name, duplicate
// registration, nil option or set, parsed set, or incompatible default type.
func Bool(options ...FlagOption) FlagKey[bool] {
	return define((*flag.FlagSet).Bool, options)
}

// Int registers an int flag with a default of zero. It follows Bool's panic rules.
func Int(options ...FlagOption) FlagKey[int] {
	return define((*flag.FlagSet).Int, options)
}

// Int64 registers an int64 flag with a default of zero. It follows Bool's panic rules.
func Int64(options ...FlagOption) FlagKey[int64] {
	return define((*flag.FlagSet).Int64, options)
}

// Uint registers a uint flag with a default of zero. It follows Bool's panic rules.
func Uint(options ...FlagOption) FlagKey[uint] {
	return define((*flag.FlagSet).Uint, options)
}

// Uint64 registers a uint64 flag with a default of zero. It follows Bool's panic rules.
func Uint64(options ...FlagOption) FlagKey[uint64] {
	return define((*flag.FlagSet).Uint64, options)
}

// String registers a string flag with an empty default. It follows Bool's panic rules.
func String(options ...FlagOption) FlagKey[string] {
	return define((*flag.FlagSet).String, options)
}

// Float64 registers a float64 flag with a default of zero. It follows Bool's panic rules.
func Float64(options ...FlagOption) FlagKey[float64] {
	return define((*flag.FlagSet).Float64, options)
}

// Duration registers a time.Duration flag with a default of zero. It follows
// Bool's panic rules. Values use time.ParseDuration syntax, such as 500ms or 2s.
func Duration(options ...FlagOption) FlagKey[time.Duration] {
	return define((*flag.FlagSet).Duration, options)
}

func define[T any](register func(*flag.FlagSet, string, T, string) *T, options []FlagOption) FlagKey[T] {
	c := newFlagConfig(options...)
	key := NewFlagKey[T](c.Name)
	if c.FlagSet == nil {
		panic("testflags: nil flag set")
	}
	if c.FlagSet.Parsed() {
		panic("testflags: flags must be registered before parsing")
	}
	var value T
	if c.Default != nil {
		var ok bool
		value, ok = c.Default.(T)
		if !ok {
			panic(fmt.Sprintf("testflags: default for %q must have type %T, got %T", c.Name, value, c.Default))
		}
	}
	register(c.FlagSet, c.Name, value, c.Usage)
	return key
}
