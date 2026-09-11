package testquarantine

import (
	"os"

	"github.com/Nikita-Filonov/axiom"
)

const (
	AxiomTestQuarantineRun = "AXIOM_TEST_QUARANTINE_RUN"

	DefaultTag    = "quarantine"
	DefaultReason = "flaky"
)

type Predicate func(cfg *axiom.Config) (reason string, quarantined bool)

type Config struct {
	Run       bool
	Tag       string
	Reason    string
	Predicate Predicate
}

type ConfigOption func(*Config)

func NewConfig(opts ...ConfigOption) Config {
	c := Config{Tag: DefaultTag, Reason: DefaultReason}
	for _, opt := range opts {
		opt(&c)
	}

	return c
}

func WithRun(run bool) ConfigOption {
	return func(c *Config) { c.Run = run }
}

func WithTag(tag string) ConfigOption {
	return func(c *Config) { c.Tag = normalize(tag) }
}

func WithReason(reason string) ConfigOption {
	return func(c *Config) { c.Reason = reason }
}

func WithPredicate(predicate Predicate) ConfigOption {
	return func(c *Config) { c.Predicate = predicate }
}

func ConfigFromEnv() ConfigOption {
	return func(c *Config) {
		if run, ok := parseBool(os.Getenv(AxiomTestQuarantineRun)); ok {
			c.Run = run
		}
	}
}

func (c Config) decide(e *axiom.Config) (string, bool) {
	if c.Predicate != nil {
		return c.Predicate(e)
	}

	if hasTag(e.Meta.Tags, c.Tag) {
		return c.Reason, true
	}

	return "", false
}
