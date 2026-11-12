package teststats

import (
	"log"
	"path/filepath"
	"strings"
)

// ConfigOption configures the statistics plugin.
type ConfigOption func(*Config)

// Config selects the collector and optional JSON output. Use the options with
// Plugin; one default collector is created per Plugin call, not per attempt.
type Config struct {
	Stats        *Stats
	OutputDir    string
	ErrorHandler func(error)
}

func newConfig(options ...ConfigOption) Config {
	c := Config{ErrorHandler: func(err error) { log.Print(err) }}
	for _, option := range options {
		if option == nil {
			panic("teststats: nil option")
		}
		option(&c)
	}
	if c.Stats == nil {
		c.Stats = NewStats()
	}
	if c.ErrorHandler == nil {
		panic("teststats: nil error handler")
	}
	if c.OutputDir != "" {
		path, err := filepath.Abs(c.OutputDir)
		if err != nil {
			panic("teststats: resolve output directory: " + err.Error())
		}
		c.OutputDir = path
	}
	return c
}

// WithStats records attempts in stats so the caller can query them in memory.
// Passing nil panics.
func WithStats(stats *Stats) ConfigOption {
	if stats == nil {
		panic("teststats: nil stats")
	}
	return func(c *Config) { c.Stats = stats }
}

// WithOutputDir enables incremental JSON artifacts below dir. Relative paths
// are resolved when Plugin is created. Directories are created on the first
// observed attempt. An empty or whitespace-only path panics.
func WithOutputDir(dir string) ConfigOption {
	if strings.TrimSpace(dir) == "" {
		panic("teststats: empty output directory")
	}
	return func(c *Config) { c.OutputDir = dir }
}

// WithErrorHandler handles output failures instead of logging them to stderr.
// It may be called concurrently. The plugin never fails a test because an
// artifact could not be written; the handler should not fail or skip tests.
func WithErrorHandler(handler func(error)) ConfigOption {
	return func(c *Config) { c.ErrorHandler = handler }
}
