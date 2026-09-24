package testtags

import "os"

// Tag filter environment variables accepted by ConfigFromEnv.
const (
	AxiomTestTagsExclude = "AXIOM_TEST_TAGS_EXCLUDE"
	AxiomTestTagsInclude = "AXIOM_TEST_TAGS_INCLUDE"
)

// Config holds tags to include or exclude during case selection.
type Config struct {
	Include []string
	Exclude []string
}

// ConfigOption configures tag filtering.
type ConfigOption func(*Config)

// NewConfig returns a tag filter with the supplied options.
func NewConfig(opts ...ConfigOption) Config {
	c := Config{}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// WithConfigInclude appends normalized tags that cases may match to run.
func WithConfigInclude(tags ...string) ConfigOption {
	return func(c *Config) {
		for _, t := range tags {
			c.Include = append(c.Include, NormalizeTag(t))
		}
	}
}

// WithConfigExclude appends normalized tags that cause matching cases to skip.
func WithConfigExclude(tags ...string) ConfigOption {
	return func(c *Config) {
		for _, t := range tags {
			c.Exclude = append(c.Exclude, NormalizeTag(t))
		}
	}
}

// ConfigFromEnv appends include and exclude tags from the corresponding
// AXIOM_TEST_TAGS_INCLUDE and AXIOM_TEST_TAGS_EXCLUDE variables.
func ConfigFromEnv() ConfigOption {
	return func(c *Config) {
		c.Include = append(c.Include, ParseList(os.Getenv(AxiomTestTagsInclude))...)
		c.Exclude = append(c.Exclude, ParseList(os.Getenv(AxiomTestTagsExclude))...)
	}
}
