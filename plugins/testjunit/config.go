package testjunit

import (
	"log"
	"path/filepath"
	"strings"
)

const defaultSuiteName = "axiom"

// Option configures a Plugin installation.
type Option func(*Config)

// Config selects a collector, suite name, and optional incremental XML output.
type Config struct {
	Reporter     *Reporter
	SuiteName    string
	OutputFile   string
	ErrorHandler func(error)
}

func newConfig(options ...Option) Config {
	c := Config{
		SuiteName:    defaultSuiteName,
		ErrorHandler: func(err error) { log.Print(err) },
	}
	for _, option := range options {
		if option == nil {
			panic("testjunit: nil option")
		}
		option(&c)
	}
	if c.Reporter == nil {
		c.Reporter = NewReporter()
	}
	if c.ErrorHandler == nil {
		panic("testjunit: nil error handler")
	}
	if c.OutputFile != "" {
		path, err := filepath.Abs(c.OutputFile)
		if err != nil {
			panic("testjunit: resolve output file: " + err.Error())
		}
		c.OutputFile = path
	}
	return c
}

// WithReporter collects attempts in reporter. Without it, Plugin creates one
// collector per Plugin call. Passing nil panics.
func WithReporter(reporter *Reporter) Option {
	if reporter == nil {
		panic("testjunit: nil reporter")
	}
	return func(c *Config) { c.Reporter = reporter }
}

// WithOutputFile enables incremental XML snapshots for the entire Reporter.
// Relative paths are resolved when Plugin is created. No file is written until
// an attempt finishes. Each process and independent Reporter needs its own path.
// The output is registered on plugin installation. Repeated registration of the
// same path uses the first installed error handler.
func WithOutputFile(path string) Option {
	if strings.TrimSpace(path) == "" {
		panic("testjunit: empty output file")
	}
	return func(c *Config) { c.OutputFile = path }
}

// WithErrorHandler replaces stderr logging of automatic output failures.
// The handler runs without internal locks and may be called concurrently.
// It should not fail or skip tests. Passing nil is rejected by Plugin.
func WithErrorHandler(handler func(error)) Option {
	return func(c *Config) { c.ErrorHandler = handler }
}

// WithSuiteName sets the suite name and classname for this Plugin's attempts.
// An empty name panics.
func WithSuiteName(name string) Option {
	if name == "" {
		panic("testjunit: empty suite name")
	}
	return func(c *Config) { c.SuiteName = name }
}
