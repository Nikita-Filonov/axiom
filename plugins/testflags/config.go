package testflags

import (
	"flag"
	"sync"

	"github.com/Nikita-Filonov/axiom"
)

// Config selects the flag set to snapshot. The default is flag.CommandLine.
type Config struct {
	Source *flag.FlagSet
}

// ConfigOption configures a runner's flag resource.
type ConfigOption func(*Config)

func newConfig(options ...ConfigOption) Config {
	c := Config{Source: flag.CommandLine}
	for _, option := range options {
		if option == nil {
			panic("testflags: nil config option")
		}
		option(&c)
	}
	if c.Source == nil {
		panic("testflags: nil flag source")
	}
	return c
}

// WithSource selects an already registered flag set. It must be non-nil and
// parsed before first access. WithFlagSet selects this set for declarations.
func WithSource(fs *flag.FlagSet) ConfigOption {
	return func(c *Config) { c.Source = fs }
}

func (c Config) build(_ *axiom.Runner) (*state, func(), error) {
	return &state{
		read:   sync.OnceValue(func() *Flags { return snapshot(c.Source) }),
		source: c.Source,
	}, nil, nil
}

// FlagConfig describes a flag declaration. Name is required; Usage is empty
// and Default is the constructor's zero value unless configured. FlagSet
// defaults to flag.CommandLine. Use the options below to configure a declaration.
type FlagConfig struct {
	Name    string
	Usage   string
	Default any
	FlagSet *flag.FlagSet
}

// FlagOption configures a flag before it is registered.
type FlagOption func(*FlagConfig)

func newFlagConfig(options ...FlagOption) FlagConfig {
	c := FlagConfig{FlagSet: flag.CommandLine}
	for _, option := range options {
		if option == nil {
			panic("testflags: nil flag option")
		}
		option(&c)
	}
	return c
}

// WithName sets the command-line name without leading dashes.
func WithName(name string) FlagOption {
	return func(c *FlagConfig) { c.Name = name }
}

// WithUsage sets the description displayed by the standard flag help output.
func WithUsage(usage string) FlagOption {
	return func(c *FlagConfig) { c.Usage = usage }
}

// WithDefault sets the default value. Its type must match the constructor
// exactly: for example int for Int and time.Duration for Duration. A constructor
// panics on a mismatched type; WithDefault panics on nil.
func WithDefault(value any) FlagOption {
	if value == nil {
		panic("testflags: nil default")
	}
	return func(c *FlagConfig) { c.Default = value }
}

// WithFlagSet selects the set on which to register a declaration. It must be
// non-nil and not yet parsed. WithSource selects the same set for a runner.
func WithFlagSet(fs *flag.FlagSet) FlagOption {
	return func(c *FlagConfig) { c.FlagSet = fs }
}
