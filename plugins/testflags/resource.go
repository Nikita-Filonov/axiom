package testflags

import (
	"errors"
	"flag"

	"github.com/Nikita-Filonov/axiom"
)

// Resource defines a flag snapshot for registration with axiom.WithRunnerResources.
// It does not read or parse flags and can be declared at package level. The first
// successful Get captures the selected source once; subsequent readers share that snapshot.
// It panics for a nil option or source. Configure runners before execution.
// Repeated installation replaces the definition, following Axiom resource rules.
func Resource(options ...ConfigOption) axiom.ResourceRegistrar {
	c := newConfig(options...)
	return axiom.DefineResource(stateKey.Name(), c.build)
}

// Get returns the runner's flag snapshot or panics if TryGet fails.
func Get(r *axiom.Runner) *Flags {
	flags, err := TryGet(r)
	if err != nil {
		panic(err)
	}
	return flags
}

// TryGet returns the snapshot or an error for a nil runner, missing resource,
// or unparsed source. Reading too early does not cache an error: a later call
// after Parse can succeed. Parsing and source mutation must not race with reads.
// Runner.Copy and Runner.Join follow Axiom resource semantics: definitions get
// fresh lazy state; already cached state and snapshots are shared.
// A panic from a custom flag value during capture is repeated on later reads.
func TryGet(r *axiom.Runner) (*Flags, error) {
	if r == nil {
		return nil, errors.New("testflags: nil runner")
	}
	s, err := stateKey.TryGet(r)
	if err != nil {
		return nil, err
	}
	if !s.source.Parsed() {
		return nil, errors.New("testflags: source has not been parsed; call flag.Parse before reading flags in TestMain")
	}
	return s.read(), nil
}

var stateKey = axiom.NewResourceKey[*state]("testflags.flags")

type state struct {
	read   func() *Flags
	source *flag.FlagSet
}
