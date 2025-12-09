package testtags

import "os"

type Config struct {
	Include []string
	Exclude []string
}

type ConfigOption func(*Config)

func NewConfig(opts ...ConfigOption) Config {
	c := Config{}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

func WithConfigInclude(tags ...string) ConfigOption {
	return func(c *Config) {
		for _, t := range tags {
			c.Include = append(c.Include, NormalizeTag(t))
		}
	}
}

func WithConfigExclude(tags ...string) ConfigOption {
	return func(c *Config) {
		for _, t := range tags {
			c.Exclude = append(c.Exclude, NormalizeTag(t))
		}
	}
}

func ConfigFromEnv() ConfigOption {
	return func(c *Config) {
		c.Include = append(c.Include, ParseList(os.Getenv("AXIOM_TEST_TAGS_INCLUDE"))...)
		c.Exclude = append(c.Exclude, ParseList(os.Getenv("AXIOM_TEST_TAGS_EXCLUDE"))...)
	}
}
