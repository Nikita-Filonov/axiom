package testtimeout

import (
	"os"
	"time"
)

const AxiomTestTimeout = "AXIOM_TEST_TIMEOUT"

type Config struct {
	Timeout        time.Duration
	Message        string
	DumpGoroutines bool
}

type ConfigOption func(*Config)

func NewConfig(opts ...ConfigOption) Config {
	c := Config{DumpGoroutines: true}
	for _, opt := range opts {
		opt(&c)
	}

	return c
}

func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) { c.Timeout = timeout }
}

func WithMessage(message string) ConfigOption {
	return func(c *Config) { c.Message = message }
}

func WithoutGoroutineDump() ConfigOption {
	return func(c *Config) { c.DumpGoroutines = false }
}

func ConfigFromEnv() ConfigOption {
	return func(c *Config) {
		raw := os.Getenv(AxiomTestTimeout)
		if raw == "" {
			return
		}

		if timeout, err := time.ParseDuration(raw); err == nil {
			c.Timeout = timeout
		}
	}
}
