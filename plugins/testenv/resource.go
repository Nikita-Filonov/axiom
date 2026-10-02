package testenv

import (
	"errors"
	"os"
	"strings"

	"github.com/Nikita-Filonov/axiom"
)

// Envs is a read-only snapshot of environment variables. Values may contain
// secrets; do not log the snapshot or its values without checking their use.
type Envs struct {
	values map[string]string
}

// Lookup returns the raw value and whether the variable was explicitly set.
// An empty value with ok == true is different from an unset variable.
func (e *Envs) Lookup(name string) (value string, ok bool) {
	value, ok = e.values[name]
	return value, ok
}

// Config selects the environment source. By default Resource captures os.Environ.
type Config struct {
	Source func() []string
}

// ConfigOption configures the environment resource.
type ConfigOption func(*Config)

// WithSource selects a source for the environment snapshot. The source is
// called once, on first access. Entries use the os.Environ KEY=value format.
func WithSource(source func() []string) ConfigOption {
	return func(c *Config) { c.Source = source }
}

// Resource registers a lazy environment snapshot on a Runner. Constructing a
// Runner does not read the environment. Repeated installation follows Axiom's
// normal resource replacement rules. It panics for nil options or source.
func Resource(options ...ConfigOption) axiom.ResourceRegistrar {
	c := Config{Source: os.Environ}
	for _, option := range options {
		if option == nil {
			panic("testenv: nil config option")
		}
		option(&c)
	}
	if c.Source == nil {
		panic("testenv: nil environment source")
	}
	return axiom.DefineResource(stateKey.Name(), c.build)
}

func (c Config) build(*axiom.Runner) (*Envs, func(), error) {
	values := make(map[string]string)
	for _, entry := range c.Source() {
		name, value, ok := strings.Cut(entry, "=")
		if ok {
			values[name] = value
		}
	}
	return &Envs{values: values}, nil, nil
}

// Get returns the runner's environment snapshot or panics if TryGet fails.
func Get(runner *axiom.Runner) *Envs {
	envs, err := TryGet(runner)
	if err != nil {
		panic(err)
	}
	return envs
}

// TryGet returns the snapshot or an error for a nil Runner or missing resource.
func TryGet(runner *axiom.Runner) (*Envs, error) {
	if runner == nil {
		return nil, errors.New("testenv: nil runner")
	}
	return stateKey.TryGet(runner)
}

var stateKey = axiom.NewResourceKey[*Envs]("testenv.envs")
