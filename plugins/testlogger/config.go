package testlogger

import (
	"log/slog"
	"os"
)

// Config selects the logger used by the plugin.
type Config struct {
	Logger *slog.Logger
}

// Option configures the logging plugin.
type Option func(*Config)

func newConfig(options ...Option) Config {
	c := Config{Logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))}
	for _, option := range options {
		if option == nil {
			panic("testlogger: nil option")
		}
		option(&c)
	}
	if c.Logger == nil {
		panic("testlogger: nil logger")
	}
	return c
}

// WithLogger uses logger for log output, filtering, formatting, and attributes.
// The caller owns its handler and output, including flushing and closing them.
// The handler must support concurrent calls. Passing nil is rejected by Plugin.
func WithLogger(logger *slog.Logger) Option {
	return func(c *Config) { c.Logger = logger }
}
