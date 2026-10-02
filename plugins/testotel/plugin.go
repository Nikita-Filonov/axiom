package testotel

import "github.com/Nikita-Filonov/axiom"

const instrumentationName = "github.com/Nikita-Filonov/axiom/plugins/testotel"

// Plugin records one span for each case attempt or policy skip. It only emits
// Axiom lifecycle event types and names; event messages, logs, assertion text,
// and artefact content are never exported. Applying it twice to one Config
// does not create duplicate spans. A nil option or provider panics.
func Plugin(options ...Option) axiom.Plugin {
	c := newConfig(options...)
	tracer := c.TracerProvider.Tracer(instrumentationName)

	return func(cfg *axiom.Config) {
		if cfg == nil {
			panic("testotel: nil config")
		}
		a, installed := installAttempt(cfg, tracer)
		if !installed {
			return
		}
		cfg.Runtime.EmitEventSink(a.observe)
	}
}
