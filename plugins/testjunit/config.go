package testjunit

const defaultSuiteName = "axiom"

// Option configures a Plugin installation.
type Option func(*Config)

// Config controls the suite assigned to attempts recorded by a Plugin.
type Config struct {
	SuiteName string
}

func newConfig(options ...Option) Config {
	c := Config{SuiteName: defaultSuiteName}
	for _, option := range options {
		if option == nil {
			panic("testjunit: nil option")
		}
		option(&c)
	}
	return c
}

// WithSuiteName sets the suite name and classname for this Plugin's attempts.
// An empty name panics.
func WithSuiteName(name string) Option {
	if name == "" {
		panic("testjunit: empty suite name")
	}
	return func(c *Config) { c.SuiteName = name }
}
