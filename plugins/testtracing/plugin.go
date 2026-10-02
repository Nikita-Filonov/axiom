package testtracing

import "github.com/Nikita-Filonov/axiom"

// Plugin records raw config-scoped events in trace. Installing it more than
// once for the same Trace and Config, including on both Runner and Case,
// registers one collector. Different traces collect independently; different
// Config values, including retry attempts, get separate records.
// A nil Trace or Config panics. Repeated events are preserved as emitted.
func Plugin(trace *Trace) axiom.Plugin {
	if trace == nil {
		panic("testtracing: nil trace")
	}
	return func(cfg *axiom.Config) {
		if cfg == nil {
			panic("testtracing: nil config")
		}
		if !markInstalled(cfg, trace) {
			return
		}
		sink := newActiveSink(trace, cfg)

		cfg.Runtime.EmitEventSink(func(event axiom.Event) { sink.Append(event) })

		cfg.Runtime.EmitTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				registerSinkCleanup(c, sink)
				next(c)
			}
		})
	}
}

func registerSinkCleanup(cfg *axiom.Config, sink *activeSink) {
	t := cfg.T()
	if t == nil {
		return
	}

	// Runtime sinks are append-only, so cleanup disables this collector after the attempt.
	t.Cleanup(sink.Close)
}
