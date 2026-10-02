package testenv

import (
	"errors"

	"github.com/Nikita-Filonov/axiom"
)

// Resource registers a lazy environment snapshot on a Runner. Constructing a
// Runner does not read the environment. Repeated installation follows Axiom's
// normal resource replacement rules. It panics for nil options or source.
func Resource(options ...ConfigOption) axiom.ResourceRegistrar {
	c := newConfig(options...)
	return axiom.DefineResource(envsKey.Name(), c.build)
}

// Get returns the runner's environment snapshot, creating it on first access.
// It panics with the error from TryGet if the snapshot cannot be obtained.
func Get(runner *axiom.Runner) *Envs {
	envs, err := TryGet(runner)
	if err != nil {
		panic(err)
	}
	return envs
}

// TryGet returns the runner's environment snapshot, creating it on first access.
// Concurrent callers share the cached snapshot; later changes to the source
// are not observed. It returns an error for a nil Runner or if the resource
// cannot be resolved, including a missing registration or type mismatch.
func TryGet(runner *axiom.Runner) (*Envs, error) {
	if runner == nil {
		return nil, errors.New("testenv: nil runner")
	}
	return envsKey.TryGet(runner)
}

var envsKey = axiom.NewResourceKey[*Envs]("testenv.envs")
