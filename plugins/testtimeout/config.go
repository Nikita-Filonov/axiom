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

// ConfigOption configures test timeout behavior.
type ConfigOption func(*Config)

func NewConfig(opts ...ConfigOption) Config {
	c := Config{DumpGoroutines: true}
	for _, opt := range opts {
		opt(&c)
	}

	return c
}

// WithTimeout sets the attempt limit. A non-positive duration disables the
// timeout wrapper.
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) { c.Timeout = timeout }
}

// WithMessage sets the failure text reported when the timeout expires.
func WithMessage(message string) ConfigOption {
	return func(c *Config) { c.Message = message }
}

// WithoutGoroutineDump omits the goroutine dump artefact on timeout.
func WithoutGoroutineDump() ConfigOption {
	return func(c *Config) { c.DumpGoroutines = false }
}

// ConfigFromEnv reads AXIOM_TEST_TIMEOUT as a Go duration, leaving Timeout
// unchanged if the variable is empty or invalid.
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
