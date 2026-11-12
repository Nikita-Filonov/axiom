package teststats

import "github.com/Nikita-Filonov/axiom"

// Plugin records an [Attempt] for every attempt that starts or is
// skipped by policy. The attempt is recorded when its testing.T finishes:
// after hooks, fixture cleanup, child subtests, and t.Cleanup callbacks.
// WithStats supplies a queryable collector; otherwise Plugin creates its own.
// WithOutputDir additionally saves incremental JSON artifacts. Reinstalling
// a collector or output directory on a Config records each attempt once.
func Plugin(options ...ConfigOption) axiom.Plugin {
	settings := newConfig(options...)
	output := newOutputWriter(settings.OutputDir, settings.ErrorHandler)

	return func(cfg *axiom.Config) {
		r := installRecorder(cfg)
		r.addCollector(settings.Stats)
		r.addOutput(output)
	}
}
