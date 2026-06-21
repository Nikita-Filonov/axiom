package testtracing

import "github.com/Nikita-Filonov/axiom"

func Plugin(trace *Trace) axiom.Plugin {
	return func(cfg *axiom.Config) {
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
