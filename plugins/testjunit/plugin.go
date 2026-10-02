package testjunit

import "github.com/Nikita-Filonov/axiom"

// Plugin records each executed or policy-skipped attempt in r. Installing it
// more than once for the same Reporter and Config records the attempt once;
// the first installation's options apply. Options are local to this plugin,
// so different runners can share a Reporter and use different suite names.
// A nil Reporter, Config, or option panics.
func Plugin(r *Reporter, options ...Option) axiom.Plugin {
	if r == nil {
		panic("testjunit: nil reporter")
	}
	c := newConfig(options...)
	return func(cfg *axiom.Config) {
		if cfg == nil {
			panic("testjunit: nil config")
		}
		if !markInstalled(cfg, r) {
			return
		}
		current := &recorder{cfg: cfg, reporter: r, config: c}
		cfg.Runtime.EmitEventSink(current.observe)
	}
}
