package testenv

import (
	"os"

	"github.com/Nikita-Filonov/axiom"
)

// Config selects the environment source. By default Resource captures os.Environ.
type Config struct {
	Source func() []string
}

// ConfigOption configures the environment resource.
type ConfigOption func(*Config)

func newConfig(options ...ConfigOption) Config {
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
	return c
}

// WithSource selects a source for the environment snapshot. The source is
// called once, on first access. Entries use the os.Environ KEY=value format.
func WithSource(source func() []string) ConfigOption {
	return func(c *Config) { c.Source = source }
}

func (c Config) build(_ *axiom.Runner) (*Envs, func(), error) {
	return snapshot(c.Source()), nil, nil
}

// EnvConfig describes a typed environment variable. Name is required.
// Variables without WithDefault are required when read.
type EnvConfig struct {
	Name    string
	Default any
}

// EnvOption configures a typed environment key.
type EnvOption func(*EnvConfig)

func newEnvConfig(options ...EnvOption) EnvConfig {
	c := EnvConfig{}
	for _, option := range options {
		if option == nil {
			panic("testenv: nil env option")
		}
		option(&c)
	}
	return c
}

// WithName sets the exact, case-sensitive environment variable name.
func WithName(name string) EnvOption {
	return func(c *EnvConfig) { c.Name = name }
}

// WithDefault makes an unset variable optional. Its type must match the key's
// constructor exactly; nil and mismatched values panic during construction.
func WithDefault(value any) EnvOption {
	if value == nil {
		panic("testenv: nil default")
	}
	return func(c *EnvConfig) { c.Default = value }
}
