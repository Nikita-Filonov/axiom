package testjunit

import "github.com/Nikita-Filonov/axiom"

// Plugin records each executed or policy-skipped attempt. Installing it
// more than once for the same Reporter and Config records the attempt once;
// the first installation's suite name applies. Different runners can share a
// Reporter and use different suite names. Output files belong to the Reporter.
// WithReporter supplies a shared collector; otherwise Plugin creates its own.
// WithOutputFile publishes during cleanup and after final status or timing corrections.
// Constructing a plugin does not modify its Reporter. Outputs are registered
// when the plugin is installed on a Config, even if its recorder already exists.
// A nil Config or option panics.
func Plugin(options ...Option) axiom.Plugin {
	settings := newConfig(options...)
	output := newOutputWriter(settings)

	return func(cfg *axiom.Config) {
		if cfg == nil {
			panic("testjunit: nil config")
		}
		reporter := settings.Reporter
		reporter.addOutput(output)
		if !markInstalled(cfg, reporter) {
			return
		}
		current := &recorder{
			cfg:       cfg,
			reporter:  reporter,
			suiteName: settings.SuiteName,
		}
		cfg.Runtime.EmitEventSink(current.observe)
		cfg.Runtime.EmitAttemptSink(current.reconcile)
	}
}
