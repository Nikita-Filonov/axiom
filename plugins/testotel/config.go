package testotel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Option configures the OpenTelemetry plugin.
type Option func(*Config)

// Config selects the tracer provider. The global provider is used by default.
type Config struct {
	TracerProvider trace.TracerProvider
}

func newConfig(options ...Option) Config {
	c := Config{TracerProvider: otel.GetTracerProvider()}
	for _, option := range options {
		if option == nil {
			panic("testotel: nil option")
		}
		option(&c)
	}
	if c.TracerProvider == nil {
		panic("testotel: nil tracer provider")
	}
	return c
}

// WithTracerProvider uses provider instead of the global tracer provider.
// The caller owns its SDK lifecycle, including flushing and shutdown.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return func(c *Config) { c.TracerProvider = provider }
}
