package teststats

import "github.com/Nikita-Filonov/axiom"

// Plugin records an [Attempt] in stats for every attempt that starts or is
// skipped by policy. The attempt is recorded when its testing.T finishes:
// after hooks, fixture cleanup, child subtests, and t.Cleanup callbacks.
// Installing the same stats more than once on a Config, for example on both
// the Runner and the Case, records each attempt once. Plugin panics if stats
// is nil.
func Plugin(stats *Stats) axiom.Plugin {
	if stats == nil {
		panic("teststats: nil stats")
	}

	return func(cfg *axiom.Config) {
		if !markInstalled(cfg, stats) {
			return
		}
		r := &recorder{cfg: cfg, stats: stats}
		cfg.Runtime.EmitEventSink(r.observe)
	}
}
