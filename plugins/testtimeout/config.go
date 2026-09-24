package testtimeout

import (
	"os"
	"time"
)

// AxiomTestTimeout is the environment variable read by ConfigFromEnv.
const AxiomTestTimeout = "AXIOM_TEST_TIMEOUT"

// Config controls the attempt timeout, failure message, and goroutine dump.
type Config struct {
	Timeout        time.Duration
	Message        string
	DumpGoroutines bool
}

// ConfigOption configures test timeout behavior.
type ConfigOption func(*Config)

// NewConfig returns timeout settings with goroutine dumps enabled by default.
func NewConfig(opts ...ConfigOption) Config {
	c := Config{DumpGoroutines: true}
	for _, opt := range opts {
		opt(&c)
	}

	return c
}

// WithTimeout sets the wall-clock threshold for reporting a failed attempt.
// It does not stop the test body. A non-positive duration disables the wrapper.
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
