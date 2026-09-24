package testquarantine

import (
	"github.com/Nikita-Filonov/axiom"
)

// Plugin applies quarantine settings to selected cases.
func Plugin(options ...ConfigOption) axiom.Plugin {
	cfg := NewConfig(options...)

	return func(e *axiom.Config) {
		reason, quarantined := cfg.decide(e)
		if !quarantined || cfg.Run {
			return
		}

		if reason == "" {
			reason = DefaultReason
		}

		e.Skip = e.Skip.Join(axiom.NewSkip(axiom.SkipBecause("quarantined: " + reason)))
	}
}
