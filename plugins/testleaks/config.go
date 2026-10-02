package testleaks

import "time"

// Option configures the leak checker.
type Option func(*Config)

// Config controls the checks performed after an attempt.
type Config struct {
	GracePeriod     time.Duration
	IgnoreFunctions []string
	Goroutines      bool
}

func newConfig(options ...Option) Config {
	c := Config{GracePeriod: 200 * time.Millisecond, Goroutines: true}
	for _, option := range options {
		if option == nil {
			panic("testleaks: nil option")
		}
		option(&c)
	}
	return c
}

// WithGracePeriod sets how long to wait for goroutines and tracked resources
// to finish after test cleanup. Zero checks immediately; negative values panic.
func WithGracePeriod(d time.Duration) Option {
	if d < 0 {
		panic("testleaks: negative grace period")
	}
	return func(c *Config) { c.GracePeriod = d }
}

// WithIgnoreFunction excludes a goroutine when any frame has exactly this
// function name. It is intended for deliberately long-lived goroutines.
func WithIgnoreFunction(name string) Option {
	if name == "" {
		panic("testleaks: empty function name")
	}
	return func(c *Config) { c.IgnoreFunctions = append(c.IgnoreFunctions, name) }
}

// WithoutGoroutines disables the goroutine check. Explicit resource tracking
// remains active.
func WithoutGoroutines() Option {
	return func(c *Config) { c.Goroutines = false }
}
